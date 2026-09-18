package httpapi

import (
	"sync"
	"time"
)

// RateLimiter is a fixed-window counter per key (IP, email, ...). It bounds
// brute-force login/register attempts; it is not a general traffic shaper.
// Backed by an in-memory map: limits reset on restart, which is acceptable
// for a single-container backend (documented; move to Redis/DB if scaled out).
type RateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
	now    func() time.Time
}

// NewRateLimiter allows limit events per window for each distinct key.
func NewRateLimiter(limit int, window time.Duration, now func() time.Time) *RateLimiter {
	if now == nil {
		now = time.Now
	}
	return &RateLimiter{hits: map[string][]time.Time{}, limit: limit, window: window, now: now}
}

// Allow reports whether an event for key fits the budget, recording it when it
// does. Expired entries are pruned on each call so the map cannot grow
// unboundedly while under attack (bounded by distinct keys in one window).
func (l *RateLimiter) Allow(key string) bool {
	now := l.now()
	cutoff := now.Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}
