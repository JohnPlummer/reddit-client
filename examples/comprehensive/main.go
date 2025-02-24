package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	"github.com/joho/godotenv"
)

const fetchIntervalDelay = time.Second

// Configuration holds the application configuration
type Config struct {
	subreddit   string
	limit       int
	sort        string
	timeframe   string
	maxPages    int
	logLevel    string
	rateLimit   int
	rateBurst   int
	readTimeout time.Duration
	outputFile  string
}

// Result represents the fetched data that will be saved
type Result struct {
	Subreddit  string        `json:"subreddit"`
	FetchedAt  time.Time     `json:"fetched_at"`
	TotalPosts int           `json:"total_posts"`
	Posts      []reddit.Post `json:"posts"`
	Error      string        `json:"error,omitempty"`
}

func main() {
	// Parse and validate command line flags
	cfg := parseFlags()
	if err := validateConfig(cfg); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Setup logging
	setupLogging(cfg.logLevel)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a WaitGroup to wait for all goroutines
	var wg sync.WaitGroup

	// Start signal handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case sig := <-sigChan:
			slog.Info("received signal, initiating shutdown", "signal", sig)
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	// Add timeout to context
	ctx, cancel = context.WithTimeout(ctx, cfg.readTimeout)
	defer cancel()

	// Load .env file
	if err := godotenv.Load(); err != nil {
		slog.Error("error loading .env file", "error", err)
		os.Exit(1)
	}

	// Initialize authentication with rate limiting
	auth, err := reddit.NewAuth(
		os.Getenv("REDDIT_CLIENT_ID"),
		os.Getenv("REDDIT_CLIENT_SECRET"),
		reddit.WithUserAgent("MyRedditBot/1.0"),
		reddit.WithRateLimit(cfg.rateLimit, cfg.rateBurst),
	)
	if err != nil {
		slog.Error("failed to create auth client", "error", err)
		os.Exit(1)
	}

	// Create a new client
	client, err := reddit.NewClient(auth)
	if err != nil {
		slog.Error("failed to create client", "error", err)
		os.Exit(1)
	}

	// Initialize result
	result := Result{
		Subreddit: cfg.subreddit,
		FetchedAt: time.Now(),
		Posts:     make([]reddit.Post, 0),
	}

	// Get posts with pagination
	var after string
	pageCount := 0
	totalPosts := 0

FetchLoop:
	for pageCount < cfg.maxPages {
		select {
		case <-ctx.Done():
			result.Error = "operation cancelled"
			break FetchLoop
		case <-time.After(fetchIntervalDelay):
		}

		// Show progress
		fmt.Printf("\rFetching page %d/%d...", pageCount+1, cfg.maxPages)

		// Parameters for the request
		params := map[string]string{
			"limit": fmt.Sprintf("%d", cfg.limit),
			"sort":  cfg.sort,
			"t":     cfg.timeframe,
		}
		if after != "" {
			params["after"] = after
		}

		// Get posts for current page
		posts, nextAfter, err := client.GetPosts(ctx, cfg.subreddit, params)
		if err != nil {
			slog.Error("Error getting posts", "error", err, "page", pageCount+1)
			result.Error = fmt.Sprintf("error fetching page %d: %v", pageCount+1, err)
			break
		}

		// Process posts
		for i, post := range posts {
			fmt.Printf("\n=== Post %d ===\n", totalPosts+i+1)
			fmt.Println(post.String())

			// Get comments for this post
			comments, err := post.GetComments(ctx)
			if err != nil {
				slog.Error("Error getting comments for post",
					"post_id", post.ID,
					"error", err,
					"subreddit", cfg.subreddit,
				)
				continue
			}

			fmt.Printf("\nComments (%d):\n", len(comments))
			for _, comment := range comments {
				fmt.Printf("%+v\n", comment)
				fmt.Println("-------------------")
			}
		}

		result.Posts = append(result.Posts, posts...)
		totalPosts += len(posts)
		result.TotalPosts = totalPosts

		// Check if we have more pages
		if nextAfter == "" || len(posts) == 0 {
			break
		}

		after = nextAfter
		pageCount++
	}

	fmt.Println() // Clear progress line
	slog.Info("Completed fetching posts",
		"total_posts", totalPosts,
		"pages_fetched", pageCount+1,
		"subreddit", cfg.subreddit,
	)

	// Cancel context to signal completion to goroutines
	cancel()

	// Save results if output file is specified
	if err := saveResult(cfg, result); err != nil {
		slog.Error("failed to save results", "error", err)
	}

	// Wait for signal handler to complete
	wg.Wait()
}

func parseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.subreddit, "subreddit", "brighton", "Subreddit to fetch posts from")
	flag.IntVar(&cfg.limit, "limit", 10, "Number of posts per page")
	flag.StringVar(&cfg.sort, "sort", "new", "Sort order (new, hot, top, rising)")
	flag.StringVar(&cfg.timeframe, "timeframe", "all", "Timeframe for top posts (hour, day, week, month, year, all)")
	flag.IntVar(&cfg.maxPages, "max-pages", 1, "Maximum number of pages to fetch")
	flag.StringVar(&cfg.logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.IntVar(&cfg.rateLimit, "rate-limit", 1, "Rate limit in requests per second")
	flag.IntVar(&cfg.rateBurst, "rate-burst", 5, "Maximum burst size for rate limiting")
	flag.DurationVar(&cfg.readTimeout, "timeout", 30*time.Second, "Read timeout duration")
	flag.StringVar(&cfg.outputFile, "output", "", "Output file path (JSON format)")

	flag.Parse()
	return cfg
}

func validateConfig(cfg *Config) error {
	if cfg.subreddit == "" {
		return fmt.Errorf("subreddit cannot be empty")
	}

	validSorts := map[string]bool{"new": true, "hot": true, "top": true, "rising": true}
	if !validSorts[cfg.sort] {
		return fmt.Errorf("invalid sort order: %s", cfg.sort)
	}

	validTimeframes := map[string]bool{
		"hour": true, "day": true, "week": true,
		"month": true, "year": true, "all": true,
	}
	if !validTimeframes[cfg.timeframe] {
		return fmt.Errorf("invalid timeframe: %s", cfg.timeframe)
	}

	if cfg.limit < 1 || cfg.limit > 100 {
		return fmt.Errorf("limit must be between 1 and 100")
	}

	if cfg.maxPages < 1 {
		return fmt.Errorf("max-pages must be at least 1")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true,
		"warn": true, "error": true,
	}
	if !validLogLevels[strings.ToLower(cfg.logLevel)] {
		return fmt.Errorf("invalid log level: %s", cfg.logLevel)
	}

	if cfg.rateLimit < 1 {
		return fmt.Errorf("rate-limit must be at least 1")
	}

	if cfg.rateBurst < cfg.rateLimit {
		return fmt.Errorf("rate-burst must be greater than or equal to rate-limit")
	}

	if cfg.readTimeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second")
	}

	return nil
}

func setupLogging(level string) {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func saveResult(cfg *Config, result Result) error {
	if cfg.outputFile == "" {
		return nil
	}

	file, err := os.Create(cfg.outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	slog.Info("results saved to file", "path", cfg.outputFile)
	return nil
}
