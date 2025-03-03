# Reddit Client

A simple and reliable Reddit API client written in Go. This client supports reading posts and comments from Reddit using their OAuth2 API.

## Features

- OAuth2 authentication
- Fetch posts from subreddits with pagination
- Retrieve comments for posts
- Configurable rate limiting
- Structured logging with slog
- Context support for timeouts and cancellation

## Installation

```bash
go get github.com/JohnPlummer/reddit-client
```

## Quick Start

```go
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
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file:", err)
    }

    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Create auth client
    auth, err := reddit.NewAuth(
        os.Getenv("REDDIT_CLIENT_ID"),
        os.Getenv("REDDIT_CLIENT_SECRET"),
        reddit.WithUserAgent("MyBot/1.0"),
    )
    if err != nil {
        log.Fatal("Failed to create auth client:", err)
    }

    // Create Reddit client
    client, err := reddit.NewClient(auth)
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }

    // Get posts from a subreddit
    posts, after, err := client.GetPosts(ctx, "golang", map[string]string{
        "limit": "10",
        "sort":  "new",
    })
    if err != nil {
        log.Fatal("Error getting posts:", err)
    }

    // Print posts
    fmt.Println("Latest posts from r/golang:")
    fmt.Println("---------------------------")
    for _, post := range posts {
        fmt.Println(post)
        fmt.Println("---------------------------")
    }

    // Get more posts using the after cursor
    if after != "" {
        morePosts, err := client.GetPostsAfter(ctx, "golang", &posts[len(posts)-1], 10)
        if err != nil {
            log.Fatal("Error getting more posts:", err)
        }
        fmt.Println("\nNext page of posts:")
        fmt.Println("---------------------------")
        for _, post := range morePosts {
            fmt.Println(post)
            fmt.Println("---------------------------")
        }
    }
}
```

## Configuration

Create a `.env` file with your Reddit API credentials:

```text
REDDIT_CLIENT_ID=your_client_id
REDDIT_CLIENT_SECRET=your_client_secret
```

To get these credentials:

1. Go to <https://www.reddit.com/prefs/apps>
2. Click "create another app..."
3. Select "script"
4. Fill in the required information

## Examples

The [examples](examples) directory contains two example implementations:

- [Basic Example](examples/basic): A simple example showing how to fetch and display posts from a subreddit.
- [Comprehensive Example](examples/comprehensive): A full-featured example demonstrating pagination, rate limiting, structured logging, and more advanced features.

Each example includes its own README and configuration files.

## Client Options

The client supports several configuration options:

```go
// Set a custom user agent
reddit.WithUserAgent("MyBot/1.0")

// Configure rate limiting (requests per minute and burst size)
reddit.WithRateLimit(60, 5)

// Set request timeout
reddit.WithTimeout(10 * time.Second)
```

## API Methods

### GetPosts

Fetches a single page of posts from a subreddit. Returns posts and a cursor for pagination.

```go
posts, after, err := client.GetPosts(ctx, "golang", map[string]string{
    "limit": "25",  // Number of posts to fetch (max 100)
    "sort": "new",  // Sort order (new, hot, top, etc.)
})
```

### GetPostsAfter

Fetches multiple pages of posts after a specific post. Useful for implementing infinite scroll or pagination.

```go
// Get posts after a specific post
lastPost := &Post{ID: "abc123"}
posts, err := client.GetPostsAfter(ctx, "golang", lastPost, 25)

// Or continue from previous GetPosts results
firstPagePosts, after, _ := client.GetPosts(ctx, "golang", nil)
if len(firstPagePosts) > 0 {
    nextPosts, err := client.GetPostsAfter(ctx, "golang", &firstPagePosts[len(firstPagePosts)-1], 25)
}

// Get all available posts (use with caution)
allPosts, err := client.GetPostsAfter(ctx, "golang", nil, 0)
```

### GetComments

Fetches comments for a post.

```go
// Get all comments for a post
post := &Post{ID: "abc123", Subreddit: "golang"}
comments, err := post.GetComments(ctx)

// With pagination using GetCommentsAfter
firstPageComments, err := post.GetComments(ctx)
if err == nil && len(firstPageComments) > 0 {
    // Get next page of comments after the last comment
    comment := firstPageComments[len(firstPageComments)-1]  // Get the last comment
    moreComments, err := post.GetCommentsAfter(ctx, &comment, 25)  // Pass its address
}

// Get all available comments starting from the beginning
allComments, err := post.GetCommentsAfter(ctx, nil, 0)

// Using functional options for more control
comments, err := post.GetComments(ctx,
    WithCommentLimit(50),
    WithCommentSort("top"),
    WithCommentDepth(5),
    WithCommentContext(3),
    WithCommentShowMore(true),
)
```

## License

MIT License - see [LICENSE](LICENSE) for details.
