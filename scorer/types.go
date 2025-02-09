package scorer

import (
	"context"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	openai "github.com/sashabaranov/go-openai"
)

// PostScore represents a score assigned to a Reddit post
type PostScore struct {
	PostID string `json:"post_id"`
	Title  string `json:"title"`
	Score  int    `json:"score"`
}

// PostScorer defines the interface for scoring Reddit posts
type PostScorer interface {
	// ScorePosts takes a slice of Reddit posts and returns scores with any error
	ScorePosts(posts []reddit.Post) ([]PostScore, error)
}

// ScorerConfig defines the configuration interface for scorers
type ScorerConfig interface {
	GetAPIKey() string
	GetTimeWindow() time.Duration
}

// OpenAIClient defines the interface for interacting with OpenAI
type OpenAIClient interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

// ScorerOption defines a function type for configuring a scorer
type ScorerOption func(PostScorer) PostScorer

// Config implements ScorerConfig
type Config struct {
	apiKey     string
	timeWindow time.Duration
}

// NewConfig creates a new Config with default values
func NewConfig(apiKey string) *Config {
	return &Config{
		apiKey:     apiKey,
		timeWindow: 7 * 24 * time.Hour, // Default to 1 week
	}
}

// GetAPIKey returns the API key
func (c *Config) GetAPIKey() string {
	return c.apiKey
}

// GetTimeWindow returns the time window
func (c *Config) GetTimeWindow() time.Duration {
	return c.timeWindow
}

// WithTimeWindow sets a custom time window for filtering posts
func (c *Config) WithTimeWindow(d time.Duration) *Config {
	c.timeWindow = d
	return c
}
