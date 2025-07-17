# Deployment Guide

## Overview

This guide covers deploying applications that use the Reddit client library in various environments. Since this is a library, deployment considerations focus on applications that integrate this package.

## Environment Configuration

### Required Environment Variables

```bash
# Reddit API Credentials (Required)
REDDIT_CLIENT_ID=your_client_id
REDDIT_CLIENT_SECRET=your_client_secret

# Optional Configuration
LOG_LEVEL=info                    # debug, info, warn, error
RATE_LIMIT_RPM=60                # Requests per minute
RATE_LIMIT_BURST=5               # Burst size
REQUEST_TIMEOUT=30s              # Request timeout
USER_AGENT=YourApp/1.0          # Custom user agent
```

### Configuration Loading

```go
package main

import (
    "os"
    "strconv"
    "time"
    
    "github.com/JohnPlummer/reddit-client/reddit"
)

func loadConfig() (*reddit.Auth, *reddit.Client, error) {
    // Load credentials
    clientID := os.Getenv("REDDIT_CLIENT_ID")
    clientSecret := os.Getenv("REDDIT_CLIENT_SECRET")
    
    if clientID == "" || clientSecret == "" {
        return nil, nil, fmt.Errorf("missing Reddit credentials")
    }
    
    // Create auth with environment-based configuration
    authOpts := []reddit.AuthOption{
        reddit.WithUserAgent(getEnvOr("USER_AGENT", "MyApp/1.0")),
    }
    
    if timeout := os.Getenv("REQUEST_TIMEOUT"); timeout != "" {
        if d, err := time.ParseDuration(timeout); err == nil {
            authOpts = append(authOpts, reddit.WithAuthTimeout(d))
        }
    }
    
    auth, err := reddit.NewAuth(clientID, clientSecret, authOpts...)
    if err != nil {
        return nil, nil, fmt.Errorf("creating auth: %w", err)
    }
    
    // Create client with environment-based configuration
    clientOpts := []reddit.ClientOption{
        reddit.WithUserAgent(getEnvOr("USER_AGENT", "MyApp/1.0")),
    }
    
    if rpm := os.Getenv("RATE_LIMIT_RPM"); rpm != "" {
        if r, err := strconv.Atoi(rpm); err == nil {
            burst := 5
            if b := os.Getenv("RATE_LIMIT_BURST"); b != "" {
                if br, err := strconv.Atoi(b); err == nil {
                    burst = br
                }
            }
            clientOpts = append(clientOpts, reddit.WithRateLimit(r, burst))
        }
    }
    
    if timeout := os.Getenv("REQUEST_TIMEOUT"); timeout != "" {
        if d, err := time.ParseDuration(timeout); err == nil {
            clientOpts = append(clientOpts, reddit.WithTimeout(d))
        }
    }
    
    client, err := reddit.NewClient(auth, clientOpts...)
    if err != nil {
        return nil, nil, fmt.Errorf("creating client: %w", err)
    }
    
    return auth, client, nil
}

func getEnvOr(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

## Docker Deployment

### Dockerfile Example

```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary
COPY --from=builder /app/main .

# Expose port (if your app serves HTTP)
EXPOSE 8080

CMD ["./main"]
```

### Docker Compose Example

```yaml
version: '3.8'

services:
  reddit-app:
    build: .
    environment:
      - REDDIT_CLIENT_ID=${REDDIT_CLIENT_ID}
      - REDDIT_CLIENT_SECRET=${REDDIT_CLIENT_SECRET}
      - LOG_LEVEL=info
      - RATE_LIMIT_RPM=60
      - RATE_LIMIT_BURST=5
      - REQUEST_TIMEOUT=30s
    restart: unless-stopped
    volumes:
      - ./data:/app/data  # For persistent data if needed
    depends_on:
      - redis  # If using Redis for caching
      
  redis:
    image: redis:7-alpine
    restart: unless-stopped
    volumes:
      - redis_data:/data
      
volumes:
  redis_data:
```

## Kubernetes Deployment

### ConfigMap for Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: reddit-app-config
data:
  LOG_LEVEL: "info"
  RATE_LIMIT_RPM: "60"
  RATE_LIMIT_BURST: "5"
  REQUEST_TIMEOUT: "30s"
  USER_AGENT: "MyApp/1.0"
```

### Secret for Credentials

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: reddit-credentials
type: Opaque
stringData:
  REDDIT_CLIENT_ID: "your_client_id"
  REDDIT_CLIENT_SECRET: "your_client_secret"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: reddit-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: reddit-app
  template:
    metadata:
      labels:
        app: reddit-app
    spec:
      containers:
      - name: reddit-app
        image: your-registry/reddit-app:latest
        ports:
        - containerPort: 8080
        env:
        - name: REDDIT_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: reddit-credentials
              key: REDDIT_CLIENT_ID
        - name: REDDIT_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: reddit-credentials
              key: REDDIT_CLIENT_SECRET
        envFrom:
        - configMapRef:
            name: reddit-app-config
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
```

## Cloud Platforms

### AWS Lambda

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/JohnPlummer/reddit-client/reddit"
)

type Request struct {
    Subreddit string `json:"subreddit"`
    Limit     int    `json:"limit"`
}

type Response struct {
    Posts []reddit.Post `json:"posts"`
    Error string        `json:"error,omitempty"`
}

var (
    auth   *reddit.Auth
    client *reddit.Client
)

func init() {
    var err error
    auth, err = reddit.NewAuth(
        os.Getenv("REDDIT_CLIENT_ID"),
        os.Getenv("REDDIT_CLIENT_SECRET"),
        reddit.WithUserAgent("LambdaBot/1.0"),
    )
    if err != nil {
        panic(fmt.Sprintf("failed to create auth: %v", err))
    }
    
    client, err = reddit.NewClient(auth)
    if err != nil {
        panic(fmt.Sprintf("failed to create client: %v", err))
    }
}

func handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    var req Request
    if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
        return events.APIGatewayProxyResponse{
            StatusCode: 400,
            Body:       fmt.Sprintf(`{"error": "invalid request: %v"}`, err),
        }, nil
    }
    
    subreddit := reddit.NewSubreddit(req.Subreddit, client)
    posts, err := subreddit.GetPosts(ctx, reddit.WithSubredditLimit(req.Limit))
    
    resp := Response{Posts: posts}
    if err != nil {
        resp.Error = err.Error()
    }
    
    body, _ := json.Marshal(resp)
    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Body:       string(body),
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
    }, nil
}

func main() {
    lambda.Start(handler)
}
```

### Google Cloud Functions

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    
    "github.com/JohnPlummer/reddit-client/reddit"
)

var (
    auth   *reddit.Auth
    client *reddit.Client
)

func init() {
    var err error
    auth, err = reddit.NewAuth(
        os.Getenv("REDDIT_CLIENT_ID"),
        os.Getenv("REDDIT_CLIENT_SECRET"),
        reddit.WithUserAgent("CloudFunctionBot/1.0"),
    )
    if err != nil {
        panic(fmt.Sprintf("failed to create auth: %v", err))
    }
    
    client, err = reddit.NewClient(auth)
    if err != nil {
        panic(fmt.Sprintf("failed to create client: %v", err))
    }
}

func GetPosts(w http.ResponseWriter, r *http.Request) {
    subredditName := r.URL.Query().Get("subreddit")
    if subredditName == "" {
        http.Error(w, "subreddit parameter required", http.StatusBadRequest)
        return
    }
    
    subreddit := reddit.NewSubreddit(subredditName, client)
    posts, err := subreddit.GetPosts(r.Context(), reddit.WithSubredditLimit(10))
    if err != nil {
        http.Error(w, fmt.Sprintf("error fetching posts: %v", err), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(posts)
}
```

## Production Considerations

### Rate Limiting and Quotas

```go
// Configure rate limiting for production
client, err := reddit.NewClient(auth,
    reddit.WithRateLimit(30, 10), // Conservative rate limiting
    reddit.WithTimeout(60*time.Second), // Longer timeout for production
)
```

### Error Handling and Retry Logic

```go
func withRetry(ctx context.Context, maxRetries int, fn func() error) error {
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        // Exponential backoff for rate limit errors
        if reddit.IsRateLimitError(err) {
            backoff := time.Duration(1<<i) * time.Second
            select {
            case <-time.After(backoff):
                continue
            case <-ctx.Done():
                return ctx.Err()
            }
        }
        
        // Don't retry non-retryable errors
        if reddit.IsNotFoundError(err) {
            return err
        }
        
        // For other errors, short delay before retry
        select {
        case <-time.After(time.Second):
            continue
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    return fmt.Errorf("max retries exceeded")
}
```

### Monitoring and Logging

```go
import (
    "context"
    "log/slog"
    "time"
)

func main() {
    // Configure structured logging
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)
    
    // Log application startup
    slog.Info("starting reddit client application",
        "version", "1.0.0",
        "rate_limit", "60/min",
    )
    
    // Create client with monitoring
    client, err := reddit.NewClient(auth,
        reddit.WithUserAgent("ProdApp/1.0"),
        reddit.WithRateLimit(60, 5),
    )
    if err != nil {
        slog.Error("failed to create client", "error", err)
        os.Exit(1)
    }
    
    // Example: Log successful operations
    subreddit := reddit.NewSubreddit("golang", client)
    posts, err := subreddit.GetPosts(context.Background())
    if err != nil {
        slog.Error("failed to fetch posts", "error", err, "subreddit", "golang")
        return
    }
    
    slog.Info("fetched posts successfully", 
        "count", len(posts),
        "subreddit", "golang",
    )
}
```

### Health Checks

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    // Test Reddit API connectivity
    subreddit := reddit.NewSubreddit("test", client)
    _, err := subreddit.GetPosts(ctx, reddit.WithSubredditLimit(1))
    
    if err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
            "error":  err.Error(),
        })
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
    })
}
```

## Security Considerations

### Credential Management

- Never hardcode credentials in source code
- Use environment variables or secure secret management
- Rotate credentials regularly
- Use least privilege principles

### Network Security

- Use HTTPS for all communications
- Implement proper TLS configuration
- Consider using VPN or private networks for sensitive deployments

### Rate Limiting

- Respect Reddit's rate limits
- Implement circuit breakers for resilience
- Monitor API usage and adjust limits accordingly

### Data Privacy

- Don't log sensitive user data
- Implement proper data retention policies
- Follow Reddit's API terms of service
