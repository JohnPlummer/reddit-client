package main

import (
	"fmt"
	"log"
	"os"

	"github.com/JohnPlummer/reddit-client/reddit"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Retrieve credentials
	clientID := os.Getenv("REDDIT_CLIENT_ID")
	clientSecret := os.Getenv("REDDIT_CLIENT_SECRET")

	// Initialize Reddit client
	client, err := reddit.NewClient(clientID, clientSecret)
	if err != nil {
		log.Fatalf("Error initializing Reddit client: %v", err)
	}

	// Fetch posts from a subreddit
	posts, err := client.GetSubredditPosts("golang")
	if err != nil || len(posts) == 0 {
		log.Fatalf("Error fetching posts: %v", err)
	}

	// Display the first post
	firstPost := posts[0]
	fmt.Println("Title:", firstPost.Title)
	fmt.Println("URL:", firstPost.URL)
	fmt.Println("\nFetching comments...\n")

	// Extract post ID properly
	postID, err := reddit.ExtractPostID(firstPost.URL)
	if err != nil {
		log.Fatalf("Failed to extract post ID: %v", err)
	}

	// Fetch comments
	comments, err := client.GetPostComments("golang", postID)
	if err != nil {
		log.Fatalf("Error fetching comments: %v", err)
	}

	// Print comments
	for _, comment := range comments {
		fmt.Printf("%s: %s\n", comment.Author, comment.Body)
		fmt.Println("----")
	}
}
