package prompts

// Prompt represents a complete prompt with system and user messages
type Prompt struct {
	SystemMessage string
	UserMessage   string
}

// PromptBuilder defines the interface for building prompts
type PromptBuilder interface {
	Build(params PromptParams) Prompt
}

// PromptParams holds parameters needed to build a prompt
type PromptParams struct {
	ScoreRange    [2]int // min and max score
	ItemCount     int    // number of items to be scored
	Items         []ItemToScore
	CustomMessage string // optional custom message to include
}

// ItemToScore represents an item that needs to be scored
type ItemToScore struct {
	ID      string
	Title   string
	Content string
}
