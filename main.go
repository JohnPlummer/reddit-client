package main

import (
	"fmt"
	"log"
	"os"

	"github.com/JohnPlummer/reddit-client/reddit"
	"github.com/JohnPlummer/reddit-client/scorer"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Check for required environment variables
	requiredEnvVars := []string{"REDDIT_CLIENT_ID", "REDDIT_CLIENT_SECRET", "OPENAI_API_KEY"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatalf("%s environment variable is not set", envVar)
		}
	}

	auth := &reddit.Auth{
		ClientID:     os.Getenv("REDDIT_CLIENT_ID"),
		ClientSecret: os.Getenv("REDDIT_CLIENT_SECRET"),
	}

	if err := auth.Authenticate(); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	redditClient := reddit.NewClient(auth, reddit.WithUserAgent("MyBot/0.0.1"))

	// Get posts from r/brighton
	subreddit := reddit.Subreddit{Name: "brighton"}
	posts, err := subreddit.GetPosts(redditClient, "new", 25) // Fetch 25 newest posts
	if err != nil {
		log.Fatal(err)
	}

	// Create dependencies for the scorer
	config := scorer.NewConfig(os.Getenv("OPENAI_API_KEY"))
	openaiClient := openai.NewClient(config.GetAPIKey())

	// Create the scorer with dependencies
	postScorer := scorer.NewChatGPTScorer(openaiClient, config)

	// Score the posts
	scores, err := postScorer.ScorePosts(posts)
	if err != nil {
		log.Fatal(err)
	}

	// Display results, sorted by score (highest first)
	fmt.Println("\nScored Posts from r/brighton (sorted by relevance to things to do):")
	fmt.Println("----------------------------------------")

	// Sort scores by score value (highest first)
	for i := 10; i >= 0; i-- {
		for _, score := range scores {
			if score.Score == i {
				fmt.Printf("\nScore: %d/10\n", score.Score)
				fmt.Printf("Title: %s\n", score.Title)
				fmt.Printf("Post ID: %s\n", score.PostID)
				fmt.Println("----------------------------------------")
			}
		}
	}
}
