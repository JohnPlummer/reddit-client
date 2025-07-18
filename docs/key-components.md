# Key Components

## Architecture Overview

The Reddit client library is built around four main components that work together to provide a clean, type-safe interface to Reddit's API.

## Core Components

### 1. Authentication Layer (`auth.go`)

**Purpose**: Manages OAuth2 client credentials authentication with automatic token refresh.

**Key Features**:

- OAuth2 client credentials flow implementation
- Automatic token refresh when expired
- Secure credential management with obfuscation
- JSON wrapper for improved response handling
- Configurable timeouts and HTTP client

**Main Types**:

```go
type Auth struct {
    ClientID     string
    ClientSecret string
    Token        string
    ExpiresAt    time.Time
    // ... private fields
}
```

**Key Methods**:

- `NewAuth(clientID, clientSecret, ...opts)` - Create authentication instance
- `Authenticate(ctx)` - Perform OAuth2 authentication
- `EnsureValidToken(ctx)` - Check and refresh token if needed
- `IsTokenExpired()` - Check token expiration status

### 2. Client Layer (`client.go`)

**Purpose**: Main entry point for API operations with rate limiting and error handling.

**Key Features**:

- HTTP request management with automatic authentication
- Intelligent rate limiting with header-based adjustments
- Circuit breaker pattern for resilience
- Retry logic with exponential backoff
- Request/response interceptors
- Structured error handling and logging
- Context support for timeouts and cancellation

**Main Types**:

```go
type Client struct {
    Auth        *Auth
    userAgent   string
    client      *http.Client
    rateLimiter *RateLimiter
}
```

**Key Methods**:

- `NewClient(auth, ...opts)` - Create client instance
- `request(ctx, method, endpoint)` - Internal HTTP request handler
- `getPosts(ctx, subreddit, ...opts)` - Fetch posts with pagination
- `getComments(ctx, subreddit, postID, ...opts)` - Fetch comments

### 3. Data Models

#### Post Model (`post.go`)

**Purpose**: Represents Reddit posts with comment fetching capabilities.

```go
type Post struct {
    Title        string
    SelfText     string
    URL          string
    Created      int64
    Subreddit    string
    ID           string
    RedditScore  int
    ContentScore int
    CommentCount int
    Comments     []Comment
    client       commentGetter // Interface for dependency injection
}
```

**Key Methods**:

- `GetComments(ctx, ...opts)` - Fetch comments for this post
- `GetCommentsAfter(ctx, after, limit)` - Paginated comment fetching
- `Fullname()` - Get Reddit fullname identifier (`t3_<id>`)
- `String()` - Formatted string representation

#### Comment Model (`comment.go`)

**Purpose**: Represents Reddit comments with parsing and formatting.

```go
type Comment struct {
    Author     string
    Body       string
    Created    int64
    ID         string
    IngestedAt int64
}
```

**Key Methods**:

- `Fullname()` - Get Reddit fullname identifier (`t1_<id>`)
- `String()` - Formatted string representation

### 4. Subreddit Operations (`subreddit.go`)

**Purpose**: Provides methods for fetching posts from specific subreddits.

```go
type Subreddit struct {
    Name   string
    client *Client
}
```

**Key Methods**:

- `NewSubreddit(name, client)` - Create subreddit instance
- `GetPosts(ctx, ...opts)` - Fetch posts with options
- `GetPostsAfter(ctx, after, limit)` - Paginated post fetching

## Supporting Components

### 5. Rate Limiting (`ratelimit.go`)

**Purpose**: Implements intelligent rate limiting with Reddit API integration.

**Features**:

- Token bucket algorithm with burst support
- Dynamic adjustment based on Reddit X-Ratelimit headers
- Context-aware waiting with cancellation support
- Performance metrics tracking
- Configurable burst and refill rates

### 6. Circuit Breaker (`circuit_breaker.go`)

**Purpose**: Implements circuit breaker pattern for improved resilience.

**Features**:

- Three states: Closed, Open, Half-Open
- Configurable failure thresholds and timeouts
- Automatic recovery attempts
- Request success/failure tracking
- Thread-safe operation

### 7. Pagination (`pagination.go`)

**Purpose**: Generic pagination utilities for Reddit API responses.

**Features**:

- Generic pagination interface `Paginator[T]`
- Automatic multi-page fetching
- Configurable page size and limits
- Context-aware cancellation
- Memory-efficient streaming

### 8. Error Handling (`errors.go`)

**Purpose**: Comprehensive error types and helper functions.

**Error Types**:

- `ErrMissingCredentials` - Invalid or missing API credentials
- `ErrRateLimited` - Rate limit exceeded
- `ErrNotFound` - Resource not found
- `ErrServerError` - Reddit server error
- `ErrBadRequest` - Invalid request parameters
- `CircuitBreakerError` - Circuit breaker errors (struct type)

**Helper Functions**:

- `IsRateLimitError(err)` - Check if error is rate limiting
- `IsNotFoundError(err)` - Check if error is not found
- `IsServerError(err)` - Check if error is server-related
- `IsRetryableError(err)` - Check if error should trigger retry

### 9. URL Utilities (`utils.go`)

**Purpose**: URL building and validation utilities.

**Features**:

- Safe URL construction and validation
- Query parameter handling
- Reddit API endpoint builders
- Path sanitization and encoding

### 10. Configuration Options

#### Client Options (`client_options.go`)

```go
// Configuration functions
WithUserAgent(userAgent string)
WithRateLimit(requestsPerMinute, burstSize int)
WithTimeout(timeout time.Duration)
WithHTTPClient(client *http.Client)
WithCircuitBreaker(config *CircuitBreakerConfig)
WithRetryConfig(maxRetries int, baseDelay time.Duration)
WithRequestInterceptor(interceptor func(*http.Request))
WithResponseInterceptor(interceptor func(*http.Response))
```

#### Subreddit Options (`subreddit_options.go`)

```go
// Subreddit-specific options
WithSort(sort string)           // "hot", "new", "top", "rising"
WithSubredditLimit(limit int)   // Number of posts to fetch
WithTimeFilter(filter string)   // "hour", "day", "week", "month", "year", "all"
```

#### Comment Options (`comment_options.go`)

```go
// Comment-specific options
WithCommentLimit(limit int)
WithCommentSort(sort string)
WithCommentDepth(depth int)
WithCommentContext(context int)
WithCommentAfter(after *Comment)
```

## Interface Design

### Dependency Injection Interfaces

#### `commentGetter` Interface

```go
type commentGetter interface {
    getComments(ctx context.Context, subreddit, postID string, opts ...CommentOption) ([]any, error)
}
```

- Allows Posts to fetch comments without direct client dependency
- Enables easy mocking for testing

#### `PostGetter` Interface

```go
type PostGetter interface {
    GetPosts(subreddit string, params map[string]string) ([]Post, string, error)
}
```

- Defines contract for post fetching operations
- Supports alternative implementations and testing

## Data Flow

### Authentication Flow

1. Create `Auth` instance with credentials
2. Call `Authenticate()` to get initial token
3. `EnsureValidToken()` called before each request
4. Automatic token refresh when expired

### Request Flow

1. Client receives method call with context and options
2. Rate limiter checks if request is allowed
3. Authentication layer ensures valid token
4. HTTP request made with proper headers
5. Response parsed and returned as typed data
6. Rate limiter updated based on response headers

### Pagination Flow

1. Initial request with base parameters
2. Parse response for `after` cursor
3. If more data available, make subsequent requests
4. Combine results until limit reached or no more data

## Testing Architecture

### Mock Generation

- Interfaces automatically generate mocks using `mockgen`
- Generated mocks in `mocks/` directory
- Supports comprehensive unit testing

### Test Structure

- Uses Ginkgo BDD framework
- Test files follow `*_test.go` naming
- Helper utilities in `http_test_helper.go`
- Comprehensive coverage of all components
