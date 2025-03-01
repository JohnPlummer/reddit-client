package reddit

import (
	"net/http"
	"time"
)

// ClientOption represents a function that configures a Client
type ClientOption func(*Client)

// WithUserAgent sets a custom user agent for Reddit API requests
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithRateLimit sets custom rate limiting parameters
func WithRateLimit(requestsPerMinute, burstSize int) ClientOption {
	return func(c *Client) {
		c.rateLimiter = NewRateLimiter(requestsPerMinute, burstSize)
	}
}

// WithTimeout sets the timeout for API requests
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		if c.client == nil {
			c.client = &http.Client{}
		}
		c.client.Timeout = timeout
	}
}

// WithHTTPClient sets the HTTP client used for making requests.
// This allows for complete customization of HTTP behavior including
// transport, timeout, cookies, and redirects.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.client = client
	}
}

// WithAuth sets the Auth client
func WithAuth(auth *Auth) ClientOption {
	return func(c *Client) {
		c.Auth = auth
	}
}

// DefaultOptions returns the default set of options
func DefaultOptions() []ClientOption {
	return []ClientOption{
		WithUserAgent("golang:reddit-client:v1.0"),
		WithRateLimit(60, 5), // Default to 60 requests per minute with burst of 5
		WithTimeout(10 * time.Second),
	}
}
