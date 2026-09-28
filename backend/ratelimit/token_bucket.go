package ratelimit

import (
	"sync"
	"time"
)

// TokenBucketLimiter is an in-process placeholder for a Redis-backed token
// bucket. In production this would be Redis so limits are shared across all
// horizontally-scaled API instances behind HAProxy; documented explicitly in
// ARCHITECTURE.md / Built vs Designed. Kept in-process here so the working
// slice runs with zero extra infra dependencies.
type TokenBucketLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity int
	refillPer time.Duration
}

type bucket struct {
	tokens   int
	lastFill time.Time
}

func NewTokenBucketLimiter(capacity int, refillPer time.Duration) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		buckets:   make(map[string]*bucket),
		capacity:  capacity,
		refillPer: refillPer,
	}
}

func (l *TokenBucketLimiter) Allow(scopeKey, subKey string) bool {
	key := scopeKey + "|" + subKey
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	now := time.Now()
	if !ok {
		b = &bucket{tokens: l.capacity - 1, lastFill: now}
		l.buckets[key] = b
		return true
	}

	elapsed := now.Sub(b.lastFill)
	refillTokens := int(elapsed / l.refillPer)
	if refillTokens > 0 {
		b.tokens = min(l.capacity, b.tokens+refillTokens)
		b.lastFill = now
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
