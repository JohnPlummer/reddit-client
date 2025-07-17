package reddit

import (
	"context"
	"fmt"
)

// PaginationResult holds the results of a paginated fetch operation
type PaginationResult[T any] struct {
	Items []T
	After string
}

// FetchPageFunc defines the signature for a function that fetches a single page of items.
// It should return the items, the "after" token for the next page, and any error.
type FetchPageFunc[T any] func(ctx context.Context, after string) ([]T, string, error)

// AfterTokenExtractor defines the signature for a function that extracts the "after" token
// from an item. This allows the pagination system to know what token to use for the next request.
type AfterTokenExtractor[T any] func(item T) string

// PaginationOptions configures pagination behavior
type PaginationOptions struct {
	// Limit is the maximum number of items to fetch across all pages.
	// Set to 0 for unlimited (use with caution).
	Limit int

	// PageSize is the number of items to request per page.
	// This is passed to the fetch function and may be used as a hint.
	PageSize int

	// StopOnEmpty determines whether to stop pagination when an empty page is received.
	// Default is true, which prevents infinite loops when the API returns empty pages
	// but still provides an "after" token.
	StopOnEmpty bool
}

// DefaultPaginationOptions returns sensible defaults for pagination
func DefaultPaginationOptions() PaginationOptions {
	return PaginationOptions{
		Limit:       0,   // No limit by default
		PageSize:    100, // Standard Reddit API page size
		StopOnEmpty: true,
	}
}

// PaginateAll fetches all pages of items using the provided fetch function.
// It handles common pagination scenarios including:
// - Respecting limits
// - Stopping on empty pages
// - Error handling and propagation
// - Automatic "after" token management
//
// The fetchPage function should handle the actual API call for a single page.
// The after parameter will be empty for the first request.
//
// Example usage:
//
//	fetchPosts := func(ctx context.Context, after string) ([]Post, string, error) {
//		return client.getPostsPage(ctx, subreddit, after)
//	}
//
//	posts, err := PaginateAll(ctx, fetchPosts, PaginationOptions{Limit: 500})
func PaginateAll[T any](
	ctx context.Context,
	fetchPage FetchPageFunc[T],
	opts PaginationOptions,
) ([]T, error) {
	if fetchPage == nil {
		return nil, fmt.Errorf("pagination.PaginateAll: fetchPage function is required")
	}

	var allItems []T
	after := ""

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Fetch the next page
		pageItems, nextAfter, err := fetchPage(ctx, after)
		if err != nil {
			return nil, fmt.Errorf("pagination.PaginateAll: fetch page failed (after=%q): %w", after, err)
		}

		// Add items to our collection
		allItems = append(allItems, pageItems...)

		// Check if we've reached the desired limit
		if opts.Limit > 0 && len(allItems) >= opts.Limit {
			// Trim to exact limit
			allItems = allItems[:opts.Limit]
			break
		}

		// Stop if there are no more pages
		if nextAfter == "" {
			break
		}

		// Stop if we got an empty page (prevents infinite loops with misbehaving APIs)
		if opts.StopOnEmpty && len(pageItems) == 0 {
			break
		}

		// Update the after token for the next request
		after = nextAfter
	}

	return allItems, nil
}

// PaginateAfter fetches pages starting after a specific item.
// This is a convenience wrapper around PaginateAll that extracts the initial "after" token
// from the provided item using the extractor function.
//
// Example usage:
//
//	fetchComments := func(ctx context.Context, after string) ([]Comment, string, error) {
//		return client.getCommentsPage(ctx, subreddit, postID, after)
//	}
//
//	extractAfter := func(c Comment) string { return c.Fullname() }
//
//	comments, err := PaginateAfter(ctx, fetchComments, extractAfter, lastComment, PaginationOptions{Limit: 200})
func PaginateAfter[T any](
	ctx context.Context,
	fetchPage FetchPageFunc[T],
	extractAfter AfterTokenExtractor[T],
	afterItem *T,
	opts PaginationOptions,
) ([]T, error) {
	if fetchPage == nil {
		return nil, fmt.Errorf("pagination.PaginateAfter: fetchPage function is required")
	}
	if extractAfter == nil {
		return nil, fmt.Errorf("pagination.PaginateAfter: extractAfter function is required")
	}

	// Create a modified fetch function that uses the initial after token
	var initialAfter string
	if afterItem != nil {
		initialAfter = extractAfter(*afterItem)
	}

	// Track whether this is the first call
	firstCall := true

	modifiedFetchPage := func(ctx context.Context, after string) ([]T, string, error) {
		if firstCall {
			firstCall = false
			// Use the extracted after token for the first call
			return fetchPage(ctx, initialAfter)
		}
		// Use the provided after token for subsequent calls
		return fetchPage(ctx, after)
	}

	return PaginateAll(ctx, modifiedFetchPage, opts)
}

// PaginateSingle fetches a single page of items.
// This is useful when you only want one page of results, not all available pages.
//
// Example usage:
//
//	fetchPosts := func(ctx context.Context, after string) ([]Post, string, error) {
//		return client.getPostsPage(ctx, subreddit, after)
//	}
//
//	result, err := PaginateSingle(ctx, fetchPosts, "")
func PaginateSingle[T any](
	ctx context.Context,
	fetchPage FetchPageFunc[T],
	after string,
) (*PaginationResult[T], error) {
	if fetchPage == nil {
		return nil, fmt.Errorf("pagination.PaginateSingle: fetchPage function is required")
	}

	items, nextAfter, err := fetchPage(ctx, after)
	if err != nil {
		return nil, fmt.Errorf("pagination.PaginateSingle: fetch page failed (after=%q): %w", after, err)
	}

	return &PaginationResult[T]{
		Items: items,
		After: nextAfter,
	}, nil
}
