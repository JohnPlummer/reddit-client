package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// Client represents a Reddit API client
type Client struct {
	Auth        *Auth
	userAgent   string
	client      *http.Client
	rateLimiter *RateLimiter
}

// request performs an HTTP request with rate limiting and error handling
func (c *Client) request(ctx context.Context, method, endpoint string) (*http.Response, error) {
	if err := c.Auth.EnsureValidToken(ctx); err != nil {
		return nil, fmt.Errorf("ensuring valid token: %w", err)
	}

	// Wait for rate limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, "https://oauth.reddit.com"+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Auth.Token)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}

	// Update rate limit based on response headers
	if remaining := resp.Header.Get("X-Ratelimit-Remaining"); remaining != "" {
		if rem, err := strconv.Atoi(remaining); err == nil {
			resetStr := resp.Header.Get("X-Ratelimit-Reset")
			resetInt, _ := strconv.ParseInt(resetStr, 10, 64)
			reset := time.Unix(resetInt, 0)
			c.rateLimiter.UpdateLimit(rem, reset)
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, NewAPIError(resp, body)
	}

	return resp, nil
}

// getComments is an internal method for fetching comments
func (c *Client) getComments(ctx context.Context, subreddit, postID string, params map[string]string) ([]interface{}, error) {
	endpoint := fmt.Sprintf("/r/%s/comments/%s", subreddit, postID)
	if len(params) > 0 {
		endpoint += "?"
		for k, v := range params {
			endpoint += fmt.Sprintf("%s=%s&", k, v)
		}
		endpoint = endpoint[:len(endpoint)-1] // Remove trailing &
	}

	resp, err := c.request(ctx, "GET", endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return data, nil
}

// GetPosts fetches a single page of posts from a subreddit
func (c *Client) GetPosts(ctx context.Context, subreddit string, params map[string]string) ([]Post, string, error) {
	endpoint := fmt.Sprintf("/r/%s.json", subreddit)
	if len(params) > 0 {
		endpoint += "?"
		for k, v := range params {
			endpoint += fmt.Sprintf("%s=%s&", k, v)
		}
		endpoint = endpoint[:len(endpoint)-1] // Remove trailing &
	}

	resp, err := c.request(ctx, "GET", endpoint)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, "", fmt.Errorf("decoding response: %w", err)
	}

	return parsePosts(data, c)
}

// NewClient creates a new Reddit client
func NewClient(auth *Auth, opts ...Option) (*Client, error) {
	if auth == nil {
		return nil, ErrMissingCredentials
	}

	c := &Client{
		Auth:        auth,
		client:      &http.Client{},
		rateLimiter: NewRateLimiter(60, 5), // Default to 60 requests per minute with burst of 5
	}

	// Apply options
	for _, opt := range opts {
		switch o := opt.(type) {
		case UserAgentOption:
			c.userAgent = o.UserAgent
		case RateLimitOption:
			c.rateLimiter = NewRateLimiter(o.RequestsPerMinute, o.BurstSize)
		case TimeoutOption:
			c.client.Timeout = o.Timeout
		}
	}

	if c.userAgent == "" {
		c.userAgent = "golang:reddit-client:v1.0"
	}

	slog.Debug("creating new client",
		"user_agent", c.userAgent,
		"rate_limit", c.rateLimiter.limiter.Limit(),
		"burst", c.rateLimiter.limiter.Burst(),
	)

	return c, nil
}
