package reddit

import (
	"fmt"
	"time"
)

// Comment represents a single comment on a Reddit post
type Comment struct {
	Author     string `json:"author"`
	Body       string `json:"body"`
	Created    int64  `json:"created_utc"`
	ID         string `json:"id"`
	IngestedAt int64  `json:"-"` // When we stored it, not from Reddit API
}

// Fullname returns the Reddit fullname identifier for this comment (t1_<id>)
func (c Comment) Fullname() string {
	return "t1_" + c.ID
}

// parseComments extracts comments from the API response
func parseComments(data []any) ([]Comment, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("comment.parseComments: unexpected response format")
	}

	commentData, ok := data[1].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("comment.parseComments: unexpected response format")
	}

	var comments []Comment
	dataMap, ok := commentData["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("comment.parseComments: invalid data structure")
	}

	children, ok := dataMap["children"].([]any)
	if !ok {
		return nil, fmt.Errorf("comment.parseComments: missing children array")
	}
	now := nowUnix()

	for _, item := range children {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue // Skip invalid items
		}

		commentBody, ok := itemMap["data"].(map[string]any)
		if !ok {
			continue // Skip invalid comment data
		}

		// Use type-safe field extractors
		comment, err := parseCommentData(commentBody, now)
		if err != nil {
			continue // Skip comments with missing essential data
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

// Helper function to get current time in Unix seconds
func nowUnix() int64 {
	return time.Now().UTC().Unix()
}

// String returns a formatted string representation of the Comment
func (c Comment) String() string {
	return fmt.Sprintf(
		"Comment{\n"+
			"    Author: %q\n"+
			"    Body: %q\n"+
			"    Created: %d\n"+
			"    ID: %q\n"+
			"    IngestedAt: %d\n"+
			"}",
		c.Author,
		c.Body,
		c.Created,
		c.ID,
		c.IngestedAt,
	)
}
