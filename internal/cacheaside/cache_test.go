package cacheaside

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type fakeStore struct {
	mu     sync.Mutex
	values map[string]string
	getErr error
	setErr error
	sets   int
}

func newFakeStore() *fakeStore { return &fakeStore{values: map[string]string{}} }
func (f *fakeStore) Get(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return "", f.getErr
	}
	v, ok := f.values[key]
	if !ok {
		return "", redis.Nil
	}
	return v, nil
}
func (f *fakeStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sets++
	if f.setErr != nil {
		return f.setErr
	}
	f.values[key] = value
	return nil
}
func (f *fakeStore) put(key, value string) { f.mu.Lock(); defer f.mu.Unlock(); f.values[key] = value }

func TestLoadCacheHitAvoidsLoader(t *testing.T) {
	store := newFakeStore()
	store.put("k", `{"value":"cached"}`)
	var calls atomic.Int32
	got, err := Load(context.Background(), store, &singleflight.Group{}, "k", time.Minute, func(context.Context) (testValue, error) { calls.Add(1); return testValue{}, nil })
	if err != nil || got.Value != "cached" || calls.Load() != 0 {
		t.Fatalf("got=%+v err=%v loader calls=%d", got, err, calls.Load())
	}
}

type testValue struct {
	Value string `json:"value"`
}

func TestLoadMissLoadsAndPopulates(t *testing.T) {
	store := newFakeStore()
	var calls atomic.Int32
	got, err := Load(context.Background(), store, &singleflight.Group{}, "k", time.Minute, func(context.Context) (testValue, error) { calls.Add(1); return testValue{Value: "source"}, nil })
	if err != nil || got.Value != "source" || calls.Load() != 1 {
		t.Fatalf("got=%+v err=%v calls=%d", got, err, calls.Load())
	}
	if raw, err := store.Get(context.Background(), "k"); err != nil || raw != `{"value":"source"}` {
		t.Fatalf("cache=%q err=%v", raw, err)
	}
}

func TestLoadRedisReadFailureFallsBack(t *testing.T) {
	store := newFakeStore()
	store.getErr = errors.New("redis down")
	var calls atomic.Int32
	got, err := Load(context.Background(), store, &singleflight.Group{}, "k", time.Minute, func(context.Context) (testValue, error) { calls.Add(1); return testValue{Value: "source"}, nil })
	if err != nil || got.Value != "source" || calls.Load() != 1 {
		t.Fatalf("got=%+v err=%v calls=%d", got, err, calls.Load())
	}
}

func TestLoadRedisWriteFailureDoesNotFailRead(t *testing.T) {
	store := newFakeStore()
	store.setErr = errors.New("redis write failed")
	got, err := Load(context.Background(), store, &singleflight.Group{}, "k", time.Minute, func(context.Context) (testValue, error) { return testValue{Value: "source"}, nil })
	if err != nil || got.Value != "source" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestLoadInvalidCachedJSONReloadsSource(t *testing.T) {
	store := newFakeStore()
	store.put("k", `not-json`)
	var calls atomic.Int32
	got, err := Load(context.Background(), store, &singleflight.Group{}, "k", time.Minute, func(context.Context) (testValue, error) { calls.Add(1); return testValue{Value: "fresh"}, nil })
	if err != nil || got.Value != "fresh" || calls.Load() != 1 {
		t.Fatalf("got=%+v err=%v calls=%d", got, err, calls.Load())
	}
}

func TestLoadCoalescesConcurrentMisses(t *testing.T) {
	store := newFakeStore()
	var calls atomic.Int32
	group := &singleflight.Group{}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := Load(context.Background(), store, group, "same", time.Minute, func(context.Context) (testValue, error) {
				calls.Add(1)
				time.Sleep(40 * time.Millisecond)
				return testValue{Value: "one"}, nil
			})
			if err != nil {
				t.Errorf("Load: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("expected one source load, got %d", calls.Load())
	}
}

func TestLoadDifferentKeysProceedIndependently(t *testing.T) {
	store := newFakeStore()
	group := &singleflight.Group{}
	entered := make(chan string, 2)
	release := make(chan struct{})
	done := make(chan struct{}, 2)
	for _, key := range []string{"a", "b"} {
		key := key
		go func() {
			_, _ = Load(context.Background(), store, group, key, time.Minute, func(context.Context) (testValue, error) { entered <- key; <-release; return testValue{Value: key}, nil })
			done <- struct{}{}
		}()
	}
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case key := <-entered:
			seen[key] = true
		case <-time.After(time.Second):
			t.Fatal("different keys blocked each other")
		}
	}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("loaders not both entered: %v", seen)
	}
	close(release)
	<-done
	<-done
}

func TestLoadCanceledWaiterDoesNotCancelSharedFill(t *testing.T) {
	store := newFakeStore()
	group := &singleflight.Group{}
	entered := make(chan struct{})
	release := make(chan struct{})
	firstCtx, cancel := context.WithCancel(context.Background())
	firstDone := make(chan error, 1)
	go func() {
		_, err := Load(firstCtx, store, group, "k", time.Minute, func(context.Context) (testValue, error) {
			close(entered)
			<-release
			return testValue{Value: "ok"}, nil
		})
		firstDone <- err
	}()
	<-entered
	secondDone := make(chan error, 1)
	go func() {
		_, err := Load(context.Background(), store, group, "k", time.Minute, func(context.Context) (testValue, error) { return testValue{Value: "duplicate"}, nil })
		secondDone <- err
	}()
	cancel()
	if err := <-firstDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("first caller err=%v", err)
	}
	close(release)
	if err := <-secondDone; err != nil {
		t.Fatalf("second caller err=%v", err)
	}
}

func TestLoadIncompatibleCachedShapeReloadsSource(t *testing.T) {
	store := newFakeStore()
	store.put("k", `{}`) // syntactically valid, but missing the model's value field
	var calls atomic.Int32
	got, err := Load(context.Background(), store, &singleflight.Group{}, "k", time.Minute, func(context.Context) (testValue, error) {
		calls.Add(1)
		return testValue{Value: "fresh"}, nil
	})
	if err != nil || got.Value != "fresh" || calls.Load() != 1 {
		t.Fatalf("got=%+v err=%v calls=%d", got, err, calls.Load())
	}
}
