package main

import (
	"fmt"
	"log"
	"os"
	"time"

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

	// Example: Fetch 50 posts
	posts, err := reddit.GetSubreddit(client, "brighton", "new", 150)
	if err != nil {
		log.Fatalf("Error fetching posts: %v", err)
	}
	reddit.Print(posts)

	// Display the first post
	firstPost := posts[0]
	fmt.Println("Title:", firstPost.Title)
	fmt.Println("URL:", firstPost.URL)
	fmt.Println("\nFetching comments...\n")

	// Example: Fetch posts created in the last 24 hours
	oneDayAgo := time.Now().Unix() - 86400 // 24 hours ago
	recentPosts, err := reddit.GetSubreddit(client, "golang", "hot", 50, reddit.Since(oneDayAgo))
	if err != nil {
		log.Fatalf("Error fetching recent posts: %v", err)
	}
	reddit.Print(recentPosts)
}
