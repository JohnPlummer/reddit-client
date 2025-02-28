package reddit

import (
	"context"
	"fmt"
)

// PostGetter defines the interface for fetching posts from Reddit
type PostGetter interface {
	GetPosts(subreddit string, params map[string]string) ([]Post, string, error)
}

// Subreddit represents a Reddit subreddit
type Subreddit struct {
	Name   string
	client *Client
}

// SubredditOption is a function type for modifying subreddit request parameters
type SubredditOption func(params map[string]string)

// NewSubreddit creates a new Subreddit instance
func NewSubreddit(name string, client *Client) *Subreddit {
	return &Subreddit{
		Name:   name,
		client: client,
	}
}

// GetPosts fetches posts from the subreddit
func (s *Subreddit) GetPosts(ctx context.Context, sort string, totalPosts int, opts ...SubredditOption) ([]Post, error) {
	params := map[string]string{
		"limit": "100",
		"sort":  sort,
	}

	for _, opt := range opts {
		opt(params)
	}

	var allPosts []Post
	after := ""

	for len(allPosts) < totalPosts {
		if after != "" {
			params["after"] = after
		}

		posts, nextPage, err := s.client.getPosts(ctx, s.Name, params)
		if err != nil {
			return nil, err
		}

		allPosts = append(allPosts, posts...)
		if nextPage == "" || len(posts) == 0 {
			break
		}
		after = nextPage
	}

	if len(allPosts) > totalPosts {
		allPosts = allPosts[:totalPosts]
	}

	return allPosts, nil
}

// GetPostsAfter fetches posts from the subreddit that come after the specified post
func (s *Subreddit) GetPostsAfter(ctx context.Context, after *Post, limit int) ([]Post, error) {
	return s.client.getPostsAfter(ctx, s.Name, after, limit)
}

// Since returns a SubredditOption that filters posts created after the given timestamp
func Since(timestamp int64) SubredditOption {
	return func(params map[string]string) {
		params["after"] = fmt.Sprintf("t3_%d", timestamp)
	}
}
