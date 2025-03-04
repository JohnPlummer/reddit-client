package reddit

import "context"

// TestCommentGetter is a public interface for testing
//
//go:generate mockgen -destination=mocks/test_comment_getter_mock.go -package=mocks github.com/JohnPlummer/reddit-client/reddit TestCommentGetter
type TestCommentGetter interface {
	// This interface should define the methods that tests need to configure the mock
	SetupComments(comments []interface{})
	SetupCommentsAfter(comments []Comment)
	SetupError(err error)
}

// testCommentGetter is a testing implementation of commentGetter that also
// implements the TestCommentGetter interface for external use
type testCommentGetter struct {
	comments      []interface{}
	commentsAfter []Comment
	commentsErr   error
}

// Ensure testCommentGetter implements both interfaces
var _ commentGetter = (*testCommentGetter)(nil)
var _ TestCommentGetter = (*testCommentGetter)(nil)

// getComments implements the commentGetter interface for testing
func (m *testCommentGetter) getComments(ctx context.Context, subreddit, postID string, opts ...CommentOption) ([]interface{}, error) {
	if m.commentsErr != nil {
		return nil, m.commentsErr
	}

	// Convert options to params for testing
	params := make(map[string]string)
	for _, opt := range opts {
		opt(params)
	}

	// If "after" parameter is present, return second page
	if after, ok := params["after"]; ok && after == "t1_c1" {
		return []interface{}{
			map[string]interface{}{}, // First element (post data)
			map[string]interface{}{ // Second element (comments data)
				"data": map[string]interface{}{
					"children": []interface{}{
						map[string]interface{}{
							"data": map[string]interface{}{
								"id":     "c2",
								"author": "user2",
								"body":   "comment2",
							},
						},
					},
				},
			},
		}, nil
	}

	// Return first page by default
	if m.comments == nil {
		return []interface{}{
			map[string]interface{}{}, // First element (post data)
			map[string]interface{}{ // Second element (comments data)
				"data": map[string]interface{}{
					"children": []interface{}{},
				},
			},
		}, nil
	}
	return m.comments, nil
}

// NewTestPost creates a post with a mock client for testing
func NewTestPost(id, title, subreddit string) (*Post, TestCommentGetter) {
	client := &testCommentGetter{}
	post := &Post{
		ID:        id,
		Title:     title,
		Subreddit: subreddit,
		client:    client,
	}
	return post, client
}

// Implementation of TestCommentGetter interface methods

// SetupComments implements TestCommentGetter.SetupComments
func (m *testCommentGetter) SetupComments(comments []interface{}) {
	m.comments = comments
}

// SetupCommentsAfter implements TestCommentGetter.SetupCommentsAfter
func (m *testCommentGetter) SetupCommentsAfter(comments []Comment) {
	m.commentsAfter = comments
}

// SetupError implements TestCommentGetter.SetupError
func (m *testCommentGetter) SetupError(err error) {
	m.commentsErr = err
}

// SetupTestCommentsData creates a standard test response with two comments
func SetupTestCommentsData() []interface{} {
	return []interface{}{
		map[string]interface{}{}, // First element (post data)
		map[string]interface{}{ // Second element (comments data)
			"data": map[string]interface{}{
				"children": []interface{}{
					map[string]interface{}{
						"data": map[string]interface{}{
							"id":     "c1",
							"author": "user1",
							"body":   "comment1",
						},
					},
					map[string]interface{}{
						"data": map[string]interface{}{
							"id":     "c2",
							"author": "user2",
							"body":   "comment2",
						},
					},
				},
			},
		},
	}
}
