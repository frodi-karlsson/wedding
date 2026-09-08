package ratelimit

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func newTestLimiter(limit int, window time.Duration) (*Limiter, func(time.Duration)) {
	now := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	advance := func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(d)
	}
	return newWithClock(limit, window, clock), advance
}

func TestAllow_PermitsUpToTheLimitThenDenies(t *testing.T) {
	limiter, _ := newTestLimiter(3, time.Minute)

	for i := range 3 {
		if !limiter.Allow("1.2.3.4") {
			t.Fatalf("Allow() denied request %d, want the first 3 allowed", i+1)
		}
	}

	if limiter.Allow("1.2.3.4") {
		t.Error("Allow() permitted request 4, want it denied")
	}
}

func TestAllow_KeepsDenyingWhileTheWindowIsFull(t *testing.T) {
	limiter, advance := newTestLimiter(2, time.Minute)

	limiter.Allow("1.2.3.4")
	limiter.Allow("1.2.3.4")
	advance(30 * time.Second)

	if limiter.Allow("1.2.3.4") {
		t.Error("Allow() permitted a request half a window later, want it denied")
	}
}

func TestAllow_TracksKeysIndependently(t *testing.T) {
	limiter, _ := newTestLimiter(1, time.Minute)

	limiter.Allow("1.2.3.4")

	if limiter.Allow("1.2.3.4") {
		t.Error("Allow() permitted a second request from an exhausted key")
	}
	if !limiter.Allow("5.6.7.8") {
		t.Error("Allow() denied a different key, want per-key budgets")
	}
}

func TestAllow_ForgetsHitsOnceTheWindowPasses(t *testing.T) {
	limiter, advance := newTestLimiter(2, time.Minute)

	limiter.Allow("1.2.3.4")
	limiter.Allow("1.2.3.4")
	advance(time.Minute + time.Second)

	if !limiter.Allow("1.2.3.4") {
		t.Error("Allow() denied a request after the window elapsed, want the budget restored")
	}
}

// The window slides rather than resetting wholesale: hits age out one at a time,
// so a caller who spends its budget gradually regains it gradually.
func TestAllow_ExpiresHitsIndividuallyRatherThanInBatches(t *testing.T) {
	limiter, advance := newTestLimiter(2, time.Minute)

	limiter.Allow("1.2.3.4")
	advance(40 * time.Second)
	limiter.Allow("1.2.3.4")
	advance(25 * time.Second) // 65s after hit one, 25s after hit two

	if !limiter.Allow("1.2.3.4") {
		t.Error("Allow() denied a request after the first hit aged out")
	}
	if limiter.Allow("1.2.3.4") {
		t.Error("Allow() permitted a request while the second hit was still inside the window")
	}
}

func TestAllow_BoundsTheNumberOfTrackedKeys(t *testing.T) {
	limiter, _ := newTestLimiter(1, time.Minute)

	for i := range maxTrackedKeys * 2 {
		limiter.Allow(strconv.Itoa(i))
	}

	limiter.mu.Lock()
	tracked := len(limiter.hits)
	limiter.mu.Unlock()
	if tracked > maxTrackedKeys {
		t.Errorf("limiter tracks %d keys, want at most %d", tracked, maxTrackedKeys)
	}
}

func TestAllow_ReclaimsKeysWhoseHitsHaveExpired(t *testing.T) {
	limiter, advance := newTestLimiter(1, time.Minute)

	for i := range maxTrackedKeys {
		limiter.Allow(strconv.Itoa(i))
	}
	advance(time.Minute + time.Second)
	limiter.Allow("fresh")

	limiter.mu.Lock()
	tracked := len(limiter.hits)
	limiter.mu.Unlock()
	if tracked != 1 {
		t.Errorf("limiter tracks %d keys, want 1 after the rest expired", tracked)
	}
}

func TestAllow_GrantsExactlyTheLimitUnderConcurrency(t *testing.T) {
	const limit = 20
	limiter, _ := newTestLimiter(limit, time.Minute)

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow("1.2.3.4") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != limit {
		t.Errorf("Allow() granted %d of 200 concurrent requests, want exactly %d", allowed, limit)
	}
}
