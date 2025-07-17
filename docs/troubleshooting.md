# Troubleshooting

## Common Issues and Solutions

### Authentication Issues

#### Invalid Credentials Error

```text
Error: reddit API error: status=401 message=invalid credentials
```

**Causes:**

- Incorrect Client ID or Client Secret
- Client ID/Secret from wrong Reddit app type
- Expired or revoked credentials

**Solutions:**

1. Verify credentials at <https://www.reddit.com/prefs/apps>
2. Ensure app type is set to "script"
3. Check environment variables are set correctly:

   ```bash
   echo $REDDIT_CLIENT_ID
   echo $REDDIT_CLIENT_SECRET
   ```

4. Regenerate credentials if needed

#### Missing Credentials Error

```text
Error: missing credentials
```

**Solutions:**

1. Set required environment variables:

   ```bash
   export REDDIT_CLIENT_ID=your_client_id
   export REDDIT_CLIENT_SECRET=your_client_secret
   ```

2. Create `.env` file with credentials
3. Verify credentials are loaded in your application

### Rate Limiting Issues

#### Rate Limited Error

```text
Error: reddit API error: status=429 message=rate limited
```

**Causes:**

- Exceeded Reddit's rate limits (60 requests per minute)
- Burst requests without proper rate limiting
- Multiple instances sharing the same credentials

**Solutions:**

1. Implement exponential backoff:

   ```go
   if reddit.IsRateLimitError(err) {
       time.Sleep(time.Minute) // Wait before retry
       // Retry logic here
   }
   ```

2. Configure conservative rate limiting:

   ```go
   client, err := reddit.NewClient(auth,
       reddit.WithRateLimit(30, 5), // 30 requests per minute
   )
   ```

3. Monitor rate limit headers in responses
4. Distribute requests across multiple credentials if needed

### Network and Connectivity Issues

#### Connection Timeout

```text
Error: making request: context deadline exceeded
```

**Causes:**

- Network connectivity issues
- Reddit API is slow or unavailable
- Timeout set too low

**Solutions:**

1. Increase timeout duration:

   ```go
   client, err := reddit.NewClient(auth,
       reddit.WithTimeout(60*time.Second),
   )
   ```

2. Check internet connectivity
3. Verify Reddit API status at <https://www.redditstatus.com/>
4. Implement retry logic with exponential backoff

#### SSL/TLS Issues

```text
Error: x509: certificate signed by unknown authority
```

**Solutions:**

1. Update CA certificates on your system
2. Configure custom HTTP client with proper TLS:

   ```go
   httpClient := &http.Client{
       Transport: &http.Transport{
           TLSClientConfig: &tls.Config{
               InsecureSkipVerify: false, // Never use true in production
           },
       },
   }
   client, err := reddit.NewClient(auth, reddit.WithHTTPClient(httpClient))
   ```

### API Response Issues

#### Empty Response

```text
No posts returned but no error
```

**Causes:**

- Subreddit doesn't exist
- Subreddit is private
- No posts match the criteria
- Pagination has reached the end

**Solutions:**

1. Verify subreddit exists and is public
2. Check with different sort options:

   ```go
   posts, err := subreddit.GetPosts(ctx,
       reddit.WithSort("new"), // Try different sorts
   )
   ```

3. Test with popular subreddits like "golang" or "programming"

#### Invalid JSON Response

```text
Error: decoding response: invalid character '<' looking for beginning of value
```

**Causes:**

- Reddit returned HTML error page instead of JSON
- Network middleware interfering with response
- Reddit API is down

**Solutions:**

1. Check Reddit API status
2. Verify network configuration
3. Enable debug logging to see raw responses:

   ```go
   slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
       Level: slog.LevelDebug,
   })))
   ```

### Memory and Performance Issues

#### High Memory Usage

```text
Memory usage growing continuously
```

**Causes:**

- Fetching too many posts/comments without limits
- Not releasing references to large data structures
- Memory leaks in long-running applications

**Solutions:**

1. Set reasonable limits:

   ```go
   posts, err := subreddit.GetPosts(ctx,
       reddit.WithSubredditLimit(100), // Reasonable limit
   )
   ```

2. Process data in batches:

   ```go
   for {
       posts, err := subreddit.GetPostsAfter(ctx, lastPost, 25)
       if err != nil || len(posts) == 0 {
           break
       }
       
       // Process posts
       processPosts(posts)
       
       // Clear references
       lastPost = &posts[len(posts)-1]
       posts = nil
   }
   ```

3. Use profiling tools to identify memory leaks

#### Slow Performance

```text
Requests taking very long to complete
```

**Causes:**

- Rate limiting delays
- Large response sizes
- Network latency
- Processing overhead

**Solutions:**

1. Optimize rate limiting configuration
2. Use pagination for large datasets
3. Implement concurrent processing where appropriate:

   ```go
   // Process multiple subreddits concurrently
   var wg sync.WaitGroup
   results := make(chan []reddit.Post, len(subreddits))
   
   for _, name := range subreddits {
       wg.Add(1)
       go func(subredditName string) {
           defer wg.Done()
           sub := reddit.NewSubreddit(subredditName, client)
           posts, err := sub.GetPosts(ctx)
           if err == nil {
               results <- posts
           }
       }(name)
   }
   
   go func() {
       wg.Wait()
       close(results)
   }()
   ```

### Testing Issues

#### Mock Generation Failures

```text
Error: mockgen: command not found
```

**Solutions:**

1. Install mockgen:

   ```bash
   go install github.com/golang/mock/mockgen@latest
   ```

2. Or use make target:

   ```bash
   make install-mockgen
   ```

#### Test Failures

```text
Error: Expected <nil> but got <some error>
```

**Solutions:**

1. Check test setup and teardown
2. Verify mock expectations
3. Ensure proper context usage in tests
4. Run tests with verbose output:

   ```bash
   ginkgo -v ./...
   ```

## Debugging Techniques

### Enable Debug Logging

```go
import "log/slog"

func main() {
    // Enable debug logging
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))
    slog.SetDefault(logger)
    
    // Your code here
}
```

### HTTP Request Debugging

```go
import (
    "net/http"
    "net/http/httputil"
)

// Custom HTTP client with request/response logging
func createDebugClient() *http.Client {
    return &http.Client{
        Transport: &debugTransport{
            Transport: http.DefaultTransport,
        },
    }
}

type debugTransport struct {
    Transport http.RoundTripper
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    // Log request
    reqDump, _ := httputil.DumpRequestOut(req, true)
    fmt.Printf("REQUEST:\n%s\n", reqDump)
    
    // Make request
    resp, err := t.Transport.RoundTrip(req)
    if err != nil {
        return nil, err
    }
    
    // Log response
    respDump, _ := httputil.DumpResponse(resp, true)
    fmt.Printf("RESPONSE:\n%s\n", respDump)
    
    return resp, nil
}

// Use debug client
client, err := reddit.NewClient(auth, reddit.WithHTTPClient(createDebugClient()))
```

### Error Context

```go
// Add context to errors for better debugging
func fetchPostsWithContext(ctx context.Context, subredditName string) ([]reddit.Post, error) {
    subreddit := reddit.NewSubreddit(subredditName, client)
    posts, err := subreddit.GetPosts(ctx)
    if err != nil {
        return nil, fmt.Errorf("fetching posts from r/%s: %w", subredditName, err)
    }
    return posts, nil
}
```

### Performance Profiling

```go
import (
    "net/http"
    _ "net/http/pprof"
)

func main() {
    // Enable pprof endpoint
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // Your application code
}
```

Access profiling at: <http://localhost:6060/debug/pprof/>

## Environment-Specific Issues

### Docker Issues

#### Container Can't Access Reddit

```text
Error: dial tcp: lookup oauth.reddit.com: no such host
```

**Solutions:**

1. Check DNS configuration in container
2. Verify network connectivity:

   ```bash
   docker run --rm your-image nslookup oauth.reddit.com
   ```

3. Use `--network=host` for debugging

#### Environment Variables Not Set

```text
Error: missing credentials
```

**Solutions:**

1. Check docker-compose.yml or Dockerfile
2. Verify environment variables are passed correctly:

   ```bash
   docker run --rm your-image env | grep REDDIT
   ```

### Kubernetes Issues

#### Pod Can't Access Reddit API

```text
Error: connection refused
```

**Solutions:**

1. Check network policies
2. Verify DNS resolution in pod:

   ```bash
   kubectl exec -it pod-name -- nslookup oauth.reddit.com
   ```

3. Check egress traffic rules

#### Secret Not Found

```text
Error: couldn't find key REDDIT_CLIENT_ID in Secret
```

**Solutions:**

1. Verify secret exists:

   ```bash
   kubectl get secret reddit-credentials -o yaml
   ```

2. Check secret key names match environment variable names

## Getting Help

### Collecting Debug Information

When reporting issues, include:

1. **Go version**: `go version`
2. **Library version**: Check `go.mod`
3. **Error message**: Full error with stack trace
4. **Configuration**: Sanitized configuration (remove credentials)
5. **Environment**: OS, container, cloud platform
6. **Debug logs**: Enable debug logging and include relevant logs

### Debug Log Example

```go
func main() {
    // Enable debug logging
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))
    slog.SetDefault(logger)
    
    // Your code that's causing issues
    auth, err := reddit.NewAuth(clientID, clientSecret)
    if err != nil {
        slog.Error("auth creation failed", "error", err)
        return
    }
    
    client, err := reddit.NewClient(auth)
    if err != nil {
        slog.Error("client creation failed", "error", err)
        return
    }
    
    // ... rest of your code
}
```

### Minimal Reproduction Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JohnPlummer/reddit-client/reddit"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    auth, err := reddit.NewAuth(
        os.Getenv("REDDIT_CLIENT_ID"),
        os.Getenv("REDDIT_CLIENT_SECRET"),
    )
    if err != nil {
        log.Fatal("Auth creation failed:", err)
    }

    client, err := reddit.NewClient(auth)
    if err != nil {
        log.Fatal("Client creation failed:", err)
    }

    subreddit := reddit.NewSubreddit("golang", client)
    posts, err := subreddit.GetPosts(ctx, reddit.WithSubredditLimit(1))
    if err != nil {
        log.Fatal("Get posts failed:", err)
    }

    fmt.Printf("Successfully fetched %d posts\n", len(posts))
}
```

This minimal example helps isolate issues and provides a starting point for debugging.
