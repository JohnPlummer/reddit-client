package prompts_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/JohnPlummer/reddit-client/scorer/prompts"
)

var _ = Describe("BrightonRecommendationPrompt", func() {
	var (
		builder prompts.PromptBuilder
		params  prompts.PromptParams
	)

	BeforeEach(func() {
		builder = prompts.NewBrightonRecommendationPrompt()
	})

	Describe("Build", func() {
		Context("with a single item", func() {
			BeforeEach(func() {
				params = prompts.PromptParams{
					ScoreRange: [2]int{0, 10},
					ItemCount:  1,
					Items: []prompts.ItemToScore{
						{
							ID:      "123",
							Title:   "Test Post",
							Content: "Test Content",
						},
					},
				}
			})

			It("should include required elements in system message", func() {
				prompt := builder.Build(params)
				systemMsg := prompt.SystemMessage

				expectedPhrases := []string{
					"score EVERY post provided",
					"post_id, title, and score fields",
					"match the number of input posts exactly",
					"activities, places, or experiences in Brighton",
				}

				for _, phrase := range expectedPhrases {
					Expect(systemMsg).To(ContainSubstring(phrase))
				}
			})

			It("should include required elements in user message", func() {
				prompt := builder.Build(params)
				userMsg := prompt.UserMessage

				expectedPhrases := []string{
					"analyze ALL of the following Reddit posts",
					"score each one on the likelihood",
					"from 0 (not likely) to 10 (extremely likely)",
					"[ID: 123] Test Post",
					"Content: Test Content",
				}

				for _, phrase := range expectedPhrases {
					Expect(userMsg).To(ContainSubstring(phrase))
				}
			})
		})

		Context("with multiple items and custom message", func() {
			BeforeEach(func() {
				params = prompts.PromptParams{
					ScoreRange: [2]int{1, 5},
					ItemCount:  2,
					Items: []prompts.ItemToScore{
						{
							ID:    "123",
							Title: "First Post",
						},
						{
							ID:      "456",
							Title:   "Second Post",
							Content: "With Content",
						},
					},
					CustomMessage: "Focus on family-friendly activities",
				}
			})

			It("should include all items and custom message", func() {
				prompt := builder.Build(params)
				userMsg := prompt.UserMessage

				expectedPhrases := []string{
					"from 1 (not likely) to 5 (extremely likely)",
					"exactly 2 objects",
					"Focus on family-friendly activities",
					"[ID: 123] First Post",
					"[ID: 456] Second Post",
					"Content: With Content",
				}

				for _, phrase := range expectedPhrases {
					Expect(userMsg).To(ContainSubstring(phrase))
				}
			})
		})

		Context("with empty content", func() {
			BeforeEach(func() {
				params = prompts.PromptParams{
					ScoreRange: [2]int{0, 10},
					ItemCount:  1,
					Items: []prompts.ItemToScore{
						{
							ID:      "123",
							Title:   "Test Post",
							Content: "",
						},
					},
				}
			})

			It("should not include content section", func() {
				prompt := builder.Build(params)
				userMsg := prompt.UserMessage

				Expect(userMsg).To(ContainSubstring("[ID: 123] Test Post"))
				Expect(userMsg).NotTo(ContainSubstring("Content:"))
			})
		})
	})
})
