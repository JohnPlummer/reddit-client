package reddit

import (
	"fmt"
	"net/http"
)

// RedditClient embeds Client, automatically exposing its methods.
type RedditClient struct {
	Auth    *Auth
	*Client // Embedding Client removes redundancy
}

// NewClient initializes a Reddit API client using client credentials flow.
func NewClient(clientID, clientSecret string, opts ...ClientOption) (*Client, error) {
	client := &Client{
		client:    &http.Client{},
		userAgent: "golang:reddit-client:v1.0 (by /u/yourusername)", // default
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	auth := &Auth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	if err := auth.Authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	client.Auth = auth
	return client, nil
}
