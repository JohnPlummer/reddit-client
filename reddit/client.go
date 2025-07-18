package reddit

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// RateLimitHook provides callbacks for rate limiting events
type RateLimitHook interface {
	// OnRateLimitWait is called when the client is waiting due to rate limits
	OnRateLimitWait(ctx context.Context, duration time.Duration)

	// OnRateLimitUpdate is called when rate limit information is updated from headers
	OnRateLimitUpdate(remaining int, reset time.Time)

	// OnRateLimitExceeded is called when rate limit is exceeded (remaining = 0)
	OnRateLimitExceeded(ctx context.Context)
}

// LoggingRateLimitHook provides a default implementation that logs rate limit events using slog
type LoggingRateLimitHook struct{}

// OnRateLimitWait logs when the client is waiting due to rate limits
func (h *LoggingRateLimitHook) OnRateLimitWait(ctx context.Context, duration time.Duration) {
	slog.InfoContext(ctx, "rate limit wait",
		"duration", duration,
		"duration_ms", duration.Milliseconds())
}

// OnRateLimitUpdate logs when rate limit information is updated
func (h *LoggingRateLimitHook) OnRateLimitUpdate(remaining int, reset time.Time) {
	slog.Info("rate limit updated",
		"remaining", remaining,
		"reset", reset,
		"reset_in", time.Until(reset))
}

// OnRateLimitExceeded logs when rate limit is exceeded
func (h *LoggingRateLimitHook) OnRateLimitExceeded(ctx context.Context) {
	slog.WarnContext(ctx, "rate limit exceeded",
		"message", "API rate limit has been exceeded")
}

// BuildEndpoint constructs a URL endpoint with query parameters using proper URL encoding
func BuildEndpoint(base string, params map[string]string) string {
	if len(params) == 0 {
		return base
	}

	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}

	return base + "?" + values.Encode()
}

// RequestInterceptor is a function that can inspect and modify HTTP requests before they are sent.
// It receives the request that is about to be sent and can return an error to cancel the request.
// Interceptors are called in the order they are registered.
type RequestInterceptor func(req *http.Request) error

// ResponseInterceptor is a function that can inspect HTTP responses after they are received.
// It receives the response that was received and can return an error to indicate a problem.
// Interceptors are called in the order they are registered.
type ResponseInterceptor func(resp *http.Response) error

// Client represents a Reddit API client
type Client struct {
	Auth                 *Auth
	userAgent            string
	client               *http.Client
	rateLimiter          *RateLimiter
	retryConfig          *RetryConfig
	rateLimitHook        RateLimitHook
	circuitBreaker       *CircuitBreaker
	requestInterceptors  []RequestInterceptor
	responseInterceptors []ResponseInterceptor
	compressionEnabled   bool
}

// isRetryableStatusCode checks if a status code should trigger a retry
func (c *Client) isRetryableStatusCode(statusCode int) bool {
	if c.retryConfig == nil {
		return false
	}
	for _, code := range c.retryConfig.RetryableCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}

// calculateRetryDelay calculates the delay for the next retry attempt with exponential backoff and jitter
func (c *Client) calculateRetryDelay(attempt int, retryAfter time.Duration) time.Duration {
	if c.retryConfig == nil {
		return 0
	}

	// If Retry-After header is present and we respect it, use that
	if retryAfter > 0 && c.retryConfig.RespectRetryAfter {
		return retryAfter
	}

	// Calculate exponential backoff: baseDelay * 2^attempt
	delay := time.Duration(float64(c.retryConfig.BaseDelay) * math.Pow(2, float64(attempt)))

	// Cap at maximum delay
	if delay > c.retryConfig.MaxDelay {
		delay = c.retryConfig.MaxDelay
	}

	// Add jitter to prevent thundering herd
	if c.retryConfig.JitterFactor > 0 {
		jitter := time.Duration(float64(delay) * c.retryConfig.JitterFactor * (rand.Float64() - 0.5))
		delay += jitter
	}

	return delay
}

// parseRetryAfter parses the Retry-After header and returns the delay duration
func parseRetryAfter(retryAfterHeader string) time.Duration {
	if retryAfterHeader == "" {
		return 0
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date (RFC 1123)
	if t, err := time.Parse(time.RFC1123, retryAfterHeader); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
	}

	return 0
}

// updateRateLimitFromHeaders extracts rate limit information from response headers and updates the rate limiter
func (c *Client) updateRateLimitFromHeaders(ctx context.Context, headers http.Header, endpoint string) {
	remainingStr := headers.Get("X-Ratelimit-Remaining")
	usedStr := headers.Get("X-Ratelimit-Used")
	resetStr := headers.Get("X-Ratelimit-Reset")

	// If no rate limit headers are present, skip update
	if remainingStr == "" && usedStr == "" && resetStr == "" {
		return
	}

	var remaining, used int
	var reset time.Time
	var hasValidData bool

	// Parse remaining requests
	if remainingStr != "" {
		if rem, err := strconv.Atoi(remainingStr); err == nil {
			remaining = rem
			hasValidData = true
		} else {
			slog.Warn("failed to parse X-Ratelimit-Remaining header",
				"header_value", remainingStr,
				"error", err,
				"endpoint", endpoint)
		}
	}

	// Parse used requests
	if usedStr != "" {
		if u, err := strconv.Atoi(usedStr); err == nil {
			used = u
		} else {
			slog.Warn("failed to parse X-Ratelimit-Used header",
				"header_value", usedStr,
				"error", err,
				"endpoint", endpoint)
		}
	}

	// Parse reset timestamp
	if resetStr != "" {
		if resetInt, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			reset = time.Unix(resetInt, 0)
			hasValidData = true
		} else {
			slog.Warn("failed to parse X-Ratelimit-Reset header",
				"header_value", resetStr,
				"error", err,
				"endpoint", endpoint)
		}
	}

	// Only update rate limiter if we have at least remaining or reset data
	if hasValidData {
		c.rateLimiter.UpdateLimitWithUsed(remaining, used, reset)

		// Call the rate limit hook if configured
		if c.rateLimitHook != nil {
			c.rateLimitHook.OnRateLimitUpdate(remaining, reset)

			// Check if rate limit is exceeded
			if remaining <= 0 {
				c.rateLimitHook.OnRateLimitExceeded(ctx)
			}
		}

		slog.Debug("rate limit headers processed",
			"remaining", remaining,
			"used", used,
			"reset", reset,
			"endpoint", endpoint)
	}
}

// getResponseReader returns the appropriate reader for the response body, handling compression if needed
func (c *Client) getResponseReader(resp *http.Response) (io.ReadCloser, error) {
	if c.compressionEnabled && strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("client.getResponseReader: creating gzip reader failed: %w", err)
		}

		// Create a composite reader that closes both gzip reader and original body
		return &gzipReaderCloser{
			gzipReader: gzipReader,
			original:   resp.Body,
		}, nil
	}

	return resp.Body, nil
}

// gzipReaderCloser wraps a gzip reader and ensures both the gzip reader and original body are closed
type gzipReaderCloser struct {
	gzipReader *gzip.Reader
	original   io.ReadCloser
}

func (g *gzipReaderCloser) Read(p []byte) (n int, err error) {
	return g.gzipReader.Read(p)
}

func (g *gzipReaderCloser) Close() error {
	// Close gzip reader first, then original body
	if err := g.gzipReader.Close(); err != nil {
		g.original.Close() // Still try to close original
		return err
	}
	return g.original.Close()
}

// requestJSON performs an HTTP request and decodes the JSON response into the provided result
func (c *Client) requestJSON(ctx context.Context, method, endpoint string, result any) error {
	resp, err := c.request(ctx, method, endpoint)
	if err != nil {
		return fmt.Errorf("client.requestJSON: request failed: %w", err)
	}
	defer resp.Body.Close()

	// Get the appropriate reader (handles compression if enabled)
	reader, err := c.getResponseReader(resp)
	if err != nil {
		return fmt.Errorf("client.requestJSON: getting response reader failed: %w", err)
	}
	defer reader.Close()

	if err := json.NewDecoder(reader).Decode(result); err != nil {
		return fmt.Errorf("client.requestJSON: decoding JSON response failed for %s %s: %w", method, endpoint, err)
	}

	return nil
}

// request performs an HTTP request with rate limiting, retry logic, and error handling
func (c *Client) request(ctx context.Context, method, endpoint string) (*http.Response, error) {
	if err := c.Auth.EnsureValidToken(ctx); err != nil {
		return nil, fmt.Errorf("client.request: ensuring valid token failed: %w", err)
	}

	// If circuit breaker is configured, wrap the request in circuit breaker protection
	if c.circuitBreaker != nil {
		var resp *http.Response
		err := c.circuitBreaker.Execute(func() error {
			var requestErr error
			resp, requestErr = c.performRequest(ctx, method, endpoint)
			return requestErr
		})
		return resp, err
	}

	// No circuit breaker, perform request directly
	return c.performRequest(ctx, method, endpoint)
}

// performRequest performs the actual HTTP request with rate limiting and retry logic
func (c *Client) performRequest(ctx context.Context, method, endpoint string) (*http.Response, error) {
	// Wait for rate limit
	if c.rateLimitHook != nil {
		// Use Reserve to check if we need to wait
		reservation := c.rateLimiter.Reserve()
		delay := reservation.Delay()
		if delay > 0 {
			c.rateLimitHook.OnRateLimitWait(ctx, delay)
		}
		// Cancel the reservation since we'll use Wait() instead
		reservation.Cancel()
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("client.performRequest: rate limit wait failed: %w", err)
	}

	var resp *http.Response
	var lastError error

	maxAttempts := 1
	if c.retryConfig != nil {
		maxAttempts = c.retryConfig.MaxRetries + 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Create a new request for each attempt
		req, err := http.NewRequestWithContext(ctx, method, "https://oauth.reddit.com"+endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("client.performRequest: creating request failed: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.Auth.Token)
		req.Header.Set("User-Agent", c.userAgent)

		// Add compression header if enabled
		if c.compressionEnabled {
			req.Header.Set("Accept-Encoding", "gzip")
		}

		// Call request interceptors
		for i, interceptor := range c.requestInterceptors {
			if err := interceptor(req); err != nil {
				return nil, fmt.Errorf("client.performRequest: request interceptor %d failed: %w", i, err)
			}
		}

		slog.Debug("making HTTP request",
			"method", method,
			"endpoint", endpoint,
			"attempt", attempt+1,
			"max_attempts", maxAttempts)

		resp, err = c.client.Do(req)
		if err != nil {
			lastError = fmt.Errorf("client.performRequest: making request failed: %w", err)

			// For network errors, only retry if we have retry config and attempts left
			if c.retryConfig != nil && attempt < maxAttempts-1 {
				delay := c.calculateRetryDelay(attempt, 0)
				slog.Warn("request failed, retrying",
					"error", err,
					"attempt", attempt+1,
					"max_attempts", maxAttempts,
					"delay", delay,
					"endpoint", endpoint)

				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, lastError
		}

		// Call response interceptors
		for i, interceptor := range c.responseInterceptors {
			if err := interceptor(resp); err != nil {
				// Close the response body since we won't be returning it
				resp.Body.Close()
				return nil, fmt.Errorf("client.performRequest: response interceptor %d failed: %w", i, err)
			}
		}

		// Parse and update rate limit based on response headers
		c.updateRateLimitFromHeaders(ctx, resp.Header, endpoint)

		// Check if the response is successful
		if resp.StatusCode == http.StatusOK {
			slog.Debug("request successful",
				"status_code", resp.StatusCode,
				"endpoint", endpoint,
				"attempt", attempt+1)
			return resp, nil
		}

		// Check if this is a retryable error
		if c.retryConfig != nil && c.isRetryableStatusCode(resp.StatusCode) && attempt < maxAttempts-1 {
			// Read and close the response body for retryable errors (handle compression)
			reader, readerErr := c.getResponseReader(resp)
			var body []byte
			if readerErr == nil {
				body, _ = io.ReadAll(reader)
				reader.Close()
			} else {
				// Fallback to reading uncompressed body
				body, _ = io.ReadAll(resp.Body)
				resp.Body.Close()
			}

			// Parse Retry-After header if present
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			delay := c.calculateRetryDelay(attempt, retryAfter)

			lastError = NewAPIError(resp, body)

			slog.Warn("received retryable error, retrying",
				"status_code", resp.StatusCode,
				"error", lastError,
				"attempt", attempt+1,
				"max_attempts", maxAttempts,
				"delay", delay,
				"retry_after", retryAfter,
				"endpoint", endpoint)

			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Non-retryable error or no more attempts
		reader, readerErr := c.getResponseReader(resp)
		var body []byte
		if readerErr == nil {
			body, _ = io.ReadAll(reader)
			reader.Close()
		} else {
			// Fallback to reading uncompressed body
			body, _ = io.ReadAll(resp.Body)
			resp.Body.Close()
		}
		return nil, NewAPIError(resp, body)
	}

	// This should never be reached, but just in case
	if lastError != nil {
		return nil, lastError
	}
	return nil, fmt.Errorf("client.performRequest: exhausted all retry attempts")
}

// getComments is an internal method for fetching comments
func (c *Client) getComments(ctx context.Context, subreddit, postID string, opts ...CommentOption) ([]any, error) {
	params := map[string]string{
		"limit": "100", // Default limit
	}

	// Apply options
	for _, opt := range opts {
		opt(params)
	}

	base := fmt.Sprintf("/r/%s/comments/%s", subreddit, postID)
	endpoint := BuildEndpoint(base, params)

	var data []any
	if err := c.requestJSON(ctx, "GET", endpoint, &data); err != nil {
		return nil, fmt.Errorf("client.getComments: %w", err)
	}

	return data, nil
}

// getPosts fetches posts from a subreddit with optional pagination and filtering.
// This method will automatically fetch multiple pages as needed up to the specified limit.
// Set limit to 0 to fetch all available posts (use with caution).
func (c *Client) getPosts(ctx context.Context, subreddit string, opts ...PostOption) ([]Post, error) {
	params := map[string]string{
		"limit": "100", // Default limit
	}

	// Apply options
	for _, opt := range opts {
		opt(params)
	}

	// Extract pagination options from params
	limit := 0
	if limitStr, ok := params["limit"]; ok {
		limit, _ = strconv.Atoi(limitStr)
	}

	initialAfter := params["after"]

	// Create fetch function that uses current parameters
	fetchPage := func(ctx context.Context, after string) ([]Post, string, error) {
		// Create a copy of params for this request
		requestParams := make(map[string]string)
		for k, v := range params {
			requestParams[k] = v
		}

		// Override the after parameter
		if after != "" {
			requestParams["after"] = after
		} else {
			// Remove after parameter if empty (for first request)
			delete(requestParams, "after")
		}

		return c.getPostsPage(ctx, subreddit, requestParams)
	}

	// Configure pagination options
	paginationOpts := PaginationOptions{
		Limit:       limit,
		PageSize:    100,
		StopOnEmpty: true,
	}

	// Handle initial after token if provided
	if initialAfter != "" {
		// Modify fetch function to use initial after for first call
		firstCall := true
		originalFetchPage := fetchPage
		fetchPage = func(ctx context.Context, after string) ([]Post, string, error) {
			if firstCall {
				firstCall = false
				return originalFetchPage(ctx, initialAfter)
			}
			return originalFetchPage(ctx, after)
		}
	}

	return PaginateAll(ctx, fetchPage, paginationOpts)
}

// getPostsPage fetches a single page of posts from a subreddit
func (c *Client) getPostsPage(ctx context.Context, subreddit string, params map[string]string) ([]Post, string, error) {
	base := fmt.Sprintf("/r/%s.json", subreddit)
	endpoint := BuildEndpoint(base, params)

	var data map[string]any
	if err := c.requestJSON(ctx, "GET", endpoint, &data); err != nil {
		return nil, "", fmt.Errorf("client.getPostsPage: %w", err)
	}

	return parsePosts(data, c)
}

// NewClient creates a new Reddit client with the provided options
func NewClient(auth *Auth, opts ...ClientOption) (*Client, error) {
	if auth == nil {
		return nil, fmt.Errorf("client.NewClient: auth is required for client creation")
	}

	// Start with default options
	c := &Client{
		Auth:               auth,
		rateLimiter:        NewRateLimiter(60, 5), // Default to 60 requests per minute with burst of 5
		userAgent:          "golang:reddit-client:v1.0",
		client:             &http.Client{}, // Default HTTP client
		compressionEnabled: true,           // Enable compression by default
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	if c.client == nil {
		c.client = &http.Client{} // Ensure we always have an HTTP client
	}

	slog.Debug("creating new client", "client", c)

	return c, nil
}

// String returns a string representation of the Client struct, safely handling sensitive data
func (c *Client) String() string {
	if c == nil {
		return "Client<nil>"
	}

	return fmt.Sprintf("Client{Auth: %v, UserAgent: %q, %v}",
		c.Auth,
		c.userAgent,
		c.rateLimiter,
	)
}
