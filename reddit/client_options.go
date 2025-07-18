package reddit

import (
	"fmt"
	"log/slog"
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

// WithCompression enables or disables HTTP response compression (gzip).
// When enabled, the client will automatically add "Accept-Encoding: gzip" headers
// to requests and decompress gzip-compressed responses transparently.
//
// Compression is enabled by default for better performance and bandwidth usage.
// You may want to disable it in specific scenarios such as:
//   - Debugging HTTP traffic with tools that don't handle compression
//   - Working with servers that have buggy compression implementations
//   - When you need to measure exact response sizes
//
// Example usage:
//
//	// Disable compression
//	client, err := reddit.NewClient(auth, reddit.WithCompression(false))
//
//	// Explicitly enable compression (default behavior)
//	client, err := reddit.NewClient(auth, reddit.WithCompression(true))
func WithCompression(enabled bool) ClientOption {
	return func(c *Client) {
		c.compressionEnabled = enabled
	}
}

// WithNoCompression disables HTTP response compression.
// This is a convenience method equivalent to WithCompression(false).
//
// Example usage:
//
//	client, err := reddit.NewClient(auth, reddit.WithNoCompression())
func WithNoCompression() ClientOption {
	return func(c *Client) {
		c.compressionEnabled = false
	}
}

// WithRateLimitHook sets a hook for monitoring rate limit events.
// The hook will be called when rate limits are updated, exceeded, or when waiting.
func WithRateLimitHook(hook RateLimitHook) ClientOption {
	return func(c *Client) {
		c.rateLimitHook = hook
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

// WithCircuitBreaker enables circuit breaker functionality for API resilience.
// The circuit breaker monitors request failures and automatically fails fast
// when the failure threshold is exceeded, helping prevent cascading failures.
//
// The circuit breaker has three states:
//   - Closed: Normal operation, all requests are allowed
//   - Open: Fast-fail mode, requests fail immediately without being sent
//   - Half-Open: Testing mode, limited requests are allowed to test recovery
//
// Example usage:
//
//	config := &reddit.CircuitBreakerConfig{
//		FailureThreshold: 5,                 // Open after 5 consecutive failures
//		SuccessThreshold: 3,                 // Close after 3 consecutive successes in half-open
//		Timeout:          30 * time.Second,  // Stay open for 30 seconds before trying half-open
//		MaxRequests:      5,                 // Allow up to 5 concurrent requests in half-open
//	}
//	client, err := reddit.NewClient(auth, reddit.WithCircuitBreaker(config))
func WithCircuitBreaker(config *CircuitBreakerConfig) ClientOption {
	return func(c *Client) {
		c.circuitBreaker = NewCircuitBreaker(config)
	}
}

// WithDefaultCircuitBreaker enables circuit breaker with default configuration.
// This is a convenience method that uses sensible defaults:
//   - Failure threshold: 5 consecutive failures
//   - Success threshold: 3 consecutive successes
//   - Timeout: 30 seconds
//   - Max requests in half-open: 5
//   - Only trips on server errors, timeouts, and temporary errors
func WithDefaultCircuitBreaker() ClientOption {
	return func(c *Client) {
		c.circuitBreaker = NewCircuitBreaker(DefaultCircuitBreakerConfig())
	}
}

// WithRequestInterceptor adds a request interceptor to the client.
// Request interceptors are called in the order they are added, before each HTTP request is sent.
// They can inspect and modify the request, or return an error to cancel the request.
//
// Example usage:
//
//	client, err := reddit.NewClient(auth,
//		reddit.WithRequestInterceptor(func(req *http.Request) error {
//			req.Header.Set("X-Custom-Header", "value")
//			return nil
//		}),
//	)
func WithRequestInterceptor(interceptor RequestInterceptor) ClientOption {
	return func(c *Client) {
		c.requestInterceptors = append(c.requestInterceptors, interceptor)
	}
}

// WithResponseInterceptor adds a response interceptor to the client.
// Response interceptors are called in the order they are added, after each HTTP response is received.
// They can inspect the response and return an error to indicate a problem.
//
// Example usage:
//
//	client, err := reddit.NewClient(auth,
//		reddit.WithResponseInterceptor(func(resp *http.Response) error {
//			if resp.Header.Get("X-Deprecated-API") != "" {
//				log.Warn("Using deprecated API endpoint")
//			}
//			return nil
//		}),
//	)
func WithResponseInterceptor(interceptor ResponseInterceptor) ClientOption {
	return func(c *Client) {
		c.responseInterceptors = append(c.responseInterceptors, interceptor)
	}
}

// TransportConfig holds configuration for HTTP transport connection pooling
type TransportConfig struct {
	// MaxIdleConns controls the maximum number of idle (keep-alive)
	// connections across all hosts. Zero means no limit.
	// Default: 100 (recommended for Reddit API)
	MaxIdleConns int

	// MaxIdleConnsPerHost controls the maximum idle (keep-alive)
	// connections to keep per-host. Zero means DefaultMaxIdleConnsPerHost.
	// Default: 10 (recommended for Reddit API)
	MaxIdleConnsPerHost int

	// IdleConnTimeout is the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing itself.
	// Zero means no limit.
	// Default: 90 seconds (recommended for Reddit API)
	IdleConnTimeout time.Duration

	// DisableKeepAlives disables HTTP keep-alives when true.
	// This is generally not recommended for API clients.
	// Default: false
	DisableKeepAlives bool

	// MaxConnsPerHost limits the total number of connections per host.
	// This includes connections in the dialing, active, and idle states.
	// Zero means no limit.
	// Default: 0 (no limit)
	MaxConnsPerHost int
}

// DefaultTransportConfig returns a default transport configuration optimized for Reddit API
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		MaxConnsPerHost:     0, // No limit by default
	}
}

// WithTransportConfig configures the HTTP transport for connection pooling.
// This allows fine-tuning of connection behavior for optimal performance.
//
// Recommended values for Reddit API:
//   - MaxIdleConns: 100 (total idle connections across all hosts)
//   - MaxIdleConnsPerHost: 10 (idle connections per Reddit endpoint)
//   - IdleConnTimeout: 90s (Reddit's typical connection timeout)
//   - DisableKeepAlives: false (keep-alive improves performance)
//   - MaxConnsPerHost: 0 (no limit, let the system manage)
//
// Example usage:
//
//	config := reddit.DefaultTransportConfig()
//	config.MaxIdleConnsPerHost = 20 // Increase for high-throughput apps
//	client, err := reddit.NewClient(auth, reddit.WithTransportConfig(config))
func WithTransportConfig(config *TransportConfig) ClientOption {
	return func(c *Client) {
		if config == nil {
			config = DefaultTransportConfig()
		}

		// Create a new transport or use the existing one
		var transport *http.Transport
		if c.client != nil && c.client.Transport != nil {
			if t, ok := c.client.Transport.(*http.Transport); ok {
				// Clone the existing transport to preserve other settings
				transport = t.Clone()
			} else {
				// If transport is not *http.Transport, create a new one
				transport = &http.Transport{}
			}
		} else {
			transport = &http.Transport{}
		}

		// Apply connection pooling configuration
		transport.MaxIdleConns = config.MaxIdleConns
		transport.MaxIdleConnsPerHost = config.MaxIdleConnsPerHost
		transport.IdleConnTimeout = config.IdleConnTimeout
		transport.DisableKeepAlives = config.DisableKeepAlives
		transport.MaxConnsPerHost = config.MaxConnsPerHost

		// Ensure we have an HTTP client
		if c.client == nil {
			c.client = &http.Client{}
		}

		// Set the configured transport
		c.client.Transport = transport
	}
}

// DefaultOptions returns the default set of options
func DefaultOptions() []ClientOption {
	return []ClientOption{
		WithUserAgent("golang:reddit-client:v1.0"),
		WithRateLimit(60, 5), // Default to 60 requests per minute with burst of 5
		WithTimeout(10 * time.Second),
		WithRetryConfig(DefaultRetryConfig()),         // Enable retries by default
		WithTransportConfig(DefaultTransportConfig()), // Enable optimized connection pooling by default
		WithCompression(true),                         // Enable compression by default for better performance
	}
}

// Example Interceptors

// LoggingRequestInterceptor returns a request interceptor that logs outgoing HTTP requests.
// This is useful for debugging and monitoring API calls.
//
// Example usage:
//
//	client, err := reddit.NewClient(auth,
//		reddit.WithRequestInterceptor(reddit.LoggingRequestInterceptor()),
//	)
func LoggingRequestInterceptor() RequestInterceptor {
	return func(req *http.Request) error {
		slog.Info("outgoing HTTP request",
			"method", req.Method,
			"url", req.URL.String(),
			"user_agent", req.Header.Get("User-Agent"),
			"headers", req.Header)
		return nil
	}
}

// LoggingResponseInterceptor returns a response interceptor that logs incoming HTTP responses.
// This is useful for debugging and monitoring API responses.
//
// Example usage:
//
//	client, err := reddit.NewClient(auth,
//		reddit.WithResponseInterceptor(reddit.LoggingResponseInterceptor()),
//	)
func LoggingResponseInterceptor() ResponseInterceptor {
	return func(resp *http.Response) error {
		var url string
		if resp.Request != nil && resp.Request.URL != nil {
			url = resp.Request.URL.String()
		} else {
			url = "unknown"
		}

		slog.Info("incoming HTTP response",
			"status_code", resp.StatusCode,
			"status", resp.Status,
			"url", url,
			"content_length", resp.ContentLength,
			"headers", resp.Header)
		return nil
	}
}

// HeaderInjectionRequestInterceptor returns a request interceptor that adds custom headers to all requests.
// This is useful for adding authentication, tracing, or custom metadata headers.
//
// Example usage:
//
//	headers := map[string]string{
//		"X-Request-ID": "unique-request-id",
//		"X-Client-Version": "1.0.0",
//	}
//	client, err := reddit.NewClient(auth,
//		reddit.WithRequestInterceptor(reddit.HeaderInjectionRequestInterceptor(headers)),
//	)
func HeaderInjectionRequestInterceptor(headers map[string]string) RequestInterceptor {
	return func(req *http.Request) error {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		return nil
	}
}

// DeprecationWarningResponseInterceptor returns a response interceptor that warns about deprecated API usage.
// It checks for deprecation headers and logs warnings when found.
//
// Example usage:
//
//	client, err := reddit.NewClient(auth,
//		reddit.WithResponseInterceptor(reddit.DeprecationWarningResponseInterceptor()),
//	)
func DeprecationWarningResponseInterceptor() ResponseInterceptor {
	return func(resp *http.Response) error {
		var url string
		if resp.Request != nil && resp.Request.URL != nil {
			url = resp.Request.URL.String()
		} else {
			url = "unknown"
		}

		if deprecation := resp.Header.Get("X-API-Deprecated"); deprecation != "" {
			slog.Warn("API deprecation warning",
				"url", url,
				"deprecation_info", deprecation)
		}
		if sunset := resp.Header.Get("Sunset"); sunset != "" {
			slog.Warn("API sunset warning",
				"url", url,
				"sunset_date", sunset)
		}
		return nil
	}
}

// RequestIDRequestInterceptor returns a request interceptor that adds a unique request ID header.
// This is useful for request tracing and correlation across logs.
//
// Example usage:
//
//	client, err := reddit.NewClient(auth,
//		reddit.WithRequestInterceptor(reddit.RequestIDRequestInterceptor("X-Request-ID")),
//	)
func RequestIDRequestInterceptor(headerName string) RequestInterceptor {
	return func(req *http.Request) error {
		if req.Header.Get(headerName) == "" {
			// Generate a simple request ID (in production, consider using UUID)
			requestID := fmt.Sprintf("req_%d_%s", time.Now().UnixNano(), req.Method)
			req.Header.Set(headerName, requestID)
		}
		return nil
	}
}
