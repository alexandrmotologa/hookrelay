package queue

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// TokenBucket implements a simple token bucket algorithm per endpoint.
type TokenBucket struct {
	rate       float64 // tokens per second
	capacity   float64 // max tokens
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

func NewTokenBucket(rate float64) *TokenBucket {
	if rate <= 0 {
		rate = 100 // default 100 req/s
	}
	return &TokenBucket{
		rate:       rate,
		capacity:   rate,
		tokens:     rate,
		lastUpdate: time.Now(),
	}
}

func (tb *TokenBucket) Take(ctx context.Context) error {
	for {
		tb.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(tb.lastUpdate).Seconds()
		tb.lastUpdate = now

		tb.tokens += elapsed * tb.rate
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}

		if tb.tokens >= 1.0 {
			tb.tokens -= 1.0
			tb.mu.Unlock()
			return nil
		}

		missing := 1.0 - tb.tokens
		waitTime := time.Duration((missing / tb.rate) * float64(time.Second))
		tb.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}
}

// EndpointRateLimiter tracks token buckets for each destination endpoint.
type EndpointRateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
}

func NewEndpointRateLimiter() *EndpointRateLimiter {
	return &EndpointRateLimiter{
		buckets: make(map[string]*TokenBucket),
	}
}

func (rl *EndpointRateLimiter) Wait(ctx context.Context, endpointID string, rateLimit int) error {
	if rateLimit <= 0 {
		return nil
	}

	rl.mu.RLock()
	tb, exists := rl.buckets[endpointID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		tb, exists = rl.buckets[endpointID]
		if !exists {
			tb = NewTokenBucket(float64(rateLimit))
			rl.buckets[endpointID] = tb
		}
		rl.mu.Unlock()
	}

	return tb.Take(ctx)
}

// ParseRetryAfter parses the HTTP Retry-After header into a time.Duration.
// Supports both integer seconds ("120") and HTTP-date ("Wed, 21 Oct 2026 07:28:00 GMT").
func ParseRetryAfter(headerValue string) (time.Duration, bool) {
	if headerValue == "" {
		return 0, false
	}

	// 1. Try parsing integer seconds
	if secs, err := strconv.Atoi(headerValue); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second, true
	}

	// 2. Try parsing HTTP Date (RFC1123 or RFC850)
	if t, err := http.ParseTime(headerValue); err == nil {
		diff := time.Until(t)
		if diff > 0 {
			return diff, true
		}
	}

	return 0, false
}
