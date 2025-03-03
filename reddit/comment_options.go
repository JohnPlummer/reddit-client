package reddit

import (
	"fmt"
	"strconv"
)

// CommentOption is a function type for modifying comment request parameters
type CommentOption func(params map[string]string)

// WithCommentLimit returns a CommentOption that sets the limit parameter
func WithCommentLimit(limit int) CommentOption {
	return func(params map[string]string) {
		if limit > 0 {
			params["limit"] = strconv.Itoa(limit)
		}
	}
}

// WithCommentAfter returns a CommentOption that sets the after parameter
func WithCommentAfter(comment *Comment) CommentOption {
	return func(params map[string]string) {
		if comment != nil {
			params["after"] = comment.Fullname()
		}
	}
}

// WithCommentSort returns a CommentOption that sets the sort parameter
func WithCommentSort(sort string) CommentOption {
	return func(params map[string]string) {
		if sort != "" {
			params["sort"] = sort
		}
	}
}

// WithCommentDepth returns a CommentOption that sets the depth parameter
func WithCommentDepth(depth int) CommentOption {
	return func(params map[string]string) {
		if depth > 0 {
			params["depth"] = strconv.Itoa(depth)
		}
	}
}

// WithCommentContext returns a CommentOption that sets the context parameter
func WithCommentContext(context int) CommentOption {
	return func(params map[string]string) {
		if context > 0 {
			params["context"] = strconv.Itoa(context)
		}
	}
}

// WithCommentShowMore returns a CommentOption that sets the show_more parameter
func WithCommentShowMore(showMore bool) CommentOption {
	return func(params map[string]string) {
		params["show_more"] = fmt.Sprintf("%v", showMore)
	}
}
