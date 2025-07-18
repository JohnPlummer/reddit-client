package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	"github.com/joho/godotenv"
)

// This example demonstrates how to configure connection pooling for optimal performance
// when making many concurrent requests to the Reddit API.

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		slog.Error("error loading .env file", "error", err)
		os.Exit(1)
	}

	// Setup logging
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Initialize authentication
	auth, err := reddit.NewAuth(
		os.Getenv("REDDIT_CLIENT_ID"),
		os.Getenv("REDDIT_CLIENT_SECRET"),
		reddit.WithAuthUserAgent("PerformanceTuningExample/1.0"),
	)
	if err != nil {
		slog.Error("failed to create auth client", "error", err)
		os.Exit(1)
	}

	// Create four different client configurations to demonstrate performance differences

	// 1. Default configuration (baseline with compression)
	slog.Info("=== Testing Default Configuration ===")
	defaultClient, err := reddit.NewClient(auth,
		reddit.WithUserAgent("DefaultConfigExample/1.0"),
	)
	if err != nil {
		slog.Error("failed to create default client", "error", err)
		os.Exit(1)
	}

	// 2. Configuration without compression (to demonstrate compression benefits)
	slog.Info("=== Testing Configuration Without Compression ===")
	noCompressionClient, err := reddit.NewClient(auth,
		reddit.WithUserAgent("NoCompressionExample/1.0"),
		reddit.WithNoCompression(), // Disable compression to show performance difference
	)
	if err != nil {
		slog.Error("failed to create no-compression client", "error", err)
		os.Exit(1)
	}

	// 3. Low-throughput configuration (conservative settings)
	slog.Info("=== Testing Low-Throughput Configuration ===")
	lowThroughputConfig := &reddit.TransportConfig{
		MaxIdleConns:        10,               // Low total idle connections
		MaxIdleConnsPerHost: 2,                // Very conservative per-host connections
		IdleConnTimeout:     30 * time.Second, // Shorter timeout
		DisableKeepAlives:   false,
		MaxConnsPerHost:     5, // Limit total connections per host
	}

	lowThroughputClient, err := reddit.NewClient(auth,
		reddit.WithUserAgent("LowThroughputExample/1.0"),
		reddit.WithTransportConfig(lowThroughputConfig),
		reddit.WithRateLimit(30, 3), // Lower rate limit
	)
	if err != nil {
		slog.Error("failed to create low-throughput client", "error", err)
		os.Exit(1)
	}

	// 4. High-throughput configuration (optimized for performance)
	slog.Info("=== Testing High-Throughput Configuration ===")
	highThroughputConfig := &reddit.TransportConfig{
		MaxIdleConns:        200,               // High total idle connections
		MaxIdleConnsPerHost: 20,                // More connections per host
		IdleConnTimeout:     120 * time.Second, // Longer timeout to reuse connections
		DisableKeepAlives:   false,             // Always use keep-alives
		MaxConnsPerHost:     0,                 // No limit on total connections
	}

	highThroughputClient, err := reddit.NewClient(auth,
		reddit.WithUserAgent("HighThroughputExample/1.0"),
		reddit.WithTransportConfig(highThroughputConfig),
		reddit.WithRateLimit(100, 10),      // Higher rate limit
		reddit.WithTimeout(30*time.Second), // Longer timeout for reliability
	)
	if err != nil {
		slog.Error("failed to create high-throughput client", "error", err)
		os.Exit(1)
	}

	// Test each configuration
	subreddits := []string{"golang", "programming", "technology", "webdev", "coding"}

	fmt.Println("\n🔧 Performance Tuning Demonstration")
	fmt.Println("=====================================")

	testConfig("Default (with compression)", defaultClient, subreddits)
	testConfig("No Compression", noCompressionClient, subreddits)
	testConfig("Low-Throughput", lowThroughputClient, subreddits)
	testConfig("High-Throughput", highThroughputClient, subreddits)

	fmt.Println("\n📊 Performance Recommendations")
	fmt.Println("==============================")
	fmt.Println("• Default Configuration: Good for most applications")
	fmt.Println("  - MaxIdleConns: 100, MaxIdleConnsPerHost: 10")
	fmt.Println("  - Compression: Enabled (saves ~30-60% bandwidth)")
	fmt.Println("  - Balances performance and resource usage")
	fmt.Println()
	fmt.Println("• Compression Benefits:")
	fmt.Println("  - Reduces response size by 30-60% for JSON data")
	fmt.Println("  - Faster download times, especially on slower connections")
	fmt.Println("  - Lower bandwidth usage and data costs")
	fmt.Println("  - Enabled by default - only disable for debugging")
	fmt.Println()
	fmt.Println("• Low-Throughput Configuration: For resource-constrained environments")
	fmt.Println("  - MaxIdleConns: 10, MaxIdleConnsPerHost: 2")
	fmt.Println("  - Minimizes memory and connection usage")
	fmt.Println("  - Best for: Mobile apps, serverless functions, low-traffic apps")
	fmt.Println()
	fmt.Println("• High-Throughput Configuration: For high-volume applications")
	fmt.Println("  - MaxIdleConns: 200, MaxIdleConnsPerHost: 20")
	fmt.Println("  - Maximizes connection reuse and performance")
	fmt.Println("  - Best for: Data analysis, bulk operations, high-traffic servers")
	fmt.Println()
	fmt.Println("🔍 Optimization Parameters Explained:")
	fmt.Println("• Compression: Gzip compression for response bodies (enable unless debugging)")
	fmt.Println("• MaxIdleConns: Total idle connections across all hosts")
	fmt.Println("• MaxIdleConnsPerHost: Idle connections per Reddit endpoint")
	fmt.Println("• IdleConnTimeout: How long to keep idle connections")
	fmt.Println("• DisableKeepAlives: Whether to reuse connections (keep false)")
	fmt.Println("• MaxConnsPerHost: Total connections per host (0 = no limit)")
	fmt.Println()
	fmt.Println("💡 Performance Tips:")
	fmt.Println("• Leave compression enabled unless debugging HTTP traffic")
	fmt.Println("• Use higher connection limits for concurrent request patterns")
	fmt.Println("• Monitor response times to find optimal rate limiting settings")
	fmt.Println("• Consider circuit breakers for resilient high-throughput applications")
}

func testConfig(configName string, client *reddit.Client, subreddits []string) {
	fmt.Printf("\n🧪 Testing %s Configuration\n", configName)
	fmt.Println(strings.Repeat("-", 40))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]int)
	errors := make(map[string]error)

	// Fetch posts from multiple subreddits concurrently
	for _, subreddit := range subreddits {
		wg.Add(1)
		go func(sub string) {
			defer wg.Done()

			subredditClient := reddit.NewSubreddit(sub, client)
			posts, err := subredditClient.GetPosts(ctx,
				reddit.WithSort("hot"),
				reddit.WithSubredditLimit(5), // Small limit for demonstration
			)

			mu.Lock()
			if err != nil {
				errors[sub] = err
				slog.Error("failed to fetch posts", "subreddit", sub, "error", err)
			} else {
				results[sub] = len(posts)
				fmt.Printf("  ✓ r/%s: %d posts\n", sub, len(posts))
			}
			mu.Unlock()
		}(subreddit)
	}

	wg.Wait()
	duration := time.Since(start)

	// Summary
	totalPosts := 0
	successCount := 0
	for _, count := range results {
		totalPosts += count
		successCount++
	}

	fmt.Printf("\n📈 %s Results:\n", configName)
	fmt.Printf("  • Duration: %v\n", duration)
	fmt.Printf("  • Successful requests: %d/%d\n", successCount, len(subreddits))
	fmt.Printf("  • Total posts fetched: %d\n", totalPosts)
	fmt.Printf("  • Average time per request: %v\n", duration/time.Duration(len(subreddits)))

	if len(errors) > 0 {
		fmt.Printf("  • Errors: %d\n", len(errors))
		for sub, err := range errors {
			fmt.Printf("    - r/%s: %v\n", sub, err)
		}
	}
}
