package scorer_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/JohnPlummer/reddit-client/reddit"
	"github.com/JohnPlummer/reddit-client/scorer"
	"github.com/JohnPlummer/reddit-client/scorer/prompts"
	openai "github.com/sashabaranov/go-openai"
)

type mockConfig struct {
	apiKey     string
	timeWindow time.Duration
}

func (m *mockConfig) GetAPIKey() string            { return m.apiKey }
func (m *mockConfig) GetTimeWindow() time.Duration { return m.timeWindow }

type mockOpenAIClient struct {
	createChatCompletionFunc func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

func (m *mockOpenAIClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	return m.createChatCompletionFunc(ctx, req)
}

var _ = Describe("ChatGPTScorer", func() {
	var (
		client *mockOpenAIClient
		config *mockConfig
		posts  []reddit.Post
		now    time.Time
	)

	BeforeEach(func() {
		now = time.Now()
		client = &mockOpenAIClient{}
		config = &mockConfig{
			apiKey:     "test-api-key",
			timeWindow: 24 * time.Hour,
		}

		posts = []reddit.Post{
			{
				ID:      "123",
				Title:   "Recent Post",
				Created: now.Unix(),
			},
			{
				ID:      "456",
				Title:   "Old Post",
				Created: now.Add(-48 * time.Hour).Unix(),
			},
		}
	})

	Describe("ScorePosts", func() {
		Context("with valid posts", func() {
			BeforeEach(func() {
				client.createChatCompletionFunc = func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					return openai.ChatCompletionResponse{
						Choices: []openai.ChatCompletionChoice{
							{
								Message: openai.ChatCompletionMessage{
									Content: `[{"post_id":"123","title":"Recent Post","score":5}]`,
								},
							},
						},
					}, nil
				}
			})

			It("should score only recent posts", func() {
				s := scorer.NewChatGPTScorer(client, config)

				scores, err := s.ScorePosts(posts)
				Expect(err).NotTo(HaveOccurred())
				Expect(scores).To(HaveLen(1))
				Expect(scores[0].PostID).To(Equal("123"))
				Expect(scores[0].Score).To(Equal(5))
			})
		})

		Context("with custom score range", func() {
			BeforeEach(func() {
				client.createChatCompletionFunc = func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					return openai.ChatCompletionResponse{
						Choices: []openai.ChatCompletionChoice{
							{
								Message: openai.ChatCompletionMessage{
									Content: `[{"post_id":"123","title":"Recent Post","score":3}]`,
								},
							},
						},
					}, nil
				}
			})

			It("should use the custom score range", func() {
				s := scorer.NewChatGPTScorer(client, config, scorer.WithScoreRange(1, 5))

				scores, err := s.ScorePosts(posts)
				Expect(err).NotTo(HaveOccurred())
				Expect(scores).To(HaveLen(1))
				Expect(scores[0].Score).To(BeNumerically(">=", 1))
				Expect(scores[0].Score).To(BeNumerically("<=", 5))
			})
		})

		Context("with custom prompt builder", func() {
			var customBuilder prompts.PromptBuilder

			BeforeEach(func() {
				customBuilder = &prompts.BrightonRecommendationPrompt{}
				client.createChatCompletionFunc = func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					return openai.ChatCompletionResponse{
						Choices: []openai.ChatCompletionChoice{
							{
								Message: openai.ChatCompletionMessage{
									Content: `[{"post_id":"123","title":"Recent Post","score":5}]`,
								},
							},
						},
					}, nil
				}
			})

			It("should use the custom prompt builder", func() {
				s := scorer.NewChatGPTScorer(client, config, scorer.WithPromptBuilder(customBuilder))

				scores, err := s.ScorePosts(posts)
				Expect(err).NotTo(HaveOccurred())
				Expect(scores).To(HaveLen(1))
			})
		})
	})
})
