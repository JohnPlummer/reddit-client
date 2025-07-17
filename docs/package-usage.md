# Package Usage

## Installation

```bash
go get github.com/JohnPlummer/reddit-client
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JohnPlummer/reddit-client/reddit"
)

func main() {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Create auth client
    auth, err := reddit.NewAuth(
        os.Getenv("REDDIT_CLIENT_ID"),
        os.Getenv("REDDIT_CLIENT_SECRET"),
        reddit.WithUserAgent("MyApp/1.0"),
    )
    if err != nil {
        log.Fatal("Failed to create auth client:", err)
    }

    // Create Reddit client
    client, err := reddit.NewClient(auth)
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }

    // Create subreddit instance
    subreddit := reddit.NewSubreddit("golang", client)

    // Fetch posts
    posts, err := subreddit.GetPosts(ctx, 
        reddit.WithSort("hot"),
        reddit.WithSubredditLimit(10),
    )
    if err != nil {
        log.Fatal("Error getting posts:", err)
    }

    // Process posts
    for _, post := range posts {
        fmt.Printf("Title: %s\n", post.Title)
        fmt.Printf("Score: %d\n", post.RedditScore)
        fmt.Printf("Comments: %d\n", post.CommentCount)
        fmt.Println("---")
    }
}
```

## Authentication

### Creating an Auth Instance

```go
// Basic authentication
auth, err := reddit.NewAuth(clientID, clientSecret)

// With custom options
auth, err := reddit.NewAuth(
    clientID,
    clientSecret,
    reddit.WithUserAgent("MyBot/1.0"),
    reddit.WithAuthTimeout(15*time.Second),
)
```

### Auth Configuration Options

```go
// Set custom user agent
reddit.WithUserAgent("MyBot/1.0")

// Set authentication timeout
reddit.WithAuthTimeout(15*time.Second)

// Use custom HTTP client
reddit.WithAuthHTTPClient(customClient)
```

## Client Configuration

### Creating a Client

```go
// Basic client
client, err := reddit.NewClient(auth)

// With custom options
client, err := reddit.NewClient(
    auth,
    reddit.WithUserAgent("MyBot/1.0"),
    reddit.WithRateLimit(30, 10), // 30 requests per minute, burst of 10
    reddit.WithTimeout(20*time.Second),
)
```

### Client Configuration Options

```go
// Set custom user agent
reddit.WithUserAgent("MyBot/1.0")

// Configure rate limiting (requests per minute, burst size)
reddit.WithRateLimit(60, 5)

// Set request timeout
reddit.WithTimeout(10*time.Second)

// Use custom HTTP client
reddit.WithHTTPClient(customClient)
```

## Fetching Posts

### Basic Post Fetching

```go
// Create subreddit instance
subreddit := reddit.NewSubreddit("programming", client)

// Get posts with default settings
posts, err := subreddit.GetPosts(ctx)

// Get posts with options
posts, err := subreddit.GetPosts(ctx,
    reddit.WithSort("new"),
    reddit.WithSubredditLimit(25),
    reddit.WithTimeFilter("day"),
)
```

### Post Fetching Options

```go
// Sort options: "hot", "new", "top", "rising"
reddit.WithSort("hot")

// Limit number of posts
reddit.WithSubredditLimit(50)

// Time filter for "top" sort: "hour", "day", "week", "month", "year", "all"
reddit.WithTimeFilter("week")
```

### Pagination

```go
// Get first page of posts
firstPage, err := subreddit.GetPosts(ctx, reddit.WithSubredditLimit(25))
if err != nil {
    log.Fatal(err)
}

// Get next page using the last post
if len(firstPage) > 0 {
    lastPost := firstPage[len(firstPage)-1]
    nextPage, err := subreddit.GetPostsAfter(ctx, &lastPost, 25)
    if err != nil {
        log.Fatal(err)
    }
}

// Get all posts (use with caution - can be many requests)
allPosts, err := subreddit.GetPostsAfter(ctx, nil, 0)
```

## Fetching Comments

### Basic Comment Fetching

```go
// Get comments for a post
comments, err := post.GetComments(ctx)

// Get comments with options
comments, err := post.GetComments(ctx,
    reddit.WithCommentLimit(100),
    reddit.WithCommentSort("top"),
    reddit.WithCommentDepth(5),
)
```

### Comment Fetching Options

```go
// Limit number of comments
reddit.WithCommentLimit(50)

// Sort comments: "confidence", "top", "new", "controversial", "old", "random", "qa"
reddit.WithCommentSort("top")

// Maximum depth of comment tree
reddit.WithCommentDepth(3)

// Context around target comment
reddit.WithCommentContext(2)

// Show more comments
reddit.WithCommentShowMore(true)
```

### Comment Pagination

```go
// Get first page of comments
firstPage, err := post.GetComments(ctx, reddit.WithCommentLimit(50))
if err != nil {
    log.Fatal(err)
}

// Get next page using the last comment
if len(firstPage) > 0 {
    lastComment := firstPage[len(firstPage)-1]
    nextPage, err := post.GetCommentsAfter(ctx, &lastComment, 50)
    if err != nil {
        log.Fatal(err)
    }
}

// Get all comments (use with caution)
allComments, err := post.GetCommentsAfter(ctx, nil, 0)
```

## Data Models

### Post Structure

```go
type Post struct {
    Title        string    // Post title
    SelfText     string    // Self-text content (for text posts)
    URL          string    // URL (for link posts)
    Created      int64     // Unix timestamp
    Subreddit    string    // Subreddit name
    ID           string    // Reddit post ID
    RedditScore  int       // Reddit score (upvotes - downvotes)
    ContentScore int       // Custom content score
    CommentCount int       // Number of comments
    Comments     []Comment // Loaded comments (if any)
}

// Get Reddit fullname (t3_<id>)
fullname := post.Fullname()
```

### Comment Structure

```go
type Comment struct {
    Author     string // Comment author
    Body       string // Comment content
    Created    int64  // Unix timestamp
    ID         string // Reddit comment ID
    IngestedAt int64  // When comment was fetched
}

// Get Reddit fullname (t1_<id>)
fullname := comment.Fullname()
```

## Error Handling

### Error Types

```go
// Check for specific error types
if reddit.IsRateLimitError(err) {
    // Handle rate limiting
    log.Println("Rate limited, backing off...")
}

if reddit.IsNotFoundError(err) {
    // Handle not found
    log.Println("Resource not found")
}

if reddit.IsServerError(err) {
    // Handle server errors
    log.Println("Reddit server error")
}
```

### Error Handling Best Practices

```go
posts, err := subreddit.GetPosts(ctx)
if err != nil {
    switch {
    case reddit.IsRateLimitError(err):
        // Implement exponential backoff
        time.Sleep(time.Minute)
        // Retry logic here
    case reddit.IsNotFoundError(err):
        // Subreddit doesn't exist or is private
        log.Printf("Subreddit not found: %s", subreddit.Name)
    case reddit.IsServerError(err):
        // Reddit is having issues
        log.Printf("Reddit server error: %v", err)
    default:
        // Other errors (network, parsing, etc.)
        log.Printf("Unexpected error: %v", err)
    }
    return
}
```

## Advanced Usage

### Custom HTTP Client

```go
// Create custom HTTP client with proxy
proxyURL, _ := url.Parse("http://proxy.example.com:8080")
customClient := &http.Client{
    Transport: &http.Transport{
        Proxy: http.ProxyURL(proxyURL),
    },
    Timeout: 30 * time.Second,
}

// Use custom client
client, err := reddit.NewClient(auth, reddit.WithHTTPClient(customClient))
```

### Concurrent Operations

```go
// Fetch posts from multiple subreddits concurrently
subreddits := []string{"golang", "programming", "webdev"}
results := make(chan []reddit.Post, len(subreddits))

for _, name := range subreddits {
    go func(subredditName string) {
        sub := reddit.NewSubreddit(subredditName, client)
        posts, err := sub.GetPosts(ctx, reddit.WithSubredditLimit(10))
        if err != nil {
            log.Printf("Error fetching from %s: %v", subredditName, err)
            results <- nil
            return
        }
        results <- posts
    }(name)
}

// Collect results
for i := 0; i < len(subreddits); i++ {
    posts := <-results
    if posts != nil {
        log.Printf("Got %d posts", len(posts))
    }
}
```

### Structured Logging

```go
import "log/slog"

// Configure structured logging
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
slog.SetDefault(logger)

// The library will use this logger for all operations
```

## Performance Considerations

### Rate Limiting

- Default: 60 requests per minute with burst of 5
- Automatically adjusts based on Reddit's response headers
- Configure with `WithRateLimit()` for your use case

### Memory Usage

- Posts and comments are loaded into memory
- For large datasets, process in batches
- Use pagination to control memory usage

### Context and Timeouts

- Always use context with timeouts
- Cancel operations that are no longer needed
- Context cancellation stops ongoing requests

```go
// Use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Use context with cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Cancel if user interrupts
go func() {
    // Handle interrupt signal
    cancel()
}()
```
