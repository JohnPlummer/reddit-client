# Development Setup

## Prerequisites

### Go Installation

- Go 1.23.1 or later
- Verify installation: `go version`

### Reddit API Credentials

You need Reddit API credentials to use this library:

1. Go to <https://www.reddit.com/prefs/apps>
2. Click "create another app..."
3. Select "script" as the application type
4. Fill in the required information:
   - **Name**: Your application name
   - **Description**: Brief description of your app
   - **Redirect URI**: Not needed for client credentials flow
5. Note down the **Client ID** and **Client Secret**

## Installation

### As a Dependency

```bash
go get github.com/JohnPlummer/reddit-client
```

### For Development

```bash
git clone https://github.com/JohnPlummer/reddit-client
cd reddit-client
go mod download
```

## Configuration

### Environment Variables

Create a `.env` file in the project root or set these environment variables:

```bash
REDDIT_CLIENT_ID=your_client_id_here
REDDIT_CLIENT_SECRET=your_client_secret_here
```

### Configuration File Example

```go
// Load environment variables (optional)
if err := godotenv.Load(); err != nil {
    log.Fatal("Error loading .env file:", err)
}

// Create auth client
auth, err := reddit.NewAuth(
    os.Getenv("REDDIT_CLIENT_ID"),
    os.Getenv("REDDIT_CLIENT_SECRET"),
    reddit.WithUserAgent("YourApp/1.0"),
)
```

## Development Commands

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run tests directly with Ginkgo
ginkgo -v ./...
```

### Code Quality

```bash
# Format code
make lint

# Format all code (including examples)
make lint-all

# Run go fmt directly
go fmt ./...
```

### Building and Running

```bash
# Run basic example
make run-basic

# Run comprehensive example
make run-comprehensive

# Run both examples
make run-examples
```

### Maintenance

```bash
# Update dependencies
make tidy

# Update all dependencies (including examples)
make tidy-all

# Run all checks (tidy, lint, test, examples)
make check
```

### Mock Generation

```bash
# Generate mocks for testing
make generate-mocks

# Install mockgen if needed
make install-mockgen
```

## IDE Setup

### VS Code

Recommended extensions:

- Go (official Go extension)
- Ginkgo Test Explorer (for BDD tests)

### GoLand/IntelliJ

- Built-in Go support
- Ginkgo plugin for BDD testing

## Testing Framework

This project uses **Ginkgo** for BDD-style testing with **Gomega** matchers:

```go
// Example test structure
var _ = Describe("Client", func() {
    Context("when creating a new client", func() {
        It("should return a valid client", func() {
            client, err := reddit.NewClient(auth)
            Expect(err).ToNot(HaveOccurred())
            Expect(client).ToNot(BeNil())
        })
    })
})
```

## Mock Generation

Mocks are generated using `mockgen` with `go:generate` directives:

```go
//go:generate mockgen -source=post.go -destination=mocks/comment_getter_mock.go -package=mocks
```

## Debugging

### Enable Debug Logging

```go
// Set log level to debug
slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})))
```

### Common Issues

1. **Rate Limiting**: Respect Reddit's rate limits (60 requests per minute)
2. **Authentication**: Ensure credentials are correct and not expired
3. **Network Issues**: Check internet connectivity and Reddit API status

## Performance Considerations

### Rate Limiting

- Default: 60 requests per minute with burst of 5
- Configurable via `WithRateLimit(requestsPerMinute, burstSize)`
- Automatic adjustment based on Reddit's response headers

### Pagination

- Use `GetPostsAfter()` for efficient pagination
- Set reasonable limits to avoid excessive API calls
- Consider Reddit's maximum limits per request

### Memory Usage

- Comments and posts are loaded into memory
- For large datasets, implement streaming or chunked processing
- Use context cancellation for long-running operations
