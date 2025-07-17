package reddit

import (
	"net/http"
	"strconv"
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

// PostOption is a function type for modifying post request parameters
type PostOption func(params map[string]string)

// WithAfter returns a PostOption that sets the "after" parameter for pagination
func WithAfter(after *Post) PostOption {
	return func(params map[string]string) {
		if after != nil {
			params["after"] = after.Fullname()
		}
	}
}

// WithLimit returns a PostOption that sets the "limit" parameter
func WithLimit(limit int) PostOption {
	return func(params map[string]string) {
		if limit > 0 {
			params["limit"] = strconv.Itoa(limit)
		}
	}
}

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries        int           // Maximum number of retry attempts (default: 3)
	BaseDelay         time.Duration // Base delay for exponential backoff (default: 1s)
	MaxDelay          time.Duration // Maximum delay between retries (default: 8s)
	JitterFactor      float64       // Jitter factor to add randomness (default: 0.1)
	RetryableCodes    []int         // HTTP status codes that should trigger retries
	RespectRetryAfter bool          // Whether to respect Retry-After headers (default: true)
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		BaseDelay:         1 * time.Second,
		MaxDelay:          8 * time.Second,
		JitterFactor:      0.1,
		RetryableCodes:    []int{429, 502, 503},
		RespectRetryAfter: true,
	}
}

// WithRetryConfig sets the retry configuration for the client
func WithRetryConfig(config *RetryConfig) ClientOption {
	return func(c *Client) {
		if config == nil {
			config = DefaultRetryConfig()
		}
		c.retryConfig = config
	}
}

// WithRetries enables retry logic with the specified maximum number of retries
func WithRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		if c.retryConfig == nil {
			c.retryConfig = DefaultRetryConfig()
		}
		c.retryConfig.MaxRetries = maxRetries
	}
}

// WithRetryDelay sets the base delay for exponential backoff
func WithRetryDelay(baseDelay time.Duration) ClientOption {
	return func(c *Client) {
		if c.retryConfig == nil {
			c.retryConfig = DefaultRetryConfig()
		}
		c.retryConfig.BaseDelay = baseDelay
	}
}

// WithNoRetries disables retry logic
func WithNoRetries() ClientOption {
	return func(c *Client) {
		c.retryConfig = nil
	}
}

// DefaultOptions returns the default set of options
func DefaultOptions() []ClientOption {
	return []ClientOption{
		WithUserAgent("golang:reddit-client:v1.0"),
		WithRateLimit(60, 5), // Default to 60 requests per minute with burst of 5
		WithTimeout(10 * time.Second),
		WithRetryConfig(DefaultRetryConfig()), // Enable retries by default
	}
}
