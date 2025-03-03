package reddit

import (
	"fmt"
	"strconv"
)

// SubredditOption is a function type for modifying subreddit request parameters
type SubredditOption func(params map[string]string)

// WithSort returns a SubredditOption that sets the sort order
func WithSort(sort string) SubredditOption {
	return func(params map[string]string) {
		if sort != "" {
			params["sort"] = sort
		}
	}
}

// WithLimit returns a SubredditOption that sets the limit parameter
func WithSubredditLimit(limit int) SubredditOption {
	return func(params map[string]string) {
		if limit > 0 {
			params["limit"] = strconv.Itoa(limit)
		}
	}
}

// WithAfterTimestamp returns a SubredditOption that filters posts created after the given timestamp
func WithAfterTimestamp(timestamp int64) SubredditOption {
	return func(params map[string]string) {
		if timestamp > 0 {
			params["after"] = fmt.Sprintf("t3_%d", timestamp)
		}
	}
}
