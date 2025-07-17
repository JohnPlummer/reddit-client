package reddit

import (
	"context"
	"fmt"
)

// Post represents a Reddit post with relevant fields.
type Post struct {
	Title        string        `json:"title"`
	SelfText     string        `json:"selftext"`
	URL          string        `json:"url"`
	Created      int64         `json:"created_utc"`
	Subreddit    string        `json:"subreddit"`
	ID           string        `json:"id"`
	RedditScore  int           `json:"score"` // Reddit's upvotes minus downvotes
	ContentScore int           `json:"-"`     // Our custom content-based score
	CommentCount int           `json:"num_comments"`
	Comments     []Comment     `json:"comments,omitempty"`
	client       commentGetter // interface for fetching comments (should hold a pointer to the client)
}

// commentGetter interface for fetching comments (private interface)
//
//go:generate mockgen -source=post.go -destination=mocks/comment_getter_mock.go -package=mocks
type commentGetter interface {
	getComments(ctx context.Context, subreddit, postID string, opts ...CommentOption) ([]any, error)
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
func parsePost(item any, client commentGetter) (Post, error) {
	postMap, ok := item.(map[string]any)
	if !ok {
		return Post{}, fmt.Errorf("post.parsePost: invalid post format")
	}

	data, ok := postMap["data"].(map[string]any)
	if !ok {
		return Post{}, fmt.Errorf("post.parsePost: invalid post data format")
	}

	// Use type-safe field extractors
	post, err := parsePostData(data)
	if err != nil {
		return Post{}, fmt.Errorf("post.parsePost: %w", err)
	}

	// Set the client for comment fetching
	post.client = client
	return post, nil
}

// parsePosts extracts posts and the pagination cursor from API response.
func parsePosts(data map[string]any, client commentGetter) ([]Post, string, error) {
	var posts []Post

	listing, ok := data["data"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("post.parsePosts: invalid response format missing data object")
	}

	children, ok := listing["children"].([]any)
	if !ok {
		return nil, "", fmt.Errorf("post.parsePosts: invalid response format missing children array")
	}

	for _, item := range children {
		post, err := parsePost(item, client)
		if err != nil {
			continue // Skip invalid posts instead of failing completely
		}
		posts = append(posts, post)
	}

	nextPage, _ := listing["after"].(string)
	return posts, nextPage, nil
}

// GetComments fetches comments for this post with optional filters
func (p *Post) GetComments(ctx context.Context, opts ...CommentOption) ([]Comment, error) {
	if p.client == nil {
		return nil, fmt.Errorf("post.GetComments: post has no associated client")
	}

	data, err := p.client.getComments(ctx, p.Subreddit, p.ID, opts...)
	if err != nil {
		return nil, fmt.Errorf("post.GetComments: fetching comments failed: %w", err)
	}
	return parseComments(data)
}

// GetCommentsAfter fetches comments that come after the specified comment.
// This method will automatically fetch multiple pages as needed up to the specified limit.
// Set limit to 0 to fetch all available comments (use with caution).
func (p *Post) GetCommentsAfter(ctx context.Context, after *Comment, limit int) ([]Comment, error) {
	if p.client == nil {
		return nil, fmt.Errorf("post.GetCommentsAfter: post has no associated client")
	}

	opts := []CommentOption{WithCommentLimit(100)}
	if after != nil {
		opts = append(opts, WithCommentAfter(after))
	}

	var allComments []Comment
	for {
		data, err := p.client.getComments(ctx, p.Subreddit, p.ID, opts...)
		if err != nil {
			return nil, fmt.Errorf("post.GetCommentsAfter: fetching comments failed: %w", err)
		}

		comments, err := parseComments(data)
		if err != nil {
			return nil, fmt.Errorf("post.GetCommentsAfter: parsing comments failed: %w", err)
		}

		allComments = append(allComments, comments...)

		// Stop if we've reached the desired limit
		if limit > 0 && len(allComments) >= limit {
			allComments = allComments[:limit]
			break
		}

		// Stop if there are no more comments
		if len(comments) == 0 {
			break
		}

		// Update the after parameter for the next request
		lastComment := comments[len(comments)-1]
		opts = []CommentOption{WithCommentLimit(100), WithCommentAfter(&lastComment)}
	}

	return allComments, nil
}

// Fullname returns the Reddit fullname identifier for this post (t3_<id>)
func (p Post) Fullname() string {
	return "t3_" + p.ID
}
