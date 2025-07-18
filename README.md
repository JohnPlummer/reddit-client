# Reddit Client

[![Go Version](https://img.shields.io/github/go-mod/go-version/JohnPlummer/reddit-client)](https://golang.org/doc/devel/release.html)
[![Release](https://img.shields.io/github/v/release/JohnPlummer/reddit-client)](https://github.com/JohnPlummer/reddit-client/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

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
go get github.com/JohnPlummer/reddit-client@v0.9.0
```

Or for the latest version:

```bash
go get github.com/JohnPlummer/reddit-client@latest
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

    // Create a subreddit instance
    subreddit := reddit.NewSubreddit("golang", client)

    // Get posts from a subreddit using functional options
    posts, err := subreddit.GetPosts(ctx, 
        reddit.WithSort("new"),
        reddit.WithSubredditLimit(10),
    )
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

    // Get more posts using GetPostsAfter
    if len(posts) > 0 {
        morePosts, err := subreddit.GetPostsAfter(ctx, &posts[len(posts)-1], 10)
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

### Subreddit

#### Creating a Subreddit

```go
// Create a subreddit instance using the client
subreddit := reddit.NewSubreddit("golang", client)
```

#### GetPosts

Fetches posts from a subreddit with optional functional options.

```go
// Get posts from a subreddit using functional options
posts, err := subreddit.GetPosts(ctx, 
    reddit.WithSort("new"),
    reddit.WithSubredditLimit(10),
)
```

#### GetPostsAfter

Fetches posts that come after a specific post. Useful for implementing pagination.

```go
// Get posts after a specific post
lastPost := &Post{ID: "abc123"}
posts, err := subreddit.GetPostsAfter(ctx, lastPost, 25)

// Or continue from previous GetPosts results
firstPagePosts, err := subreddit.GetPosts(ctx)
if err == nil && len(firstPagePosts) > 0 {
    nextPosts, err := subreddit.GetPostsAfter(ctx, &firstPagePosts[len(firstPagePosts)-1], 25)
}

// Get all available posts (use with caution)
allPosts, err := subreddit.GetPostsAfter(ctx, nil, 0)
```

### Post

#### GetComments

Fetches comments for a post using functional options.

```go
// Get comments for a post
post := &Post{ID: "abc123", Subreddit: "golang"}
comments, err := post.GetComments(ctx)

// Using functional options for more control
comments, err := post.GetComments(ctx,
    reddit.WithCommentLimit(50),
    reddit.WithCommentSort("top"),
    reddit.WithCommentDepth(5),
    reddit.WithCommentContext(3),
    reddit.WithCommentShowMore(true),
)
```

#### GetCommentsAfter

Fetches comments that come after a specific comment. Useful for implementing pagination.

```go
// Get next page of comments after the last comment
firstPageComments, err := post.GetComments(ctx)
if err == nil && len(firstPageComments) > 0 {
    // Get the last comment
    lastComment := firstPageComments[len(firstPageComments)-1] 
    // Pass its address to GetCommentsAfter
    moreComments, err := post.GetCommentsAfter(ctx, &lastComment, 25)
}

// Get all available comments starting from the beginning
allComments, err := post.GetCommentsAfter(ctx, nil, 0)
```

## License

MIT License - see [LICENSE](LICENSE) for details.
