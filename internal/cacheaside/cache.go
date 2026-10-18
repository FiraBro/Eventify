// Package cacheaside implements bounded, best-effort Redis cache-aside reads.
// Redis is an optimization; callers' loaders remain the source of truth.
package cacheaside

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// Store is the small Redis surface needed by cache-aside reads.
type Store interface {
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string, time.Duration) error
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

const (
	// CacheFillTimeout bounds a fill even when its initiating HTTP caller disconnects.
	CacheFillTimeout = 10 * time.Second
	AvailabilityTTL  = 15 * time.Second
	ServicesTTL      = 2 * time.Minute
	SettingsTTL      = 1 * time.Minute
)

// Load returns a valid cached value, or invokes load on a miss. Redis and JSON
// failures are fail-open. Cache fills are coalesced only within this process.
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
		// Redis outage: avoid adding singleflight/cache latency to source fallback.
		return load(ctx)
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
