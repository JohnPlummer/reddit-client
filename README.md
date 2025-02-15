# Reddit Client

A simple and reliable Reddit API client written in Go. This client supports reading posts and comments from Reddit using their OAuth2 API.

## Features

- OAuth2 authentication
- Fetch posts from subreddits
- Retrieve comments for posts
- Rate limit handling
- Structured logging

## Installation

```bash
go get github.com/JohnPlummer/reddit-client
```

## Quick Start

```go
package main

import (
    "github.com/JohnPlummer/reddit-client/reddit"
    "github.com/joho/godotenv"
)

func main() {
    // Load environment variables
    godotenv.Load()

    // Create auth client
    auth := reddit.NewAuth(
        os.Getenv("REDDIT_CLIENT_ID"),
        os.Getenv("REDDIT_CLIENT_SECRET"),
    )

    // Create Reddit client
    client, err := reddit.NewClient(auth, reddit.WithUserAgent("MyBot/1.0"))
    if err != nil {
        panic(err)
    }

    // Get posts from a subreddit
    posts, _, err := client.GetPosts("golang", map[string]string{
        "limit": "10",
        "sort": "new",
    })
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

See the [examples](examples) directory for complete usage examples.

## License

MIT License - see [LICENSE](LICENSE) for details.
