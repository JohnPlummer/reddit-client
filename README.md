# Reddit API Client (Go)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A simple **Golang client** for fetching posts and comments from Reddit using **OAuth2 (client credentials flow).**

## 🚀 Features

- Authenticate with Reddit using **app-only authentication** (OAuth2)
- Fetch **subreddit posts** with different sorting options
- Retrieve **comments** for a specific post

## 🛠 Installation

Clone the repository:

```sh
git clone git@github.com:JohnPlummer/reddit-client.git
cd reddit-client
go mod tidy
```

## ⚙️ Configuration

Create a `.env` file with your **Reddit API credentials**:

```ini
REDDIT_CLIENT_ID=your-client-id
REDDIT_CLIENT_SECRET=your-client-secret
```

## ▶️ Usage

Run the client to fetch posts from the **golang** subreddit:

```sh
go run main.go
```

### **Sorting Options**

By default, the client fetches **hot** posts, but you can specify different sorting methods:

| Sort Type  | API Parameter | Description                            |
| ---------- | ------------- | -------------------------------------- |
| **Hot**    | `hot`         | Most popular posts right now (default) |
| **New**    | `new`         | Latest posts                           |
| **Top**    | `top`         | Highest-voted posts                    |
| **Rising** | `rising`      | Posts that are trending                |

#### **Example: Fetch New Posts**

Modify `main.go`:

```go
posts, err := client.GetSubredditPosts("golang", "new") // Change "new" to "top", "rising", or "hot"
```

## 📝 License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.
