package reddit_test

import (
	"context"
	"errors"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Post", func() {
	Describe("Fullname", func() {
		It("returns the correct Reddit fullname format", func() {
			post := reddit.Post{ID: "abc123"}
			Expect(post.Fullname()).To(Equal("t3_abc123"))
		})

		It("handles empty ID", func() {
			post := reddit.Post{}
			Expect(post.Fullname()).To(Equal("t3_"))
		})
	})

	Describe("GetComments", func() {
		var (
			post     *reddit.Post
			testMock reddit.TestCommentGetter // Exposing just an interface-typed reference
			ctx      context.Context
		)

		BeforeEach(func() {
			// Use the test helper in the reddit package
			post, testMock = reddit.NewTestPost("123", "Test Post", "golang")
			ctx = context.Background()
		})

		It("fetches comments for a post", func() {
			// Configure the mock directly through interface
			testMock.SetupComments(reddit.SetupTestCommentsData())

			comments, err := post.GetComments(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(comments).To(HaveLen(2))
			Expect(comments[0].ID).To(Equal("c1"))
			Expect(comments[0].Author).To(Equal("user1"))
			Expect(comments[0].Body).To(Equal("comment1"))
		})

		It("handles errors when fetching comments", func() {
			expectedErr := errors.New("API error")
			testMock.SetupError(expectedErr)

			comments, err := post.GetComments(ctx)
			Expect(err).To(MatchError("fetching comments: API error"))
			Expect(errors.Is(err, expectedErr)).To(BeTrue())
			Expect(comments).To(BeNil())
		})

		It("fetches comments after a specific comment", func() {
			// First page setup - single comment
			commentsData := []interface{}{
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
			testMock.SetupComments(commentsData)

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
			firstComment := reddit.Comment{ID: "c1"}
			expectedErr := errors.New("API error")
			testMock.SetupError(expectedErr)

			moreComments, err := post.GetCommentsAfter(ctx, &firstComment, 1)
			Expect(err).To(MatchError("fetching comments after: API error"))
			Expect(errors.Is(err, expectedErr)).To(BeTrue())
			Expect(moreComments).To(BeNil())
		})
	})

	Describe("Comment", func() {
		It("returns the correct fullname format", func() {
			comment := reddit.Comment{ID: "abc123"}
			Expect(comment.Fullname()).To(Equal("t1_abc123"))
		})

		It("handles empty ID", func() {
			comment := reddit.Comment{}
			Expect(comment.Fullname()).To(Equal("t1_"))
		})
	})
})
