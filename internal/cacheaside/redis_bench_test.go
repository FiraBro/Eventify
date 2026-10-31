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

var redisBenchmarkResult testValue

func BenchmarkRedisLoadCacheHit(b *testing.B) {
	benchmarkRedisCacheHit(b, false)
}

func BenchmarkRedisLoadVersionedCacheHit(b *testing.B) {
	benchmarkRedisCacheHit(b, true)
}

func benchmarkRedisCacheHit(b *testing.B, versioned bool) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		b.Skip("REDIS_URL is not set")
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		opts = &redis.Options{Addr: url}
	}

	client := redis.NewClient(opts)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		b.Fatalf("connect to benchmark Redis: %v", err)
	}

	store := RedisStore{Client: client}
	group := &singleflight.Group{}
	namespace := fmt.Sprintf("local-go:benchmark:%d", time.Now().UnixNano())
	key := namespace + ":value"
	ttl := time.Minute

	if versioned {
		token := "benchmark-generation"
		if err := store.Set(ctx, namespace+":generation", token, 0); err != nil {
			b.Fatal(err)
		}
		key = namespace + ":g:" + token + ":value"
	}

	value := testValue{Value: "benchmark-service-value"}
	if err := store.Set(ctx, key, `{"value":"benchmark-service-value"}`, ttl); err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cleanupCancel()
		_ = client.Del(cleanupCtx, key, namespace+":generation").Err()
	})

	loader := func(context.Context) (testValue, error) {
		return testValue{}, fmt.Errorf("loader unexpectedly called during cache-hit benchmark")
	}

	benchmarkCtx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var got testValue

		if versioned {
			got, err = LoadVersioned(
				benchmarkCtx, store, group, namespace, "value", ttl, loader,
			)
		} else {
			got, err = Load(benchmarkCtx, store, group, key, ttl, loader)
		}

		if err != nil {
			b.Fatal(err)
		}

		redisBenchmarkResult = got
	}

	_ = value
}
