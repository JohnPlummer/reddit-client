package reddit

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockCommentGetter implements commentGetter for testing
type mockCommentGetter struct {
	comments []interface{}
	err      error
}

func (m *mockCommentGetter) getComments(ctx context.Context, subreddit, postID string, params map[string]string) ([]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.comments, nil
}

var _ = Describe("Post", func() {
	Describe("GetComments", func() {
		It("fetches comments for a post", func() {
			mockComments := []interface{}{
				map[string]interface{}{}, // Post data
				map[string]interface{}{ // Comments data
					"data": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
								"data": map[string]interface{}{
									"author":      "user1",
									"body":        "comment1",
									"created_utc": float64(1234567890),
									"id":          "c1",
								},
							},
						},
					},
				},
			}

			client := &mockCommentGetter{comments: mockComments}
			post := Post{ID: "123", Subreddit: "golang", client: client}

			comments, err := post.GetComments(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(comments).To(HaveLen(1))
			Expect(comments[0].Author).To(Equal("user1"))
			Expect(comments[0].Body).To(Equal("comment1"))
		})

		It("handles comment filtering by timestamp", func() {
			mockComments := []interface{}{
				map[string]interface{}{},
				map[string]interface{}{
					"data": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
								"data": map[string]interface{}{
									"author":      "user1",
									"body":        "comment1",
									"created_utc": float64(1000),
									"id":          "c1",
								},
							},
						},
					},
				},
			}

			client := &mockCommentGetter{comments: mockComments}
			post := Post{ID: "123", Subreddit: "golang", client: client}

			comments, err := post.GetComments(context.Background(), CommentsSince(500))

			Expect(err).NotTo(HaveOccurred())
			Expect(comments).To(HaveLen(1))
		})

		It("handles errors", func() {
			client := &mockCommentGetter{err: fmt.Errorf("API error")}
			post := Post{ID: "123", Subreddit: "golang", client: client}

			_, err := post.GetComments(context.Background())

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API error"))
		})
	})
})
