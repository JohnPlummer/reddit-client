package reddit

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockClient implements just the methods needed for Post tests
type mockClient struct {
	comments      []interface{}
	commentsAfter []Comment
	commentsErr   error
}

func (m *mockClient) getComments(ctx context.Context, subreddit, postID string, params map[string]string) ([]interface{}, error) {
	if m.commentsErr != nil {
		return nil, m.commentsErr
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

func (m *mockClient) getCommentsAfter(ctx context.Context, subreddit string, postID string, after *Comment, limit int) ([]Comment, error) {
	if m.commentsErr != nil {
		return nil, m.commentsErr
	}
	return m.commentsAfter, nil
}

var _ = Describe("Post", func() {
	Describe("Fullname", func() {
		It("returns the correct Reddit fullname format", func() {
			post := Post{ID: "abc123"}
			Expect(post.Fullname()).To(Equal("t3_abc123"))
		})

		It("handles empty ID", func() {
			post := Post{}
			Expect(post.Fullname()).To(Equal("t3_"))
		})
	})

	Describe("GetComments", func() {
		var (
			client *mockClient
			post   *Post
			ctx    context.Context
		)

		BeforeEach(func() {
			client = &mockClient{}
			post = &Post{ID: "123", Title: "Test Post", Subreddit: "golang", client: client}
			ctx = context.Background()
		})

		It("fetches comments for a post", func() {
			client.comments = []interface{}{
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

			comments, err := post.GetComments(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(comments).To(HaveLen(2))
			Expect(comments[0].ID).To(Equal("c1"))
			Expect(comments[0].Author).To(Equal("user1"))
			Expect(comments[0].Body).To(Equal("comment1"))
		})

		It("handles errors when fetching comments", func() {
			expectedErr := errors.New("API error")
			client.commentsErr = expectedErr

			comments, err := post.GetComments(ctx)
			Expect(err).To(MatchError("fetching comments: API error"))
			Expect(errors.Is(err, expectedErr)).To(BeTrue())
			Expect(comments).To(BeNil())
		})

		It("fetches comments after a specific comment", func() {
			// First page setup
			client.comments = []interface{}{
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
						},
					},
				},
			}

			// Get first page of comments
			comments, err := post.GetComments(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(comments).To(HaveLen(1))
			Expect(comments[0].ID).To(Equal("c1"))
			Expect(comments[0].Author).To(Equal("user1"))
			Expect(comments[0].Body).To(Equal("comment1"))

			// Get comments after the first comment
			moreComments, err := post.GetCommentsAfter(ctx, &comments[0], 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(moreComments).To(HaveLen(1))
			Expect(moreComments[0].ID).To(Equal("c2"))
			Expect(moreComments[0].Author).To(Equal("user2"))
			Expect(moreComments[0].Body).To(Equal("comment2"))
		})

		It("handles errors when fetching comments after", func() {
			firstComment := Comment{ID: "c1"}
			expectedErr := errors.New("API error")
			client.commentsErr = expectedErr

			moreComments, err := post.GetCommentsAfter(ctx, &firstComment, 1)
			Expect(err).To(MatchError("fetching comments after: API error"))
			Expect(errors.Is(err, expectedErr)).To(BeTrue())
			Expect(moreComments).To(BeNil())
		})
	})

	Describe("Comment", func() {
		It("returns the correct fullname format", func() {
			comment := Comment{ID: "abc123"}
			Expect(comment.Fullname()).To(Equal("t1_abc123"))
		})

		It("handles empty ID", func() {
			comment := Comment{}
			Expect(comment.Fullname()).To(Equal("t1_"))
		})
	})
})
