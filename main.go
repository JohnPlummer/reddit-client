package main

import (
	"fmt"
	"log"
	"os"

	"github.com/JohnPlummer/reddit-client/reddit"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	auth := &reddit.Auth{
		ClientID:     os.Getenv("REDDIT_CLIENT_ID"),
		ClientSecret: os.Getenv("REDDIT_CLIENT_SECRET"),
	}

	if err := auth.Authenticate(); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	client := reddit.NewClient(auth, reddit.WithUserAgent("MyBot/0.0.1"))

	subreddit := reddit.Subreddit{Name: "brighton"}
	posts, err := subreddit.GetPosts(client, "new", 10)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Posts from r/golang:")
	for i, post := range posts {
		fmt.Printf("\nPost %d:\n%v\n", i+1, post)

		comments, err := post.GetComments(client)
		if err != nil {
			log.Printf("Error getting comments: %v\n", err)
			continue
		}

		fmt.Println("\nComments:")
		for j, comment := range comments {
			fmt.Printf("\nComment %d:\n%v\n", j+1, comment)
		}
		fmt.Println("----------------------------------------")
	}
}
