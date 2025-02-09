package prompts

import "fmt"

// BrightonRecommendationPrompt builds prompts for scoring Brighton recommendations
type BrightonRecommendationPrompt struct{}

// NewBrightonRecommendationPrompt creates a new BrightonRecommendationPrompt
func NewBrightonRecommendationPrompt() *BrightonRecommendationPrompt {
	return &BrightonRecommendationPrompt{}
}

// Build implements the PromptBuilder interface
func (b *BrightonRecommendationPrompt) Build(params PromptParams) Prompt {
	systemMsg := "You are a helpful assistant that always responds with valid JSON arrays. " +
		"When scoring posts, you must score EVERY post provided and output a JSON array containing objects with post_id, title, and score fields. " +
		"The output array length must match the number of input posts exactly. " +
		"Score posts based on how likely they are to help visitors find interesting activities, places, or experiences in Brighton. " +
		"Do not include any other text or explanation in your response."

	userMsg := fmt.Sprintf("Please analyze ALL of the following Reddit posts from r/brighton and score each one on the likelihood that it recommends things to do in Brighton. "+
		"You MUST score EVERY post listed below. "+
		"Provide each score as an integer from %d (not likely) to %d (extremely likely). "+
		"Return a JSON array containing exactly %d objects (one for each post) in the following format: "+
		"[{\"post_id\": \"post_id_here\", \"title\": \"post title\", \"score\": 0}, ...].\n\n",
		params.ScoreRange[0], params.ScoreRange[1], params.ItemCount)

	if params.CustomMessage != "" {
		userMsg += params.CustomMessage + "\n\n"
	}

	userMsg += "Posts to score:\n"

	for idx, item := range params.Items {
		userMsg += fmt.Sprintf("%d. [ID: %s] %s\n", idx+1, item.ID, item.Title)
		if item.Content != "" {
			userMsg += fmt.Sprintf("Content: %s\n", item.Content)
		}
		userMsg += "\n"
	}

	return Prompt{
		SystemMessage: systemMsg,
		UserMessage:   userMsg,
	}
}
