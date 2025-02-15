# Basic Reddit Client Example

This example demonstrates how to use the Reddit client library to fetch posts and comments from a subreddit.

## Features

- Fetches posts from a specified subreddit with pagination
- Retrieves comments for each post
- Configurable rate limiting
- Structured logging
- Command line flags for customization

## Prerequisites

1. Create a Reddit application at <https://www.reddit.com/prefs/apps>
2. Create a `.env` file in this directory with your Reddit API credentials:

```env
REDDIT_CLIENT_ID=your_client_id
REDDIT_CLIENT_SECRET=your_client_secret
```

## Usage

Run the example with default settings:

```bash
go run main.go
```

### Available Flags

- `-subreddit string`: Subreddit to fetch posts from (default "brighton")
- `-limit int`: Number of posts per page (default 10)
- `-sort string`: Sort order (new, hot, top, rising) (default "new")
- `-timeframe string`: Timeframe for top posts (hour, day, week, month, year, all) (default "all")
- `-max-pages int`: Maximum number of pages to fetch (default 1)
- `-log-level string`: Log level (debug, info, warn, error) (default "info")
- `-rate-limit int`: Rate limit in requests per second (default 1)
- `-rate-burst int`: Maximum burst size for rate limiting (default 5)
- `-timeout duration`: Read timeout duration (default 30s)

### Examples

Fetch 50 top posts from r/golang from the past week:

```bash
go run main.go -subreddit golang -sort top -timeframe week -limit 50 -max-pages 1
```

Fetch 100 new posts from r/news with increased rate limit:

```bash
go run main.go -subreddit news -sort new -limit 25 -max-pages 4 -rate-limit 2 -rate-burst 10
```

Enable debug logging:

```bash
go run main.go -log-level debug
```
