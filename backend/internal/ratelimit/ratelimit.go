// Package ratelimit caps how often a single caller may hit an endpoint.
//
// It exists because the public invite routes are addressed by a short,
// guessable id. The id space is large enough that finding a valid invite takes
// hundreds of thousands of tries, which is only a deterrent if those tries
// cannot be spent quickly.
package ratelimit

import (
	"sync"
	"time"
)

// maxTrackedKeys bounds memory so a flood from many addresses cannot grow the
// map without limit. Once full, the key whose activity is oldest is dropped,
// which at worst hands back an unearned budget to a long-idle caller.
const maxTrackedKeys = 1000

// Limiter permits at most limit events per key within a sliding window.
// Individual events age out as the window moves, so a caller that spends its
// budget gradually regains it gradually rather than in one lump at a window
// boundary.
type Limiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	now    func() time.Time
	hits   map[string][]time.Time
}

// New returns a Limiter permitting limit events per key per window.
func New(limit int, window time.Duration) *Limiter {
	return newWithClock(limit, window, time.Now)
}

func newWithClock(limit int, window time.Duration, now func() time.Time) *Limiter {
	return &Limiter{
		limit:  limit,
		window: window,
		now:    now,
		hits:   make(map[string][]time.Time),
	}
}

// Allow records an event for key and reports whether it fits within the budget.
// A denied event is not recorded, so hammering a blocked key does not extend
// the block.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := l.now().Add(-l.window)
	recent := keepAfter(l.hits[key], cutoff)
	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false
	}

	if _, tracked := l.hits[key]; !tracked {
		l.makeRoomFor(cutoff)
	}
	l.hits[key] = append(recent, l.now())
	return true
}

// makeRoomFor frees a map slot for a key that is not yet tracked, first by
// discarding keys with nothing left inside the window and then, if that was not
// enough, by evicting the least recently active one.
func (l *Limiter) makeRoomFor(cutoff time.Time) {
	if len(l.hits) < maxTrackedKeys {
		return
	}
	for key, hits := range l.hits {
		if len(keepAfter(hits, cutoff)) == 0 {
			delete(l.hits, key)
		}
	}
	if len(l.hits) < maxTrackedKeys {
		return
	}

	var oldestKey string
	var oldestSeen time.Time
	for key, hits := range l.hits {
		last := hits[len(hits)-1]
		if oldestKey == "" || last.Before(oldestSeen) {
			oldestKey = key
			oldestSeen = last
		}
	}
	delete(l.hits, oldestKey)
}

// keepAfter returns the hits at or after cutoff. Hits are appended in
// chronological order, so the survivors are always a suffix of the slice.
func keepAfter(hits []time.Time, cutoff time.Time) []time.Time {
	for i, h := range hits {
		if h.After(cutoff) {
			return hits[i:]
		}
	}
	return nil
}
