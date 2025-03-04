package reddit

import (
	"context"
	"fmt"
	"strconv"
)

// PostGetter defines the interface for fetching posts from Reddit
//
//go:generate mockgen -destination=mocks/subreddit_mocks.go -package=mocks github.com/JohnPlummer/reddit-client/reddit PostGetter
type PostGetter interface {
	GetPosts(subreddit string, params map[string]string) ([]Post, string, error)
}

// Subreddit represents a Reddit subreddit
type Subreddit struct {
	Name   string
	client *Client
}

// NewSubreddit creates a new Subreddit instance
func NewSubreddit(name string, client *Client) *Subreddit {
	return &Subreddit{
		Name:   name,
		client: client,
	}
}

// GetPosts fetches posts from the subreddit with optional pagination and filtering
func (s *Subreddit) GetPosts(ctx context.Context, opts ...SubredditOption) ([]Post, error) {
	params := map[string]string{
		"limit": "100", // Default limit
	}

	// Apply options
	for _, opt := range opts {
		opt(params)
	}

	// Convert params to PostOptions
	var postOpts []PostOption

	// Handle limit
	if limitStr, ok := params["limit"]; ok {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			postOpts = append(postOpts, WithLimit(limit))
		}
	}

	// Handle after parameter
	if after, ok := params["after"]; ok {
		postOpts = append(postOpts, WithAfter(&Post{ID: after[3:]})) // Remove "t3_" prefix
	}

	return s.client.getPosts(ctx, s.Name, postOpts...)
}

// GetPostsAfter fetches posts from the subreddit that come after the specified post.
// This method will automatically fetch multiple pages as needed up to the specified limit.
// Set limit to 0 to fetch all available posts (use with caution).
func (s *Subreddit) GetPostsAfter(ctx context.Context, after *Post, limit int) ([]Post, error) {
	return s.client.getPosts(ctx, s.Name, WithAfter(after), WithLimit(limit))
}

// String returns a string representation of the Subreddit struct
func (s *Subreddit) String() string {
	if s == nil {
		return "Subreddit<nil>"
	}

	return fmt.Sprintf("Subreddit{Name: %q, Client: %v}",
		s.Name,
		s.client,
	)
}
