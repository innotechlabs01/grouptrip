package middleware

import (
	"net/http"
	"sync"
	"time"
)

// TokenBucket implements a leaky bucket rate limiter.
type TokenBucket struct {
	capacity   float64
	tokens     float64
	refillRate float64
	last       time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket with the given capacity and refill rate (tokens per second).
func NewTokenBucket(capacity, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		last:       time.Now(),
	}
}

// Allow consumes a token if available. Returns true if allowed.
func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now

	// Refill tokens
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}

	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}
	return false
}

// RateLimiter holds per-client buckets.
type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*TokenBucket
	capacity   float64
	refillRate float64
	window     time.Duration
}

// NewRateLimiter creates a rate limiter with capacity tokens per window.
func NewRateLimiter(capacity int, per time.Duration) *RateLimiter {
	refillRate := float64(capacity) / per.Seconds()
	if refillRate <= 0 {
		refillRate = 1
	}
	return &RateLimiter{
		buckets:    make(map[string]*TokenBucket),
		capacity:   float64(capacity),
		refillRate: refillRate,
		window:     per,
	}
}

// Middleware returns an http.Handler that rate limits requests per client IP.
// Returns 429 Too Many Requests when limit is exceeded.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			key = forwarded
		}

		rl.mu.Lock()
		bucket, ok := rl.buckets[key]
		if !ok {
			bucket = NewTokenBucket(rl.capacity, rl.refillRate)
			rl.buckets[key] = bucket
		}
		rl.mu.Unlock()

		if !bucket.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
