# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Testing
- `make test` - Run all tests using Ginkgo test framework
- `ginkgo -v ./...` - Run tests directly with Ginkgo

### Linting and Formatting
- `make lint` - Run go fmt on root project 
- `make lint-all` - Run go fmt on entire codebase including examples
- `go fmt ./...` - Format code directly

### Building and Running
- `make run-basic` - Run the basic example
- `make run-comprehensive` - Run the comprehensive example with default parameters
- `make run-examples` - Run both examples

### Maintenance
- `make tidy` - Run go mod tidy on root project
- `make tidy-all` - Run go mod tidy on entire codebase
- `make check` - Run all checks: tidy, lint, test, and run examples

### Coverage and Mocks
- `make coverage` - Generate coverage report in markdown format
- `make generate-mocks` - Generate mocks using mockgen

## Architecture

This is a Go Reddit API client library that provides OAuth2 authentication and supports fetching posts and comments from Reddit.

### Core Components

**Authentication Layer (`reddit/auth.go`)**
- `Auth` struct handles OAuth2 client credentials flow
- Automatic token refresh when expired
- Rate limiting integration

**Client Layer (`reddit/client.go`)**
- `Client` struct is the main entry point for API operations
- Handles HTTP requests with automatic rate limiting
- Provides internal methods for fetching posts and comments

**Data Models**
- `Post` (`reddit/post.go`) - Represents Reddit posts with comment fetching capability
- `Comment` (`reddit/comment.go`) - Represents Reddit comments 
- `Subreddit` (`reddit/subreddit.go`) - Provides methods for fetching posts from specific subreddits

**Configuration**
- Functional options pattern used throughout for configuration
- Separate option types for Client, Auth, Post, Comment, and Subreddit operations

### Key Design Patterns

**Functional Options Pattern**
- All components use functional options for configuration
- Options defined in separate `*_options.go` files
- Allows for extensible, backward-compatible APIs

**Interface-Based Design**
- `commentGetter` interface allows Post to fetch comments
- `PostGetter` interface for subreddit post fetching 
- Enables easy mocking for testing

**Pagination Support**
- `GetPostsAfter` and `GetCommentsAfter` methods for pagination
- Automatic multi-page fetching with configurable limits

## Testing

- Uses Ginkgo BDD testing framework with Gomega matchers
- Mocks generated using `mockgen` for interfaces
- Test files follow `*_test.go` naming convention
- Test helper utilities in `http_test_helper.go`

## Code Standards

### From Cursor Rules
- Follow Go best practices: avoid `Get` prefixes, use clear naming
- Use structured logging with `slog` package
- Implement proper error handling with context
- Use interfaces for testability and dependency injection
- Follow functional options pattern for configuration

### Project-Specific Patterns
- All API methods accept `context.Context` as first parameter
- Rate limiting is handled automatically by the client
- Sensitive data (tokens, secrets) is obfuscated in String() methods
- Posts and comments include client references for related operations