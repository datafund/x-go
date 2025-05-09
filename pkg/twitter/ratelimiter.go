package twitter

import (
	"context"
	"fmt"
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

type RateLimitError struct {
	Endpoint string
	WaitTime time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded for endpoint %s, wait for %v", e.Endpoint, e.WaitTime)
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		lastCallTime:  time.Now(),
		endpointCalls: make(map[string]*endpointLimit),
	}
}

func (r *rateLimiter) waitForGlobalLimit() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	elapsed := time.Since(r.lastCallTime)
	if elapsed < 1500*time.Millisecond {
		waitTime := 1500*time.Millisecond - elapsed
		time.Sleep(waitTime)
	}
	r.lastCallTime = time.Now()
	return nil
}

func (r *rateLimiter) checkEndpointLimit(endpoint string) (bool, time.Duration) {
	r.mu.Lock() // Lock for map access
	limit, exists := r.endpointCalls[endpoint]
	if !exists {
		limit = &endpointLimit{
			windowLength: 15 * time.Minute,
			maxCalls:     100,
			windowStart:  time.Now(),
		}
		r.endpointCalls[endpoint] = limit
	}
	r.mu.Unlock()

	limit.mu.Lock()
	defer limit.mu.Unlock()

	now := time.Now()
	windowElapsed := now.Sub(limit.windowStart)

	if windowElapsed > limit.windowLength {
		limit.calls = 0
		limit.windowStart = now
		return true, 0
	}

	if limit.calls >= limit.maxCalls {
		waitTime := limit.windowLength - windowElapsed
		return false, waitTime
	}

	limit.calls++
	return true, 0
}

func (r *rateLimiter) waitForEndpoint(ctx context.Context, endpoint string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if allowed, waitTime := r.checkEndpointLimit(endpoint); allowed {
				if err := r.waitForGlobalLimit(); err != nil {
					return err
				}
				return nil
			} else {
				if waitTime > 0 {
					timer := time.NewTimer(waitTime)
					select {
					case <-ctx.Done():
						timer.Stop()
						return ctx.Err()
					case <-timer.C:
						continue
					}
				}
			}
		}
	}
}
