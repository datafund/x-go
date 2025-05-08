package twitter

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu            sync.Mutex
	lastCallTime  time.Time
	endpointCalls map[string]*endpointLimit
}

type endpointLimit struct {
	mu           sync.Mutex
	calls        int
	windowStart  time.Time
	windowLength time.Duration
	maxCalls     int
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		lastCallTime:  time.Now(),
		endpointCalls: make(map[string]*endpointLimit),
	}
}

func (r *rateLimiter) waitForGlobalLimit() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure 1.5 seconds between requests
	elapsed := time.Since(r.lastCallTime)
	if elapsed < 1500*time.Millisecond {
		time.Sleep(1500*time.Millisecond - elapsed)
	}
	r.lastCallTime = time.Now()
}

func (r *rateLimiter) checkEndpointLimit(endpoint string) bool {
	limit, exists := r.endpointCalls[endpoint]
	if !exists {
		// Default to 150 requests per 15 minutes
		limit = &endpointLimit{
			windowLength: 15 * time.Minute,
			maxCalls:     150,
			windowStart:  time.Now(),
		}
		r.endpointCalls[endpoint] = limit
	}

	limit.mu.Lock()
	defer limit.mu.Unlock()

	// Reset window if needed
	if time.Since(limit.windowStart) > limit.windowLength {
		limit.calls = 0
		limit.windowStart = time.Now()
	}

	// Check if we can make another call
	if limit.calls >= limit.maxCalls {
		return false
	}

	limit.calls++
	return true
}

func (r *rateLimiter) waitForEndpoint(endpoint string) {
	for !r.checkEndpointLimit(endpoint) {
		time.Sleep(time.Second)
	}
	r.waitForGlobalLimit()
}
