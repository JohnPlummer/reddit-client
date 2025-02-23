package reddit

import (
	"context"
	"fmt"
)

// Post represents a Reddit post with relevant fields.
type Post struct {
	Title        string    `json:"title"`
	SelfText     string    `json:"selftext"`
	URL          string    `json:"url"`
	Created      int64     `json:"created_utc"`
	Subreddit    string    `json:"subreddit"`
	ID           string    `json:"id"`
	RedditScore  int       `json:"score"` // Reddit's upvotes minus downvotes
	ContentScore int       `json:"-"`     // Our custom content-based score
	CommentCount int       `json:"num_comments"`
	Comments     []Comment `json:"comments,omitempty"`
}

// String returns a formatted string representation of the Post
func (p Post) String() string {
	return fmt.Sprintf(
		"Post{\n"+
			"    Title: %q\n"+
			"    SelfText: %q\n"+
			"    URL: %q\n"+
			"    Created: %d\n"+
			"    Subreddit: %q\n"+
			"    ID: %q\n"+
			"    RedditScore: %d\n"+
			"    ContentScore: %d\n"+
			"    CommentCount: %d\n"+
			"    Comments: %d\n"+
			"}",
		p.Title,
		p.SelfText,
		p.URL,
		p.Created,
		p.Subreddit,
		p.ID,
		p.RedditScore,
		p.ContentScore,
		p.CommentCount,
		len(p.Comments),
	)
}

// parsePost extracts a single post from the API response.
func parsePost(item interface{}) (Post, error) {
	postMap, ok := item.(map[string]interface{})
	if !ok {
		return Post{}, fmt.Errorf("invalid post format")
	}

	data, ok := postMap["data"].(map[string]interface{})
	if !ok {
		return Post{}, fmt.Errorf("invalid post data format")
	}

	// Safely extract fields with type assertions
	title, _ := data["title"].(string)
	selfText, _ := data["selftext"].(string)
	url, _ := data["url"].(string)
	created, _ := data["created_utc"].(float64)
	subreddit, _ := data["subreddit"].(string)
	id, _ := data["id"].(string)
	score, _ := data["score"].(float64)
	commentCount, _ := data["num_comments"].(float64)

	return Post{
		Title:        title,
		SelfText:     selfText,
		URL:          url,
		Created:      int64(created),
		Subreddit:    subreddit,
		ID:           id,
		RedditScore:  int(score),
		ContentScore: 0, // Initialize to 0, will be set by content analysis
		CommentCount: int(commentCount),
	}, nil
}

// parsePosts extracts posts and the pagination cursor from API response.
func parsePosts(data map[string]interface{}) ([]Post, string, error) {
	var posts []Post

	listing, ok := data["data"].(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid response format: missing data object")
	}

	children, ok := listing["children"].([]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid response format: missing children array")
	}

	for _, item := range children {
		post, err := parsePost(item)
		if err != nil {
			continue // Skip invalid posts instead of failing completely
		}
		posts = append(posts, post)
	}

	nextPage, _ := listing["after"].(string)
	return posts, nextPage, nil
}

// CommentOption is a function type for modifying comment request parameters
type CommentOption func(params map[string]string)

// CommentGetter interface for fetching comments
type CommentGetter interface {
	GetComments(ctx context.Context, subreddit, postID string, params map[string]string) ([]interface{}, error)
}

// GetComments fetches comments for this post with optional filters
func (p Post) GetComments(ctx context.Context, c CommentGetter, opts ...CommentOption) ([]Comment, error) {
	params := make(map[string]string)
	for _, opt := range opts {
		opt(params)
	}

	data, err := c.GetComments(ctx, p.Subreddit, p.ID, params)
	if err != nil {
		return nil, fmt.Errorf("fetching comments: %w", err)
	}
	return parseComments(data)
}

// CommentsSince returns a CommentOption that filters comments created after the given timestamp
func CommentsSince(timestamp int64) CommentOption {
	return func(params map[string]string) {
		params["after"] = fmt.Sprintf("t1_%d", timestamp)
	}
}
