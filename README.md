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
    posts, _, err := client.GetPosts(ctx, "golang", map[string]string{
        "limit": "10",
        "sort":  "new",
    })
    if err != nil {
        log.Fatal("Error getting posts:", err)
    }

    // Print posts
    for _, post := range posts {
        fmt.Printf("Title: %s\nScore: %d\nURL: %s\n\n", post.Title, post.Score, post.URL)
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

## License

MIT License - see [LICENSE](LICENSE) for details.
