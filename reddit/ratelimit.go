package reddit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter handles rate limiting for Reddit API requests
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a new rate limiter with the specified rate and burst
func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	// Convert requests per minute to requests per second
	rps := float64(requestsPerMinute) / 60.0
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// Wait blocks until a request can be made according to the rate limit
func (r *RateLimiter) Wait(ctx context.Context) error {
	if err := r.limiter.Wait(ctx); err != nil {
		slog.WarnContext(ctx, "rate limit exceeded",
			"error", err,
			"current_limit", r.limiter.Limit(),
			"current_burst", r.limiter.Burst(),
		)
		return err
	}
	return nil
}

// Allow returns true if a request can be made according to the rate limit
func (r *RateLimiter) Allow() bool {
	return r.limiter.Allow()
}

// Reserve returns a Reservation that tells the caller how long to wait before
// making the request
func (r *RateLimiter) Reserve() *rate.Reservation {
	return r.limiter.Reserve()
}

// UpdateLimit updates the rate limit based on the server response
func (r *RateLimiter) UpdateLimit(remaining int, reset time.Time) {
	if remaining <= 0 {
		// If we're out of requests, set a very low limit
		r.limiter.SetLimit(0.1) // One request every 10 seconds
		r.limiter.SetBurst(1)
		return
	}

	// Calculate new rate based on remaining requests and reset time
	duration := time.Until(reset)
	if duration <= 0 {
		return
	}

	// Calculate requests per second
	rps := float64(remaining) / duration.Seconds()
	r.limiter.SetLimit(rate.Limit(rps))

	// Set burst to min(remaining/10, 5) to allow some bursting but not too much
	burst := remaining / 10
	if burst > 5 {
		burst = 5
	}
	if burst < 1 {
		burst = 1
	}
	r.limiter.SetBurst(burst)
}

// GetConfig returns the current rate limit configuration
func (r *RateLimiter) GetConfig() (requestsPerMinute float64, burst int) {
	if r.limiter == nil {
		return 0, 0
	}
	// Convert requests per second back to requests per minute
	return float64(r.limiter.Limit()) * 60, r.limiter.Burst()
}

// String returns a string representation of the RateLimiter
func (r *RateLimiter) String() string {
	if r == nil {
		return "RateLimiter<nil>"
	}
	requestsPerMinute, burst := r.GetConfig()
	return fmt.Sprintf("RateLimiter{requests_per_minute: %.1f, burst: %d}", requestsPerMinute, burst)
}
