package middlewares

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestFallbackLimiterEnforcesLimit(t *testing.T) {
	limiter := newFallbackLimiter()
	now := time.Now()

	for i := 0; i < 5; i++ {
		if !limiter.allow("192.0.2.1", 5, time.Minute, now) {
			t.Fatalf("request %d unexpectedly rejected", i+1)
		}
	}

	if limiter.allow("192.0.2.1", 5, time.Minute, now) {
		t.Fatal("request over the limit was allowed")
	}
}

func TestFallbackLimiterResetsAfterWindow(t *testing.T) {
	limiter := newFallbackLimiter()
	now := time.Now()

	if !limiter.allow("192.0.2.2", 1, time.Minute, now) {
		t.Fatal("first request unexpectedly rejected")
	}
	if limiter.allow("192.0.2.2", 1, time.Minute, now) {
		t.Fatal("second request unexpectedly allowed")
	}
	if !limiter.allow("192.0.2.2", 1, time.Minute, now.Add(time.Minute)) {
		t.Fatal("request after window reset was rejected")
	}
}

func TestFallbackLimiterSeparatesClients(t *testing.T) {
	limiter := newFallbackLimiter()
	now := time.Now()

	if !limiter.allow("192.0.2.3", 1, time.Minute, now) {
		t.Fatal("first client unexpectedly rejected")
	}
	if !limiter.allow("192.0.2.4", 1, time.Minute, now) {
		t.Fatal("second client unexpectedly rejected")
	}
}

func TestFallbackLimiterBoundsMemory(t *testing.T) {
	limiter := newFallbackLimiter()
	now := time.Now()

	for i := 0; i < maxFallbackEntries; i++ {
		ip := fmt.Sprintf("client-%d", i)
		if !limiter.allow(ip, 1, time.Hour, now) {
			t.Fatalf("client %d unexpectedly rejected before capacity", i)
		}
	}

	if limiter.allow("new-client", 1, time.Hour, now) {
		t.Fatal("new client should be rejected when the fallback map is full")
	}

	if len(limiter.entries) > maxFallbackEntries {
		t.Fatalf("entry count %d exceeds limit %d", len(limiter.entries), maxFallbackEntries)
	}
}

func TestFallbackLimiterConcurrentRequests(t *testing.T) {
	limiter := newFallbackLimiter()
	now := time.Now()

	const (
		goroutines = 100
		limit      = 10
	)

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.allow("192.0.2.5", limit, time.Minute, now) {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if allowed != limit {
		t.Fatalf("allowed %d requests; want exactly %d", allowed, limit)
	}
}
