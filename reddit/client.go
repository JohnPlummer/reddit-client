package reddit

import (
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
	"time"
)

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

// Client represents a Reddit API client
type Client struct {
	Auth        *Auth
	userAgent   string
	client      *http.Client
	rateLimiter *RateLimiter
	retryConfig *RetryConfig
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
func (c *Client) updateRateLimitFromHeaders(headers http.Header, endpoint string) {
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
		slog.Debug("rate limit headers processed",
			"remaining", remaining,
			"used", used,
			"reset", reset,
			"endpoint", endpoint)
	}
}

// requestJSON performs an HTTP request and decodes the JSON response into the provided result
func (c *Client) requestJSON(ctx context.Context, method, endpoint string, result any) error {
	resp, err := c.request(ctx, method, endpoint)
	if err != nil {
		return fmt.Errorf("client.requestJSON: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("client.requestJSON: decoding JSON response failed for %s %s: %w", method, endpoint, err)
	}

	return nil
}

// request performs an HTTP request with rate limiting, retry logic, and error handling
func (c *Client) request(ctx context.Context, method, endpoint string) (*http.Response, error) {
	if err := c.Auth.EnsureValidToken(ctx); err != nil {
		return nil, fmt.Errorf("client.request: ensuring valid token failed: %w", err)
	}

	// Wait for rate limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("client.request: rate limit wait failed: %w", err)
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
			return nil, fmt.Errorf("client.request: creating request failed: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.Auth.Token)
		req.Header.Set("User-Agent", c.userAgent)

		slog.Debug("making HTTP request",
			"method", method,
			"endpoint", endpoint,
			"attempt", attempt+1,
			"max_attempts", maxAttempts)

		resp, err = c.client.Do(req)
		if err != nil {
			lastError = fmt.Errorf("client.request: making request failed: %w", err)

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

		// Parse and update rate limit based on response headers
		c.updateRateLimitFromHeaders(resp.Header, endpoint)

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
			// Read and close the response body for retryable errors
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

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
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, NewAPIError(resp, body)
	}

	// This should never be reached, but just in case
	if lastError != nil {
		return nil, lastError
	}
	return nil, fmt.Errorf("client.request: exhausted all retry attempts")
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
		Auth:        auth,
		rateLimiter: NewRateLimiter(60, 5), // Default to 60 requests per minute with burst of 5
		userAgent:   "golang:reddit-client:v1.0",
		client:      &http.Client{}, // Default HTTP client
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
