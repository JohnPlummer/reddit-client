package scorer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	"github.com/JohnPlummer/reddit-client/scorer/prompts"
	openai "github.com/sashabaranov/go-openai"
)

// ChatGPTScorer implements PostScorer using OpenAI's ChatGPT
type ChatGPTScorer struct {
	client        OpenAIClient
	config        ScorerConfig
	promptBuilder prompts.PromptBuilder
	scoreRange    [2]int
}

// ChatGPTScorerOption defines a function type for configuring a ChatGPTScorer
type ChatGPTScorerOption func(*ChatGPTScorer)

// WithScoreRange returns an option to set the score range
func WithScoreRange(min, max int) ChatGPTScorerOption {
	return func(s *ChatGPTScorer) {
		s.scoreRange = [2]int{min, max}
	}
}

// WithPromptBuilder returns an option to set the prompt builder
func WithPromptBuilder(builder prompts.PromptBuilder) ChatGPTScorerOption {
	return func(s *ChatGPTScorer) {
		s.promptBuilder = builder
	}
}

// NewChatGPTScorer creates a new ChatGPTScorer instance
func NewChatGPTScorer(client OpenAIClient, config ScorerConfig, opts ...ChatGPTScorerOption) *ChatGPTScorer {
	scorer := &ChatGPTScorer{
		client:        client,
		config:        config,
		promptBuilder: prompts.NewBrightonRecommendationPrompt(),
		scoreRange:    [2]int{0, 10},
	}

	for _, opt := range opts {
		opt(scorer)
	}

	return scorer
}

// ScorePosts implements the PostScorer interface
func (s *ChatGPTScorer) ScorePosts(posts []reddit.Post) ([]PostScore, error) {
	fmt.Printf("Total posts received: %d\n", len(posts))

	// Filter posts by time window
	cutoff := time.Now().Add(-s.config.GetTimeWindow())
	var recentPosts []reddit.Post
	for _, post := range posts {
		postTime := time.Unix(post.Created, 0)
		if postTime.After(cutoff) {
			recentPosts = append(recentPosts, post)
		}
	}

	fmt.Printf("Posts within time window: %d\n", len(recentPosts))
	fmt.Printf("Time window: %v\n", s.config.GetTimeWindow())
	fmt.Printf("Cutoff time: %v\n", cutoff)

	if len(recentPosts) == 0 {
		return nil, fmt.Errorf("no posts found within the last %v", s.config.GetTimeWindow())
	}

	// Convert posts to ItemToScore
	items := make([]prompts.ItemToScore, len(recentPosts))
	for i, p := range recentPosts {
		items[i] = prompts.ItemToScore{
			ID:      p.ID,
			Title:   p.Title,
			Content: p.SelfText,
		}
	}

	// Build the prompt using the prompt builder
	prompt := s.promptBuilder.Build(prompts.PromptParams{
		ScoreRange: s.scoreRange,
		ItemCount:  len(recentPosts),
		Items:      items,
	})

	// Create chat completion request
	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: prompt.SystemMessage,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt.UserMessage,
			},
		},
	}

	resp, err := s.client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("ChatCompletion error: %w", err)
	}

	// Parse ChatGPT response
	var scores []PostScore
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &scores)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w\nRaw response: %s", err, resp.Choices[0].Message.Content)
	}

	fmt.Printf("Scores returned by ChatGPT: %d\n", len(scores))

	return scores, nil
}
