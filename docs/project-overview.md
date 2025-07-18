# Project Overview

## Reddit Client Library

A robust and feature-rich Reddit API client library written in Go that provides OAuth2 authentication and comprehensive support for fetching posts and comments from Reddit.

### Project Type

**Library** - Go package designed for integration into other applications

### Core Purpose

This library simplifies interaction with Reddit's API by providing:

- OAuth2 client credentials authentication with automatic token refresh
- Type-safe methods for fetching posts and comments
- Built-in rate limiting and error handling
- Pagination support for large data sets
- Configurable options using the functional options pattern

### Key Features

- **OAuth2 Authentication**: Automatic token management with refresh capabilities
- **Subreddit Operations**: Fetch posts from any subreddit with filtering and sorting
- **Comment Retrieval**: Get comments for posts with configurable depth and limits
- **Pagination Support**: Efficient handling of large result sets with generic pagination utilities
- **Rate Limiting**: Intelligent rate limiting with header-based adjustments
- **Circuit Breaker**: Built-in circuit breaker pattern for resilience
- **Error Handling**: Comprehensive error types with retry logic and backoff strategies
- **Structured Logging**: Built-in logging using Go's `slog` package
- **Context Support**: Full context support for timeouts and cancellation
- **Interface-Based Design**: Easy mocking and testing capabilities
- **Performance Monitoring**: Built-in metrics and performance tracking

### Architecture Highlights

- **Functional Options Pattern**: Extensible configuration for all components
- **Interface-Based Design**: Easy testing and dependency injection
- **Error Handling**: Comprehensive error types with helper functions
- **Data Models**: Clean, well-structured types for posts and comments
- **Rate Limiting**: Intelligent rate limiting based on Reddit's response headers

### Target Use Cases

- Building Reddit bots and automation tools
- Data analysis and research applications
- Content aggregation services
- Social media monitoring tools
- Educational projects learning Reddit API integration

### Package Structure

```text
reddit/
├── auth.go                # OAuth2 authentication
├── client.go              # Main client implementation  
├── post.go                # Post data model and operations
├── comment.go             # Comment data model and operations
├── subreddit.go           # Subreddit operations
├── errors.go              # Error definitions and helpers
├── ratelimit.go           # Rate limiting implementation
├── circuit_breaker.go     # Circuit breaker implementation
├── pagination.go          # Generic pagination utilities
├── utils.go               # URL utilities and helpers
├── *_options.go           # Functional options for configuration
└── mocks/                # Generated mocks for testing
```

### Dependencies

- **Core Go**: 1.23.1+
- **Testing**: Ginkgo v2 (BDD testing framework)
- **Mocking**: golang/mock for generated mocks
- **Rate Limiting**: golang.org/x/time for rate limiting
- **HTTP**: Standard library with custom transport support

### Examples

The project includes comprehensive examples demonstrating various features:

- **Basic Example**: Simple post fetching demonstration
- **Comprehensive Example**: Advanced usage with all features
- **Performance Tuning Example**: Performance optimization techniques
- **Interceptors Example**: Request/response interceptors and middleware
