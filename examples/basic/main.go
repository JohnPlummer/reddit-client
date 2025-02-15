package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize authentication
	auth, err := reddit.NewAuth(
		os.Getenv("REDDIT_CLIENT_ID"),
		os.Getenv("REDDIT_CLIENT_SECRET"),
		reddit.WithUserAgent("MyRedditBot/1.0"),
	)
	if err != nil {
		log.Fatal("Failed to create auth client:", err)
	}

	// Create a new client
	client, err := reddit.NewClient(auth)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Get latest posts from r/golang
	posts, _, err := client.GetPosts(ctx, "golang", map[string]string{
		"limit": "5",
		"sort":  "new",
	})
	if err != nil {
		log.Fatal("Error getting posts:", err)
	}

	// Print posts
	fmt.Println("Latest posts from r/golang:")
	fmt.Println("---------------------------")
	for _, post := range posts {
		fmt.Printf("\nTitle: %s\n", post.Title)
		fmt.Printf("Created: %d\n", post.Created)
		fmt.Printf("Score: %d\n", post.Score)
		fmt.Printf("URL: %s\n", post.URL)
		fmt.Println("---------------------------")
	}
}
