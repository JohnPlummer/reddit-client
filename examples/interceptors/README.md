# Request/Response Interceptors Example

This example demonstrates how to use the Reddit client's request and response interceptor functionality for logging, debugging, monitoring, and customization.

## What This Example Shows

1. **Basic Logging**: Using built-in logging interceptors for requests and responses
2. **Header Injection**: Adding custom headers to all requests
3. **Request Tracing**: Generating and tracking request IDs for correlation
4. **Performance Monitoring**: Measuring request duration and timing
5. **Deprecation Detection**: Detecting and warning about deprecated API usage
6. **Error Handling**: Validating requests and handling response errors
7. **Interceptor Chaining**: Using multiple interceptors together

## Prerequisites

- Go 1.21 or later
- Reddit API credentials (client ID and secret)

## Setup

1. **Get Reddit API Credentials**:
   - Go to https://www.reddit.com/prefs/apps
   - Create a new "script" application
   - Note the client ID (under the app name) and secret

2. **Set Environment Variables**:
   ```bash
   export REDDIT_CLIENT_ID="your_client_id_here"
   export REDDIT_CLIENT_SECRET="your_client_secret_here"
   ```

3. **Install Dependencies**:
   ```bash
   go mod tidy
   ```

## Running the Example

```bash
go run main.go
```

## Sample Output

The example will demonstrate various interceptor capabilities:

```
=== Reddit Client Interceptors Demo ===

1. Basic Logging Interceptors:
INFO outgoing HTTP request method=GET url=https://oauth.reddit.com/r/golang.json?limit=1
INFO incoming HTTP response status_code=200 url=https://oauth.reddit.com/r/golang.json?limit=1
Fetched 1 posts

2. Header Injection Interceptor:
Request headers now include: X-Client-Version=1.0.0
Fetched 1 posts with custom headers

3. Request ID Tracing:
Generated Request ID: req_1234567890_GET for /r/webdev.json
Response received for Request ID: req_1234567890_GET (Status: 200)
Fetched 1 posts with request tracing

4. Performance Monitoring:
Request started at: 14:30:15.123 for /r/technology.json
Request completed in: 245ms (Status: 200)
Fetched 1 posts with performance monitoring

5. Deprecation Warning Detection:
WARN API deprecation warning url=https://oauth.reddit.com/r/coding.json deprecation_info="This endpoint will be deprecated in v3.0"
Fetched 1 posts with deprecation detection

6. Error Handling in Interceptors:
Request validation passed for /r/golang.json
Fetched 1 posts with error handling

7. Multiple Interceptors in Action:
  → Interceptor 1: Request #1 to /r/programming.json
  → Interceptor 3: Added header X-Interceptor=demo
  ← Response Interceptor 1: Status 200
  ← Response Interceptor 2: Content-Length 1234
Fetched 1 posts through multiple interceptors
```

## Key Concepts Demonstrated

### Built-in Interceptors

The Reddit client provides several pre-built interceptors:

- `LoggingRequestInterceptor()`: Logs outgoing HTTP requests
- `LoggingResponseInterceptor()`: Logs incoming HTTP responses  
- `HeaderInjectionRequestInterceptor(headers)`: Adds custom headers to requests
- `DeprecationWarningResponseInterceptor()`: Warns about deprecated API usage
- `RequestIDRequestInterceptor(headerName)`: Generates unique request IDs

### Custom Interceptors

You can create custom interceptors for any purpose:

```go
// Request interceptor function signature
func(req *http.Request) error

// Response interceptor function signature  
func(resp *http.Response) error
```

### Interceptor Ordering

- Request interceptors are called in the order they are added (before the request is sent)
- Response interceptors are called in the order they are added (after the response is received)
- If any interceptor returns an error, the request fails and subsequent interceptors are not called

### Common Use Cases

1. **Debugging**: Log requests and responses for troubleshooting
2. **Monitoring**: Track performance metrics and request patterns
3. **Authentication**: Add authentication headers or tokens
4. **Correlation**: Generate and track request IDs across services
5. **Validation**: Ensure requests meet certain criteria
6. **Rate Limiting**: Monitor and respond to rate limit headers
7. **Deprecation**: Detect and warn about deprecated API usage
8. **Custom Headers**: Add application-specific metadata

## Best Practices

1. **Keep interceptors lightweight**: Avoid heavy processing that could slow down requests
2. **Handle errors gracefully**: Return meaningful error messages from interceptors
3. **Use structured logging**: Include relevant context in log messages
4. **Chain interceptors logically**: Order interceptors by their purpose and dependencies
5. **Test interceptor behavior**: Ensure interceptors work correctly with retry logic and error handling
6. **Avoid side effects**: Don't modify shared state from interceptors without proper synchronization

## Integration with Other Features

Interceptors work seamlessly with all other Reddit client features:

- **Retries**: Interceptors are called on each retry attempt
- **Rate Limiting**: Interceptors can monitor rate limit headers
- **Circuit Breaker**: Interceptors are called even when circuit breaker is active
- **Authentication**: Interceptors can add additional auth headers
- **Pagination**: Interceptors are called for each page request

This makes interceptors a powerful tool for cross-cutting concerns in your Reddit API integration.