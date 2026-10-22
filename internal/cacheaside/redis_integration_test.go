//go:build integration

package cacheaside

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// TestRedisNamespaceInvalidationAcrossClients verifies shared generation
// invalidation against a real Redis server, not an in-memory fake. Set REDIS_URL
// to a dedicated disposable Redis instance to enable it.
func TestRedisNamespaceInvalidationAcrossClients(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL is not set; real Redis cross-client test skipped")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		// Permit the same host:port form accepted by the application config.
		opts = &redis.Options{Addr: url}
	}
	clientA := redis.NewClient(opts)
	defer clientA.Close()
	clientB := redis.NewClient(opts)
	defer clientB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := clientA.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect to dedicated Redis test instance: %v", err)
	}

	storeA := RedisStore{Client: clientA}
	storeB := RedisStore{Client: clientB}
	namespace := fmt.Sprintf("local-go:c3:test:%d", time.Now().UnixNano())
	groupA := &singleflight.Group{}
	groupB := &singleflight.Group{}

	first, err := LoadVersioned(ctx, storeA, groupA, namespace, "variant", time.Minute, func(context.Context) (testValue, error) {
		return testValue{Value: "before"}, nil
	})
	if err != nil || first.Value != "before" {
		t.Fatalf("initial load got=%+v err=%v", first, err)
	}
	oldToken, err := clientA.Get(ctx, namespace+":generation").Result()
	if err != nil {
		t.Fatal(err)
	}
	if err := InvalidateNamespace(ctx, storeB, namespace); err != nil {
		t.Fatalf("invalidate from second client: %v", err)
	}
	second, err := LoadVersioned(ctx, storeA, groupA, namespace, "variant", time.Minute, func(context.Context) (testValue, error) {
		return testValue{Value: "after"}, nil
	})
	if err != nil || second.Value != "after" {
		t.Fatalf("cross-client load got=%+v err=%v", second, err)
	}
	newToken, err := clientB.Get(ctx, namespace+":generation").Result()
	if err != nil || newToken == oldToken {
		t.Fatalf("generation did not change across clients: old=%q new=%q err=%v", oldToken, newToken, err)
	}

	// Reproduce a fill that starts before invalidation and finishes afterwards.
	raceNS := namespace + ":race"
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := LoadVersioned(context.Background(), storeA, groupA, raceNS, "variant", time.Minute, func(context.Context) (testValue, error) {
			close(entered)
			<-release
			return testValue{Value: "stale-fill"}, nil
		})
		done <- err
	}()
	<-entered
	if err := InvalidateNamespace(ctx, storeB, raceNS); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	fresh, err := LoadVersioned(ctx, storeB, groupB, raceNS, "variant", time.Minute, func(context.Context) (testValue, error) {
		return testValue{Value: "fresh-after-invalidation"}, nil
	})
	if err != nil || fresh.Value != "fresh-after-invalidation" {
		t.Fatalf("stale fill became visible after invalidation: got=%+v err=%v", fresh, err)
	}

	// Delete only the known keys created by this test; no Redis key scan is used.
	for _, ns := range []string{namespace, raceNS} {
		if token, getErr := clientA.Get(ctx, ns+":generation").Result(); getErr == nil {
			_ = clientA.Del(ctx, ns+":g:"+token+":variant").Err()
		}
		_ = clientA.Del(ctx, ns+":generation").Err()
	}
}
