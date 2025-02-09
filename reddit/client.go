package reddit

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	Auth      *Auth
	userAgent string
	client    *http.Client
}

func (c *Client) request(endpoint string) (*http.Response, error) {
	if err := c.Auth.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("ensuring valid token: %w", err)
	}

	req, err := http.NewRequest("GET", "https://oauth.reddit.com"+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Auth.Token)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return resp, nil
}

// GetComments fetches comments for a given post
func (c *Client) GetComments(subreddit, postID string, params map[string]string) ([]interface{}, error) {
	endpoint := fmt.Sprintf("/r/%s/comments/%s", subreddit, postID)
	if len(params) > 0 {
		endpoint += "?"
		for k, v := range params {
			endpoint += fmt.Sprintf("%s=%s&", k, v)
		}
		endpoint = endpoint[:len(endpoint)-1] // Remove trailing &
	}

	resp, err := c.request(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

// Add configuration options
type ClientOption func(*Client)

func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// GetPosts fetches a single page of posts from a subreddit
func (c *Client) GetPosts(subreddit string, params map[string]string) ([]Post, string, error) {
	endpoint := fmt.Sprintf("/r/%s.json", subreddit)
	if len(params) > 0 {
		endpoint += "?"
		for k, v := range params {
			endpoint += fmt.Sprintf("%s=%s&", k, v)
		}
		endpoint = endpoint[:len(endpoint)-1] // Remove trailing &
	}

	resp, err := c.request(endpoint)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, "", err
	}

	return parsePosts(data)
}

// NewClient creates a new Reddit client
func NewClient(auth *Auth, opts ...ClientOption) *Client {
	c := &Client{
		Auth:   auth,
		client: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
