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

// getPosts fetches a single page of posts from a subreddit
func (c *Client) getPosts(ctx context.Context, subreddit string, params map[string]string) ([]Post, string, error) {
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

// getPostsAfter fetches posts from a subreddit that come after the specified post.
// This method will automatically fetch multiple pages as needed up to the specified limit.
// Set limit to 0 to fetch all available posts (use with caution).
//
// This is useful for implementing infinite scroll or pagination. The posts are returned
// in chronological order (newest first).
func (c *Client) getPostsAfter(ctx context.Context, subreddit string, after *Post, limit int) ([]Post, error) {
	params := map[string]string{"limit": "100"}
	var allPosts []Post

	// If after is provided, use its fullname as the after cursor
	if after != nil {
		params["after"] = after.Fullname()
	}

	for {
		posts, nextAfter, err := c.getPosts(ctx, subreddit, params)
		if err != nil {
			return nil, err
		}

		allPosts = append(allPosts, posts...)

		// Stop if we've reached the desired limit
		if limit > 0 && len(allPosts) >= limit {
			allPosts = allPosts[:limit]
			break
		}

		// Stop if there are no more pages
		if nextAfter == "" {
			break
		}

		// Update the after parameter for the next request
		params["after"] = nextAfter
	}

	return allPosts, nil
}

// NewClient creates a new Reddit client with the provided options
func NewClient(opts ...ClientOption) (*Client, error) {
	// Start with default options
	c := &Client{
		rateLimiter: NewRateLimiter(60, 5), // Default to 60 requests per minute with burst of 5
		userAgent:   "golang:reddit-client:v1.0",
		client:      &http.Client{}, // Default HTTP client
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Validate required configuration
	if c.Auth == nil {
		// Create default Auth if none provided
		var err error
		c.Auth, err = NewAuth("", "", WithAuthUserAgent(c.userAgent))
		if err != nil {
			return nil, fmt.Errorf("creating default auth client: %w", err)
		}
	}

	if c.client == nil {
		c.client = &http.Client{} // Ensure we always have an HTTP client
	}

	slog.Debug("creating new client",
		"user_agent", c.userAgent,
		"rate_limit", c.rateLimiter.limiter.Limit(),
		"burst", c.rateLimiter.limiter.Burst(),
	)

	return c, nil
}
