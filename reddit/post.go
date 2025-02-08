package reddit

import "fmt"

// Post represents a Reddit post with relevant fields.
type Post struct {
	Title     string `json:"title"`
	SelfText  string `json:"selftext"`
	URL       string `json:"url"`
	Created   int64  `json:"created_utc"`
	Subreddit string `json:"subreddit"`
	ID        string `json:"id"`
	Score     int    `json:"score"`
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
			"    Score: %d\n"+
			"}",
		p.Title,
		p.SelfText,
		p.URL,
		p.Created,
		p.Subreddit,
		p.ID,
		p.Score,
	)
}

// Print formats and prints a slice of posts to stdout
func Print(posts []Post) {
	fmt.Println("Posts:")
	for i, post := range posts {
		fmt.Printf("Post %d:\n%s\n", i+1, post)
	}
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

	return Post{
		Title:     title,
		SelfText:  selfText,
		URL:       url,
		Created:   int64(created),
		Subreddit: subreddit,
		ID:        id,
		Score:     int(score),
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

// SubredditOption is a function type for modifying subreddit request parameters
type SubredditOption func(params map[string]string)

// GetSubreddit fetches a specific number of posts using pagination.
func GetSubreddit(c *Client, subreddit, sort string, totalPosts int, opts ...SubredditOption) ([]Post, error) {
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

		posts, nextPage, err := c.getPosts(subreddit, params)
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

// Since returns a SubredditOption that filters posts created after the given timestamp
func Since(timestamp int64) SubredditOption {
	return func(params map[string]string) {
		params["after"] = fmt.Sprintf("t3_%d", timestamp)
	}
}
