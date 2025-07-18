# Reddit Client Improvement Tasks

## Sequencing Guide

Tasks are organized with sequence numbers (e.g., `sequence-01`, `sequence-02`) to indicate which tasks can be executed concurrently:

- **sequence-01**: Tasks that can be started immediately and worked on in parallel
- **sequence-02**: Tasks that can begin after sequence-01 is complete
- **sequence-03**: Tasks that depend on earlier sequences
- **sequence-04**: Final tasks that require most other improvements to be complete

Multiple developers can pick up any tasks within the same sequence number and work on them simultaneously without conflicts.

---

## High Priority Tasks

### 1. Create test file for errors.go with comprehensive error scenarios ✅ COMPLETED

**Sequence**: `sequence-01`  
**File**: `reddit/errors_test.go` (new file)  
**Current Coverage**: 100%  
**Details**:

- Create comprehensive test suite for all functions in `errors.go`
- Test `NewAPIError()` with various HTTP status codes (401, 429, 404, 400, 500+)
- Test `IsRateLimitError()`, `IsNotFoundError()`, `IsServerError()` with both simple errors and APIError types
- Test edge cases: nil errors, wrapped errors using `errors.As()`
- Verify error messages and APIError struct fields are populated correctly

### 2. Add tests for auth_options.go functions with 0% coverage ✅ COMPLETED

**Sequence**: `sequence-01`  
**File**: `reddit/auth_options_test.go`  
**Functions to test**:

- `WithAuthHTTPClient()` (line 19) - test custom HTTP client configuration
- `WithAuthHTTPTimeout()` (line 40) - test timeout duration setting
**Details**:
- Verify options correctly modify Auth configuration
- Test with various timeout values including edge cases (0, negative, very large)
- Ensure custom HTTP clients are properly set

### 3. Add tests for client_options.go functions with 0% coverage ✅ COMPLETED

**Sequence**: `sequence-01`  
**File**: `reddit/client_options_test.go`  
**Functions to test**:

- `WithHTTPClient()` (line 27) - test custom HTTP client configuration
- `WithDebug()` (line 74) - test debug flag setting
**Details**:
- Verify options correctly modify Client configuration
- Test that custom HTTP clients override defaults
- Ensure debug flag is properly propagated

### 4. Add tests for comment_options.go functions with 0% coverage ✅ COMPLETED

**Sequence**: `sequence-01`  
**File**: `reddit/comment_options_test.go`  
**Functions to test**:

- `WithCommentSort()` (line 30)
- `WithCommentAfter()` (line 39)
- `WithCommentBefore()` (line 48)
- `WithCommentCount()` (line 57)
**Details**:
- Test all sort options (new, top, controversial, etc.)
- Verify after/before pagination parameters are set correctly
- Test edge cases for count (0, negative, very large numbers)
- Ensure options properly modify request parameters

### 5. Add tests for RateLimiter Allow(), Reserve(), and UpdateLimit() methods ✅ COMPLETED

**Sequence**: `sequence-01`  
**File**: `reddit/ratelimit_test.go`  
**Methods to test**:

- `Allow()` (line 40) - test non-blocking rate limit check
- `Reserve()` (line 46) - test reservation mechanism
- `UpdateLimit()` (line 51) - test dynamic rate limit updates
**Details**:
- Test `Allow()` returns false when rate limit exceeded
- Test `Reserve()` provides correct wait times
- Test `UpdateLimit()` with various remaining/reset values
- Test edge cases: remaining=0, past reset times, future reset times
- Verify burst and limit adjustments are correct

### 6. Add edge case tests for pagination in GetPostsAfter and GetCommentsAfter ✅ COMPLETED

**Sequence**: `sequence-01`  
**Files**: `reddit/post_test.go`, `reddit/subreddit_test.go`  
**Details**:

- Test pagination with empty pages (recent fix)
- Test pagination limits (exact limit, over limit, under limit)
- Test error handling during pagination (network errors mid-pagination)
- Test pagination with nil after parameter
- Test very large limit values
- Verify proper handling when API returns duplicate items

### 7. Implement consistent error wrapping with context throughout codebase ✅ COMPLETED

**Sequence**: `sequence-02`  
**Files**: All files with error returns  
**Details**:

- Audit all error returns to ensure they use `fmt.Errorf` with `%w` verb
- Add descriptive context to each error (e.g., "auth.GetToken: refresh failed: %w")
- Ensure error messages follow consistent format: "component.method: action failed: %w"
- Special attention to:
  - `auth.go`: Token refresh errors
  - `client.go`: HTTP request errors
  - `post.go`/`comment.go`: Parsing errors
- Maintain backward compatibility for error checking

### 8. Add retry logic with exponential backoff for transient errors (429, 503) ✅ COMPLETED

**Sequence**: `sequence-02`  
**File**: `reddit/client.go` (modify `request` method)  
**Details**:

- Implement retry logic for status codes: 429 (rate limited), 503 (service unavailable), 502 (bad gateway)
- Use exponential backoff: 1s, 2s, 4s, 8s (max 3 retries)
- Make retry configuration optional via ClientOption
- Add jitter to prevent thundering herd
- Log retry attempts with context
- Respect Retry-After header if present
- Add tests for retry scenarios

## Medium Priority Tasks

### 9. Parse and utilize Reddit rate limit headers (X-Ratelimit-*) ✅ COMPLETED

**Sequence**: `sequence-02`  
**File**: `reddit/client.go`  
**Headers to parse**:

- `X-Ratelimit-Remaining`: Requests remaining
- `X-Ratelimit-Reset`: Unix timestamp of reset
- `X-Ratelimit-Used`: Requests used
**Details**:
- Extract headers in `request()` method after successful response
- Call `rateLimiter.UpdateLimit()` with parsed values
- Handle missing or malformed headers gracefully
- Add logging for rate limit updates
- Add tests with mocked responses containing these headers

### 10. Add metrics/hooks for monitoring rate limit usage ✅ COMPLETED

**Sequence**: `sequence-03`  
**Files**: `reddit/client.go`, `reddit/client_options.go`  
**Details**:

- Add hooks interface for rate limit events:

  ```go
  type RateLimitHook interface {
      OnRateLimitWait(ctx context.Context, duration time.Duration)
      OnRateLimitUpdate(remaining int, reset time.Time)
      OnRateLimitExceeded(ctx context.Context)
  }
  ```

- Add `WithRateLimitHook()` client option
- Call hooks at appropriate points in request flow
- Provide default no-op implementation
- Add example implementation that logs to slog

### 11. Extract common pagination logic into reusable generic helper ✅ COMPLETED

**Sequence**: `sequence-03`  
**File**: `reddit/pagination.go` (new file)  
**Details**:

- Create generic pagination function that accepts:
  - Fetch function for single page
  - Extract "after" token function
  - Limit parameter
- Refactor `GetPostsAfter` and `GetCommentsAfter` to use this helper
- Handle common cases: empty pages, limit reached, errors
- Ensure type safety with generics (Go 1.18+)
- Add comprehensive tests for the pagination helper

### 12. Create URL building utility function to eliminate query string duplication ✅ COMPLETED

**Sequence**: `sequence-02`  
**File**: `reddit/client.go` or `reddit/utils.go` (new file)  
**Details**:

- Create `buildEndpoint(base string, params map[string]string) string`
- Use `url.Values` for proper encoding
- Replace duplicated query string building in:
  - `getComments()` (lines 76-83)
  - `getPostsPage()` (lines 146-153)
- Handle empty params gracefully
- Add unit tests for various parameter combinations

### 13. Create type-safe field extractors for API response parsing ✅ COMPLETED

**Sequence**: `sequence-02`  
**File**: `reddit/utils.go` (new file)  
**Details**:

- Create helper functions:

  ```go
  func getStringField(data map[string]any, key string) string
  func getFloat64Field(data map[string]any, key string) float64
  func getBoolField(data map[string]any, key string) bool
  ```

- Refactor parsing in `parsePostData()` and `parseCommentData()`
- Add optional default value parameter
- Consider adding validation (e.g., non-negative scores)
- Add comprehensive tests

### 14. Create generic HTTP request wrapper for JSON responses ✅ COMPLETED

**Sequence**: `sequence-03`  
**File**: `reddit/client.go`  
**Details**:

- Create method: `requestJSON(ctx, method, endpoint string, result any) error`
- Consolidate common pattern:
  - Make request
  - Check response
  - Decode JSON
  - Handle errors
- Use in `getComments()`, `getPostsPage()`, and auth methods
- Maintain detailed error context
- Add tests with various response scenarios

## Low Priority Tasks

### 15. Add connection pooling configuration to HTTP client ✅ COMPLETED

**Sequence**: `sequence-04`  
**File**: `reddit/client_options.go`  
**Details**:

- Add options for:
  - MaxIdleConns
  - MaxIdleConnsPerHost
  - IdleConnTimeout
- Create `WithTransportConfig()` option
- Document recommended values for Reddit API
- Add example showing performance tuning

### 16. Implement circuit breaker pattern for API resilience ✅ COMPLETED

**Sequence**: `sequence-04`  
**File**: `reddit/circuit_breaker.go` (new file)  
**Details**:

- Implement simple circuit breaker with states: closed, open, half-open
- Configure thresholds: failure count, timeout duration
- Integrate into `client.request()` method
- Add `WithCircuitBreaker()` client option
- Fast-fail when circuit is open
- Add metrics/logging for circuit state changes

### 17. Add request/response interceptors for logging and debugging ✅ COMPLETED

**Sequence**: `sequence-04`  
**Files**: `reddit/client.go`, `reddit/client_options.go`  
**Details**:

- Add interceptor interfaces:

  ```go
  type RequestInterceptor func(req *http.Request) error
  type ResponseInterceptor func(resp *http.Response) error
  ```

- Add `WithRequestInterceptor()` and `WithResponseInterceptor()` options
- Call interceptors in `request()` method
- Provide example interceptors: logging, header injection
- Ensure interceptors don't break existing functionality

### 18. Add compression support for HTTP requests ✅ COMPLETED

**Sequence**: `sequence-04`  
**File**: `reddit/client.go`  
**Details**:

- Add Accept-Encoding: gzip header to requests
- Handle compressed responses automatically
- Add option to disable compression if needed
- Test with Reddit API to ensure compatibility
- Measure performance improvement

### 19. Run coverage report and verify improvements ✅ COMPLETED

**Sequence**: `sequence-04`  
**Details**:

- Run `make coverage` after all test improvements
- Verify coverage increased from 61.3% to 80%+
- Identify any remaining gaps
- Update coverage.md with new results
- Create summary of coverage improvements
- Document any areas intentionally left with lower coverage

---

## Success Metrics

- Test coverage increased from 61.3% to 91.7% ✅ (exceeded 80%+ goal)
- All functions have at least some test coverage (no 0% files) ✅
- Error handling is consistent and provides good context ✅
- Code duplication is significantly reduced ✅  
- Rate limiting is more intelligent and observable ✅
- The codebase is more maintainable and reliable ✅

**Project Status**: 19/19 tasks completed (100% complete) 🎉
