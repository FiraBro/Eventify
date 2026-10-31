package cacheaside

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"golang.org/x/sync/singleflight"
)

var benchmarkResult testValue

func BenchmarkJSONUnmarshalOnly(b *testing.B) {
	raw := []byte(`{"value":"benchmark-service-value"}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var value testValue
		if err := json.Unmarshal(raw, &value); err != nil {
			b.Fatal(err)
		}
		benchmarkResult = value
	}
}

func BenchmarkJSONCanonicalValidation(b *testing.B) {
	raw := `{"value":"benchmark-service-value"}`

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var value testValue
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			b.Fatal(err)
		}

		canonical, err := json.Marshal(value)
		if err != nil {
			b.Fatal(err)
		}
		if string(canonical) != raw {
			b.Fatal("unexpected canonical JSON")
		}

		benchmarkResult = value
	}
}

func BenchmarkLoadCacheHit(b *testing.B) {
	store := newFakeStore()
	store.put("benchmark", `{"value":"benchmark-service-value"}`)
	group := &singleflight.Group{}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		value, err := Load(ctx, store, group, "benchmark", time.Minute,
			func(context.Context) (testValue, error) {
				b.Fatal("loader called on cache hit")
				return testValue{}, nil
			})
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = value
	}
}
