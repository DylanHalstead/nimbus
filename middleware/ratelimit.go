package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/DylanHalstead/nimbus"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	rate      int
	capacity  int
	cleanup   time.Duration
	done      chan struct{} // Channel to signal cleanup goroutine to stop
	closeOnce sync.Once     // Ensures Close() can only be called once
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: requests per second
// capacity: maximum burst size
func NewRateLimiter(rate, capacity int) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		capacity: capacity,
		cleanup:  time.Minute * 5,
		done:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Close stops the cleanup goroutine and releases resources
// Can be called multiple times safely (only closes once)
// Should be called when the rate limiter is no longer needed
func (rl *RateLimiter) Close() {
	rl.closeOnce.Do(func() {
		close(rl.done)
		unregisterLimiter(rl)
	})
}

// cleanupLoop periodically removes old buckets
// Stops when Close() is called on the RateLimiter
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Perform cleanup
			rl.mu.Lock()
			now := time.Now()
			for key, b := range rl.buckets {
				if now.Sub(b.lastSeen) > rl.cleanup {
					delete(rl.buckets, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.done:
			// Stop cleanup loop
			return
		}
	}
}

// allow checks if a request should be allowed
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]

	if !exists {
		b = &bucket{
			tokens:   rl.capacity - 1,
			lastSeen: now,
		}
		rl.buckets[key] = b
		return true
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(b.lastSeen)
	refill := int(elapsed.Seconds() * float64(rl.rate))
	b.tokens = min(rl.capacity, b.tokens+refill)
	b.lastSeen = now

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// rateLimiterRegistry keeps track of all created rate limiters for cleanup
var (
	rateLimiterRegistry = make(map[*RateLimiter]bool)
	registryMu          sync.Mutex
)

// registerLimiter adds a rate limiter to the registry
func registerLimiter(rl *RateLimiter) {
	registryMu.Lock()
	rateLimiterRegistry[rl] = true
	registryMu.Unlock()
}

// unregisterLimiter removes a rate limiter from the registry
func unregisterLimiter(rl *RateLimiter) {
	registryMu.Lock()
	delete(rateLimiterRegistry, rl)
	registryMu.Unlock()
}

// ShutdownAllRateLimiters stops all rate limiter cleanup goroutines
// Call this when shutting down your application
func ShutdownAllRateLimiters() {
	registryMu.Lock()
	limiters := make([]*RateLimiter, 0, len(rateLimiterRegistry))
	for rl := range rateLimiterRegistry {
		limiters = append(limiters, rl)
	}
	registryMu.Unlock()

	for _, rl := range limiters {
		rl.Close()
	}
}

// RateLimitWithRouter returns a rate limiting middleware and registers cleanup with the router.
// Limits requests per IP address.
// The rate limiter's cleanup goroutine will be automatically stopped when router.Shutdown() is called.
// This is the recommended way to use rate limiting.
func RateLimitWithRouter(router interface{ RegisterCleanup(func()) }, requestsPerSecond, burst int) nimbus.Middleware {
	limiter := NewRateLimiter(requestsPerSecond, burst)
	router.RegisterCleanup(limiter.Close)

	return func(next nimbus.Handler) nimbus.Handler {
		return func(ctx *nimbus.Context) (any, int, error) {
			// Use IP address as key
			key := ctx.Request.RemoteAddr

			if !limiter.allow(key) {
				return nil, http.StatusTooManyRequests, nimbus.NewAPIError("rate_limit_exceeded", "Too many requests, please try again later")
			}

			return next(ctx)
		}
	}
}

// RateLimit returns a rate limiting middleware
// Limits requests per IP address
// DEPRECATED: Use RateLimitWithRouter instead for automatic cleanup.
// Note: The rate limiter's cleanup goroutine will run until the application exits
// or ShutdownAllRateLimiters() is called
func RateLimit(requestsPerSecond, burst int) nimbus.Middleware {
	limiter := NewRateLimiter(requestsPerSecond, burst)
	registerLimiter(limiter)

	return func(next nimbus.Handler) nimbus.Handler {
		return func(ctx *nimbus.Context) (any, int, error) {
			// Use IP address as key
			key := ctx.Request.RemoteAddr

			if !limiter.allow(key) {
				return nil, http.StatusTooManyRequests, nimbus.NewAPIError("rate_limit_exceeded", "Too many requests, please try again later")
			}

			return next(ctx)
		}
	}
}

// RateLimitByHeaderWithRouter returns a rate limiting middleware based on a header value
// and registers cleanup with the router.
// Useful for API key based rate limiting.
// The rate limiter's cleanup goroutine will be automatically stopped when router.Shutdown() is called.
// This is the recommended way to use rate limiting.
func RateLimitByHeaderWithRouter(router interface{ RegisterCleanup(func()) }, header string, requestsPerSecond, burst int) nimbus.Middleware {
	limiter := NewRateLimiter(requestsPerSecond, burst)
	router.RegisterCleanup(limiter.Close)

	return func(next nimbus.Handler) nimbus.Handler {
		return func(ctx *nimbus.Context) (any, int, error) {
			key := ctx.GetHeader(header)
			if key == "" {
				key = ctx.Request.RemoteAddr
			}

			if !limiter.allow(key) {
				return nil, http.StatusTooManyRequests, nimbus.NewAPIError("rate_limit_exceeded", "Too many requests, please try again later")
			}

			return next(ctx)
		}
	}
}

// RateLimitByHeader returns a rate limiting middleware based on a header value
// Useful for API key based rate limiting
// DEPRECATED: Use RateLimitByHeaderWithRouter instead for automatic cleanup.
// Note: The rate limiter's cleanup goroutine will run until the application exits
// or ShutdownAllRateLimiters() is called
func RateLimitByHeader(header string, requestsPerSecond, burst int) nimbus.Middleware {
	limiter := NewRateLimiter(requestsPerSecond, burst)
	registerLimiter(limiter)

	return func(next nimbus.Handler) nimbus.Handler {
		return func(ctx *nimbus.Context) (any, int, error) {
			key := ctx.GetHeader(header)
			if key == "" {
				key = ctx.Request.RemoteAddr
			}

			if !limiter.allow(key) {
				return nil, http.StatusTooManyRequests, nimbus.NewAPIError("rate_limit_exceeded", "Too many requests, please try again later")
			}

			return next(ctx)
		}
	}
}
