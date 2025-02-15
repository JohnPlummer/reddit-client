# Basic Reddit Client Example

This is a minimal example demonstrating how to use the Reddit client library to fetch the latest posts from a subreddit.

## Prerequisites

1. Create a Reddit application at <https://www.reddit.com/prefs/apps>
2. Create a `.env` file in this directory with your Reddit API credentials:

```env
REDDIT_CLIENT_ID=your_client_id
REDDIT_CLIENT_SECRET=your_client_secret
```

## Usage

Run the example:

```bash
go run main.go
```

This will fetch and display the 5 most recent posts from r/golang.
