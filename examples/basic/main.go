package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	client, err := reddit.NewClient(auth, &http.Client{})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Create a subreddit instance
	subreddit := reddit.NewSubreddit("golang", client)

	// Get latest posts from r/golang
	posts, err := subreddit.GetPosts(ctx, "new", 5)
	if err != nil {
		log.Fatal("Error getting posts:", err)
	}

	// Print posts
	fmt.Println("Latest posts from r/golang:")
	fmt.Println("---------------------------")
	for _, post := range posts {
		fmt.Println(post.String())
		fmt.Println("---------------------------")
	}
}
