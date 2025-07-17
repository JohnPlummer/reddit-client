package reddit

import (
	"fmt"
	"strconv"
)

// getStringField safely extracts a string field from a map with optional default value
func getStringField(data map[string]any, key string, defaultValue ...string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// getFloat64Field safely extracts a float64 field from a map with optional default value
func getFloat64Field(data map[string]any, key string, defaultValue ...float64) float64 {
	if value, ok := data[key]; ok {
		switch v := value.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			// Attempt to parse string as float64
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				return parsed
			}
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0.0
}

// getBoolField safely extracts a boolean field from a map with optional default value
func getBoolField(data map[string]any, key string, defaultValue ...bool) bool {
	if value, ok := data[key]; ok {
		switch v := value.(type) {
		case bool:
			return v
		case string:
			// Attempt to parse string as boolean
			if parsed, err := strconv.ParseBool(v); err == nil {
				return parsed
			}
		case int:
			return v != 0
		case float64:
			return v != 0
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// getIntField safely extracts an int field from a map with optional default value and validation
func getIntField(data map[string]any, key string, defaultValue ...int) int {
	floatValue := getFloat64Field(data, key)
	if floatValue == 0.0 && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return int(floatValue)
}

// getInt64Field safely extracts an int64 field from a map with optional default value
func getInt64Field(data map[string]any, key string, defaultValue ...int64) int64 {
	floatValue := getFloat64Field(data, key)
	if floatValue == 0.0 && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return int64(floatValue)
}

// getValidatedIntField safely extracts an int field with validation (e.g., non-negative scores)
func getValidatedIntField(data map[string]any, key string, validator func(int) bool, defaultValue ...int) int {
	value := getIntField(data, key)
	if validator(value) {
		return value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// parsePostData safely extracts post data from API response using type-safe field extractors
func parsePostData(data map[string]any) (Post, error) {
	// Validate required fields
	id := getStringField(data, "id")
	if id == "" {
		return Post{}, fmt.Errorf("utils.parsePostData: missing required field 'id'")
	}

	// Extract fields with defaults and validation
	title := getStringField(data, "title")
	selfText := getStringField(data, "selftext")
	url := getStringField(data, "url")
	created := getInt64Field(data, "created_utc")
	subreddit := getStringField(data, "subreddit")

	// Validate score is non-negative (Reddit scores can be negative, but we want to catch parsing errors)
	score := getIntField(data, "score")
	commentCount := getValidatedIntField(data, "num_comments", func(v int) bool { return v >= 0 }, 0)

	return Post{
		Title:        title,
		SelfText:     selfText,
		URL:          url,
		Created:      created,
		Subreddit:    subreddit,
		ID:           id,
		RedditScore:  score,
		ContentScore: 0, // Initialize to 0, will be set by content analysis
		CommentCount: commentCount,
	}, nil
}

// parseCommentData safely extracts comment data from API response using type-safe field extractors
func parseCommentData(data map[string]any, ingestedAt int64) (Comment, error) {
	// Validate required fields
	id := getStringField(data, "id")
	if id == "" {
		return Comment{}, fmt.Errorf("utils.parseCommentData: missing required field 'id'")
	}

	// Extract fields with defaults
	author := getStringField(data, "author")
	body := getStringField(data, "body")
	created := getInt64Field(data, "created_utc")

	return Comment{
		Author:     author,
		Body:       body,
		Created:    created,
		ID:         id,
		IngestedAt: ingestedAt,
	}, nil
}
