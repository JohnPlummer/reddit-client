package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
)

func main() {
	// Get credentials from environment variables
	clientID := os.Getenv("REDDIT_CLIENT_ID")
	clientSecret := os.Getenv("REDDIT_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Fatal("REDDIT_CLIENT_ID and REDDIT_CLIENT_SECRET environment variables must be set")
	}

	// Create auth instance
	auth, err := reddit.NewAuth(clientID, clientSecret)
	if err != nil {
		log.Fatalf("Failed to create auth: %v", err)
	}

	// Demonstrate various interceptors
	demonstrateInterceptors(auth)
}

func demonstrateInterceptors(auth *reddit.Auth) {
	fmt.Println("=== Reddit Client Interceptors Demo ===\n")

	// 1. Basic Request and Response Logging
	fmt.Println("1. Basic Logging Interceptors:")
	client1, err := reddit.NewClient(auth,
		reddit.WithRequestInterceptor(reddit.LoggingRequestInterceptor()),
		reddit.WithResponseInterceptor(reddit.LoggingResponseInterceptor()),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	subreddit1 := reddit.NewSubreddit("golang", client1)
	posts, err := subreddit1.GetPosts(context.Background(), reddit.WithSubredditLimit(1))
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
	} else {
		fmt.Printf("Fetched %d posts\n", len(posts))
	}

	fmt.Println()

	// 2. Custom Header Injection
	fmt.Println("2. Header Injection Interceptor:")
	headers := map[string]string{
		"X-Client-Version": "1.0.0",
		"X-Request-Source": "interceptor-demo",
		"X-Custom-Header":  "demo-value",
	}
	
	client2, err := reddit.NewClient(auth,
		reddit.WithRequestInterceptor(reddit.HeaderInjectionRequestInterceptor(headers)),
		reddit.WithRequestInterceptor(func(req *http.Request) error {
			fmt.Printf("Request headers now include: X-Client-Version=%s\n", req.Header.Get("X-Client-Version"))
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	subreddit2 := reddit.NewSubreddit("programming", client2)
	posts, err = subreddit2.GetPosts(context.Background(), reddit.WithSubredditLimit(1))
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
	} else {
		fmt.Printf("Fetched %d posts with custom headers\n", len(posts))
	}

	fmt.Println()

	// 3. Request ID Generation and Tracing
	fmt.Println("3. Request ID Tracing:")
	client3, err := reddit.NewClient(auth,
		reddit.WithRequestInterceptor(reddit.RequestIDRequestInterceptor("X-Request-ID")),
		reddit.WithRequestInterceptor(func(req *http.Request) error {
			requestID := req.Header.Get("X-Request-ID")
			fmt.Printf("Generated Request ID: %s for %s\n", requestID, req.URL.Path)
			return nil
		}),
		reddit.WithResponseInterceptor(func(resp *http.Response) error {
			if resp.Request != nil {
				requestID := resp.Request.Header.Get("X-Request-ID")
				fmt.Printf("Response received for Request ID: %s (Status: %d)\n", requestID, resp.StatusCode)
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	subreddit3 := reddit.NewSubreddit("webdev", client3)
	posts, err = subreddit3.GetPosts(context.Background(), reddit.WithSubredditLimit(1))
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
	} else {
		fmt.Printf("Fetched %d posts with request tracing\n", len(posts))
	}

	fmt.Println()

	// 4. Performance Monitoring
	fmt.Println("4. Performance Monitoring:")
	client4, err := reddit.NewClient(auth,
		reddit.WithRequestInterceptor(func(req *http.Request) error {
			startTime := time.Now()
			// Store start time in context (in real code, you'd use proper context)
			req.Header.Set("X-Start-Time", startTime.Format(time.RFC3339Nano))
			fmt.Printf("Request started at: %s for %s\n", startTime.Format("15:04:05.000"), req.URL.Path)
			return nil
		}),
		reddit.WithResponseInterceptor(func(resp *http.Response) error {
			if resp.Request != nil {
				startTimeStr := resp.Request.Header.Get("X-Start-Time")
				if startTime, err := time.Parse(time.RFC3339Nano, startTimeStr); err == nil {
					duration := time.Since(startTime)
					fmt.Printf("Request completed in: %v (Status: %d)\n", duration, resp.StatusCode)
				}
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	subreddit4 := reddit.NewSubreddit("technology", client4)
	posts, err = subreddit4.GetPosts(context.Background(), reddit.WithSubredditLimit(1))
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
	} else {
		fmt.Printf("Fetched %d posts with performance monitoring\n", len(posts))
	}

	fmt.Println()

	// 5. Deprecation Warning Detection
	fmt.Println("5. Deprecation Warning Detection:")
	client5, err := reddit.NewClient(auth,
		reddit.WithResponseInterceptor(reddit.DeprecationWarningResponseInterceptor()),
		reddit.WithResponseInterceptor(func(resp *http.Response) error {
			// Simulate a deprecation header for demo purposes
			resp.Header.Set("X-API-Deprecated", "This endpoint will be deprecated in v3.0")
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	subreddit5 := reddit.NewSubreddit("coding", client5)
	posts, err = subreddit5.GetPosts(context.Background(), reddit.WithSubredditLimit(1))
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
	} else {
		fmt.Printf("Fetched %d posts with deprecation detection\n", len(posts))
	}

	fmt.Println()

	// 6. Error Handling and Validation
	fmt.Println("6. Error Handling in Interceptors:")
	client6, err := reddit.NewClient(auth,
		reddit.WithRequestInterceptor(func(req *http.Request) error {
			// Validate that required headers are present
			if req.Header.Get("User-Agent") == "" {
				return fmt.Errorf("User-Agent header is required")
			}
			fmt.Printf("Request validation passed for %s\n", req.URL.Path)
			return nil
		}),
		reddit.WithResponseInterceptor(func(resp *http.Response) error {
			// Check for rate limiting
			if resp.StatusCode == 429 {
				fmt.Printf("Rate limit detected! Headers: %v\n", resp.Header)
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	subreddit6 := reddit.NewSubreddit("golang", client6)
	posts, err = subreddit6.GetPosts(context.Background(), reddit.WithSubredditLimit(1))
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
	} else {
		fmt.Printf("Fetched %d posts with error handling\n", len(posts))
	}

	fmt.Println()

	// 7. Chaining Multiple Interceptors
	fmt.Println("7. Multiple Interceptors in Action:")
	var requestCount int
	
	client7, err := reddit.NewClient(auth,
		// Request interceptors (called in order)
		reddit.WithRequestInterceptor(func(req *http.Request) error {
			requestCount++
			fmt.Printf("  → Interceptor 1: Request #%d to %s\n", requestCount, req.URL.Path)
			return nil
		}),
		reddit.WithRequestInterceptor(reddit.HeaderInjectionRequestInterceptor(map[string]string{
			"X-Interceptor": "demo",
		})),
		reddit.WithRequestInterceptor(func(req *http.Request) error {
			fmt.Printf("  → Interceptor 3: Added header X-Interceptor=%s\n", req.Header.Get("X-Interceptor"))
			return nil
		}),
		
		// Response interceptors (called in order)
		reddit.WithResponseInterceptor(func(resp *http.Response) error {
			fmt.Printf("  ← Response Interceptor 1: Status %d\n", resp.StatusCode)
			return nil
		}),
		reddit.WithResponseInterceptor(func(resp *http.Response) error {
			fmt.Printf("  ← Response Interceptor 2: Content-Length %d\n", resp.ContentLength)
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	subreddit7 := reddit.NewSubreddit("programming", client7)
	posts, err = subreddit7.GetPosts(context.Background(), reddit.WithSubredditLimit(1))
	if err != nil {
		log.Printf("Error fetching posts: %v", err)
	} else {
		fmt.Printf("Fetched %d posts through multiple interceptors\n", len(posts))
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("This demo showed how to use request/response interceptors for:")
	fmt.Println("- Logging and debugging")
	fmt.Println("- Header injection and modification") 
	fmt.Println("- Request tracing and correlation")
	fmt.Println("- Performance monitoring")
	fmt.Println("- Deprecation detection")
	fmt.Println("- Error handling and validation")
	fmt.Println("- Chaining multiple interceptors")
}

// Set up structured logging (optional)
func init() {
	// Configure slog for better output
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}