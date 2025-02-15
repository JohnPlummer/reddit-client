package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/JohnPlummer/reddit-client/reddit"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		slog.Error("error loading .env file", "error", err)
		os.Exit(1)
	}

	// Initialize authentication
	auth := reddit.NewAuth(
		os.Getenv("REDDIT_CLIENT_ID"),
		os.Getenv("REDDIT_CLIENT_SECRET"),
	)
	if auth == nil {
		slog.Error("failed to create auth client")
		os.Exit(1)
	}

	// Create a new client
	client, err := reddit.NewClient(auth, reddit.WithUserAgent("MyRedditBot/1.0"))
	if err != nil {
		slog.Error("failed to create client", "error", err)
		os.Exit(1)
	}

	// Parameters for the request
	params := map[string]string{
		"limit": "10",
		"sort":  "new",
	}

	// Get the 10 most recent posts from r/brighton
	posts, _, err := client.GetPosts("brighton", params)
	if err != nil {
		slog.Error("Error getting posts", "error", err)
		os.Exit(1)
	}

	// Print each post and its comments
	for i, post := range posts {
		fmt.Printf("\n=== Post %d ===\n", i+1)
		fmt.Println(post.String())

		// Get comments for this post using the Post's GetComments method
		comments, err := post.GetComments(client)
		if err != nil {
			slog.Error("Error getting comments for post", "post_id", post.ID, "error", err)
			continue
		}

		fmt.Printf("\nComments (%d):\n", len(comments))
		for _, comment := range comments {
			fmt.Println(comment.String())
			fmt.Println("-------------------")
		}
	}
}
