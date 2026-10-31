// Package cacheaside implements bounded, best-effort Redis cache-aside reads.
// Redis is an optimization; callers' loaders remain the source of truth.
package cacheaside

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// Store is the small Redis surface needed by cache-aside reads.
type Store interface {
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string, time.Duration) error
}

// VersionedStore supports shared generation tokens. A token change makes every
// key from an earlier generation unreachable without scanning or deleting keys.
type VersionedStore interface {
	Store
	SetNX(context.Context, string, string, time.Duration) (bool, error)
}

// RedisStore adapts the application's existing go-redis client.
type RedisStore struct{ Client redis.UniversalClient }

func (s RedisStore) Get(ctx context.Context, key string) (string, error) {
	if s.Client == nil {
		return "", errors.New("redis client is nil")
	}
	return s.Client.Get(ctx, key).Result()
}
func (s RedisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if s.Client == nil {
		return errors.New("redis client is nil")
	}
	return s.Client.Set(ctx, key, value, ttl).Err()
}
func (s RedisStore) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if s.Client == nil {
		return false, errors.New("redis client is nil")
	}
	return s.Client.SetNX(ctx, key, value, ttl).Result()
}

const (
	// CacheFillTimeout bounds a fill even when its initiating HTTP caller disconnects.
	CacheFillTimeout = 10 * time.Second
	AvailabilityTTL  = 15 * time.Second
	ServicesTTL      = 2 * time.Minute
	SettingsTTL      = 1 * time.Minute
)

// LoadVersioned is cache-aside with cross-instance namespace invalidation.
// keySuffix must be a stable, non-sensitive identifier within namespace. A
// generation token is read before the cache lookup; fills are written only to
// that token's key. If a mutation changes the token while a fill is in flight,
// the old fill may finish but no subsequent reader can address its key.
// If Redis is unavailable, source loading falls back to process-local singleflight.
func LoadVersioned[T any](ctx context.Context, store Store, group *singleflight.Group, namespace, keySuffix string, ttl time.Duration, load func(context.Context) (T, error)) (T, error) {
	var zero T
	versioned, ok := store.(VersionedStore)
	if store == nil || group == nil || !ok {
		return load(ctx)
	}
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	token, err := generationToken(ctx, versioned, namespace)
	if err != nil {
		// Redis outage: keep the source fallback coalesced per resource variant
		// so a cache failure does not fan out into a database load storm.
		return loadCoalesced(ctx, group, namespace+":fallback:"+keySuffix, load)
	}
	key := namespace + ":g:" + token + ":" + keySuffix
	return Load(ctx, store, group, key, ttl, load)
}

// InvalidateNamespace atomically replaces a shared namespace token. It does not
// scan Redis keys and works across application instances sharing the same Redis.
func InvalidateNamespace(ctx context.Context, store Store, namespace string) error {
	versioned, ok := store.(VersionedStore)
	if store == nil || !ok {
		return errors.New("cache store does not support namespace invalidation")
	}
	token, err := newToken()
	if err != nil {
		return err
	}
	return versioned.Set(ctx, namespace+":generation", token, 0)
}

func generationToken(ctx context.Context, store VersionedStore, namespace string) (string, error) {
	key := namespace + ":generation"
	token, err := store.Get(ctx, key)
	if err == nil && token != "" {
		return token, nil
	}
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", err
	}
	candidate, err := newToken()
	if err != nil {
		return "", err
	}
	created, err := store.SetNX(ctx, key, candidate, 0)
	if err != nil {
		return "", err
	}
	if created {
		return candidate, nil
	}
	// Another reader or an invalidation won the initialization race.
	token, err = store.Get(ctx, key)
	if err != nil || token == "" {
		if err == nil {
			err = errors.New("empty cache generation token")
		}
		return "", err
	}
	return token, nil
}

func newToken() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

// Mutate runs a database mutation and invalidates namespaces only after the
// mutation succeeds. Repositories using transactions must return only after
// Commit succeeds. Cache failures are logged and never replace the mutation result.
func Mutate[T any](ctx context.Context, store Store, namespaces []string, mutate func(context.Context) (T, error)) (T, error) {
	value, err := mutate(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	InvalidateNamespacesBestEffort(store, namespaces...)
	return value, nil
}

// InvalidateNamespacesBestEffort invalidates shared namespaces using a bounded
// context independent of the request. Only static namespace names are logged.
func InvalidateNamespacesBestEffort(store Store, namespaces ...string) {
	if store == nil || len(namespaces) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, namespace := range namespaces {
		if err := InvalidateNamespace(ctx, store, namespace); err != nil {
			slog.Warn("read cache namespace invalidation failed", "namespace", namespace)
		}
	}
}

// Load returns a valid cached value, or invokes load on a miss. Redis and JSON
// failures are fail-open. Cache fills and outage fallbacks are coalesced only within this process.
// The shared fill uses a bounded context detached from the initiating caller so
// one canceled waiter cannot abort work needed by other waiters. Each waiter may
// still stop waiting immediately when its own context is canceled.
func Load[T any](ctx context.Context, store Store, group *singleflight.Group, key string, ttl time.Duration, load func(context.Context) (T, error)) (T, error) {
	var zero T
	if store == nil || group == nil {
		return load(ctx)
	}
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if value, err := read[T](ctx, store, key); err == nil {
		return value, nil
	} else if !errors.Is(err, redis.Nil) {
		// Redis outage: use the same singleflight key as a cache miss. Concurrent
		// callers therefore share one bounded source load even while Redis is down.
		return loadCoalesced(ctx, group, key, load)
	}

	resultCh := group.DoChan(key, func() (any, error) {
		fillCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), CacheFillTimeout)
		defer cancel()
		if value, err := read[T](fillCtx, store, key); err == nil {
			return value, nil
		}
		value, err := load(fillCtx)
		if err != nil {
			return nil, err
		}
		if encoded, marshalErr := json.Marshal(value); marshalErr == nil {
			_ = store.Set(fillCtx, key, string(encoded), ttl) // cache writes never fail the read
		}
		return value, nil
	})
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return zero, result.Err
		}
		value, ok := result.Val.(T)
		if !ok {
			return load(ctx)
		}
		return value, nil
	}
}

// loadCoalesced shares a bounded source fallback when Redis cannot be read.
// The initiating caller may stop waiting without canceling work needed by other
// callers. It deliberately does not attempt a cache write while Redis is failing.
func loadCoalesced[T any](ctx context.Context, group *singleflight.Group, key string, load func(context.Context) (T, error)) (T, error) {
	var zero T
	if group == nil {
		return load(ctx)
	}
	resultCh := group.DoChan(key, func() (any, error) {
		fillCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), CacheFillTimeout)
		defer cancel()
		return load(fillCtx)
	})
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return zero, result.Err
		}
		value, ok := result.Val.(T)
		if !ok {
			return load(ctx)
		}
		return value, nil
	}
}

func read[T any](ctx context.Context, store Store, key string) (T, error) {
	var value T
	raw, err := store.Get(ctx, key)
	if err != nil {
		return value, err
	}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		// Invalid/incompatible entries are treated as misses and overwritten if possible.
		return value, redis.Nil
	}
	// Entries written by this package are compact json.Marshal output. Requiring
	// a canonical round-trip rejects unknown/missing fields and wrong JSON shapes
	// that encoding/json would otherwise silently accept as zero values.
	canonical, err := json.Marshal(value)
	if err != nil || string(canonical) != raw {
		return value, redis.Nil
	}
	return value, nil
}
