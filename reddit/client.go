package reddit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	Auth *Auth
}

func (c *Client) request(endpoint string) (*http.Response, error) {
	if c.Auth.Token == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	req, _ := http.NewRequest("GET", "https://oauth.reddit.com"+endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+c.Auth.Token)
	req.Header.Set("User-Agent", "golang:reddit-client:v1.0 (by /u/yourusername)")

	client := &http.Client{}
	return client.Do(req)
}

// ExtractPostID extracts the post ID from a Reddit post URL.
func ExtractPostID(postURL string) (string, error) {
	parts := strings.Split(postURL, "/")
	for i, part := range parts {
		if part == "comments" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}
	return "", fmt.Errorf("failed to extract post ID from URL")
}

// Post represents a Reddit post with relevant fields.
type Post struct {
	Title    string `json:"title"`
	SelfText string `json:"selftext"`
	URL      string `json:"url"`
}

// GetSubredditPosts fetches posts from a subreddit.
func (c *Client) GetSubredditPosts(subreddit string) ([]Post, error) {
	resp, err := c.request(fmt.Sprintf("/r/%s/hot", subreddit))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	var posts []Post
	for _, item := range data["data"].(map[string]interface{})["children"].([]interface{}) {
		postData := item.(map[string]interface{})["data"].(map[string]interface{})
		post := Post{
			Title:    postData["title"].(string),
			SelfText: postData["selftext"].(string), // Empty for link posts
			URL:      postData["url"].(string),
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// Comment represents a single comment on a Reddit post.
type Comment struct {
	Author string `json:"author"`
	Body   string `json:"body"`
}

// GetPostComments fetches comments for a given post.
func (c *Client) GetPostComments(subreddit, postID string) ([]Comment, error) {
	resp, err := c.request(fmt.Sprintf("/r/%s/comments/%s", subreddit, postID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	// Comments are in the second element of the response
	commentData, ok := data[1].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	var comments []Comment
	for _, item := range commentData["data"].(map[string]interface{})["children"].([]interface{}) {
		commentBody := item.(map[string]interface{})["data"].(map[string]interface{})
		comments = append(comments, Comment{
			Author: commentBody["author"].(string),
			Body:   commentBody["body"].(string),
		})
	}

	return comments, nil
}
