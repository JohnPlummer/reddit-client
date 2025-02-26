package reddit

import (
	"net/http"
	"time"
)

// Option represents a configuration option that can be applied to various components
type Option interface {
	isOption() // Marker method to ensure type safety
}

// UserAgentOption configures the user agent for Reddit API requests
type UserAgentOption struct {
	UserAgent string
}

func (UserAgentOption) isOption() {}

// WithUserAgent sets a custom user agent for Reddit API requests
func WithUserAgent(userAgent string) Option {
	return UserAgentOption{UserAgent: userAgent}
}

// RateLimitOption configures rate limiting behavior
type RateLimitOption struct {
	RequestsPerMinute int
	BurstSize         int
}

func (RateLimitOption) isOption() {}

// WithRateLimit sets custom rate limiting parameters
func WithRateLimit(requestsPerMinute, burstSize int) Option {
	return RateLimitOption{
		RequestsPerMinute: requestsPerMinute,
		BurstSize:         burstSize,
	}
}

// TimeoutOption configures request timeouts
type TimeoutOption struct {
	Timeout time.Duration
}

func (TimeoutOption) isOption() {}

// WithTimeout sets the timeout for API requests
func WithTimeout(timeout time.Duration) Option {
	return TimeoutOption{Timeout: timeout}
}

// TransportOption configures the HTTP transport
type TransportOption struct {
	Transport http.RoundTripper
}

func (TransportOption) isOption() {}

// WithTransport sets a custom transport for HTTP requests
func WithTransport(transport http.RoundTripper) Option {
	return TransportOption{Transport: transport}
}

// DefaultOptions returns the default set of options
func DefaultOptions() []Option {
	return []Option{
		WithUserAgent("golang:reddit-client:v1.0"),
		WithRateLimit(60, 5), // Default to 60 requests per minute with burst of 5
		WithTimeout(10 * time.Second),
	}
}
