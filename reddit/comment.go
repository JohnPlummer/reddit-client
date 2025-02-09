package reddit

import (
	"fmt"
)

// TODO: Add Created and ID fields to Comment struct
// Comment represents a single comment on a Reddit post
type Comment struct {
	Author  string `json:"author"`
	Body    string `json:"body"`
	Created int64  `json:"created_utc"`
	ID      string `json:"id"`
}

// TODO: Implement these functions:
// - GetPostComments(c *Client, subreddit, postID string) ([]Comment, error)
// - GetCommentsSince(c *Client, timestamp int64) ([]Comment, error)
// - Print(comments []Comment) - pretty print comments similar to Post.Print()

// parseComments extracts comments from the API response
func parseComments(data []interface{}) ([]Comment, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("unexpected response format")
	}

	commentData, ok := data[1].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	var comments []Comment
	children := commentData["data"].(map[string]interface{})["children"].([]interface{})
	for _, item := range children {
		commentBody := item.(map[string]interface{})["data"].(map[string]interface{})
		created, _ := commentBody["created_utc"].(float64)
		comments = append(comments, Comment{
			Author:  commentBody["author"].(string),
			Body:    commentBody["body"].(string),
			Created: int64(created),
			ID:      commentBody["id"].(string),
		})
	}

	return comments, nil
}

// String returns a formatted string representation of the Comment
func (c Comment) String() string {
	return fmt.Sprintf(
		"Comment{\n"+
			"    Author: %q\n"+
			"    Body: %q\n"+
			"    Created: %d\n"+
			"    ID: %q\n"+
			"}",
		c.Author,
		c.Body,
		c.Created,
		c.ID,
	)
}
