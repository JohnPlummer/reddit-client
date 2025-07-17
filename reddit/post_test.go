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
			commentsData := []any{
				map[string]any{}, // First element (post data)
				map[string]any{ // Second element (comments data)
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
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

		Context("GetCommentsAfter edge cases", func() {
			BeforeEach(func() {
				testMock.Reset()
			})

			It("handles pagination with empty pages", func() {
				// Setup first page with comments
				firstPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c1",
										"author": "user1",
										"body":   "comment1",
									},
								},
							},
						},
					},
				}
				testMock.SetupComments(firstPageData)

				// Setup empty page after c1
				emptyPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{}, // Empty children array
						},
					},
				}
				testMock.SetupPageResponse("t1_c1", emptyPageData)

				comments, err := post.GetCommentsAfter(ctx, nil, 5)
				Expect(err).NotTo(HaveOccurred())
				Expect(comments).To(HaveLen(1))
				Expect(comments[0].ID).To(Equal("c1"))
				Expect(testMock.GetCallCount()).To(Equal(2)) // Should make 2 calls and stop on empty page
			})

			It("handles pagination with nil after parameter", func() {
				commentsData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c1",
										"author": "user1",
										"body":   "comment1",
									},
								},
								map[string]any{
									"data": map[string]any{
										"id":     "c2",
										"author": "user2",
										"body":   "comment2",
									},
								},
							},
						},
					},
				}
				testMock.SetupComments(commentsData)

				comments, err := post.GetCommentsAfter(ctx, nil, 2)
				Expect(err).NotTo(HaveOccurred())
				Expect(comments).To(HaveLen(2))
				Expect(comments[0].ID).To(Equal("c1"))
				Expect(comments[1].ID).To(Equal("c2"))
			})

			It("respects exact limit with pagination", func() {
				// First page with 2 comments
				firstPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c1",
										"author": "user1",
										"body":   "comment1",
									},
								},
								map[string]any{
									"data": map[string]any{
										"id":     "c2",
										"author": "user2",
										"body":   "comment2",
									},
								},
							},
						},
					},
				}
				testMock.SetupComments(firstPageData)

				// Second page with 2 more comments
				secondPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c3",
										"author": "user3",
										"body":   "comment3",
									},
								},
								map[string]any{
									"data": map[string]any{
										"id":     "c4",
										"author": "user4",
										"body":   "comment4",
									},
								},
							},
						},
					},
				}
				testMock.SetupPageResponse("t1_c2", secondPageData)

				// Request exactly 3 comments
				comments, err := post.GetCommentsAfter(ctx, nil, 3)
				Expect(err).NotTo(HaveOccurred())
				Expect(comments).To(HaveLen(3))
				Expect(comments[0].ID).To(Equal("c1"))
				Expect(comments[1].ID).To(Equal("c2"))
				Expect(comments[2].ID).To(Equal("c3"))
			})

			It("handles over limit pagination", func() {
				// Single comment available
				singleCommentData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c1",
										"author": "user1",
										"body":   "comment1",
									},
								},
							},
						},
					},
				}
				testMock.SetupComments(singleCommentData)

				// Empty second page
				emptyPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{},
						},
					},
				}
				testMock.SetupPageResponse("t1_c1", emptyPageData)

				// Request more than available
				comments, err := post.GetCommentsAfter(ctx, nil, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(comments).To(HaveLen(1)) // Should only return what's available
				Expect(comments[0].ID).To(Equal("c1"))
			})

			It("handles under limit pagination", func() {
				// Multiple comments available
				multiCommentData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c1",
										"author": "user1",
										"body":   "comment1",
									},
								},
								map[string]any{
									"data": map[string]any{
										"id":     "c2",
										"author": "user2",
										"body":   "comment2",
									},
								},
								map[string]any{
									"data": map[string]any{
										"id":     "c3",
										"author": "user3",
										"body":   "comment3",
									},
								},
							},
						},
					},
				}
				testMock.SetupComments(multiCommentData)

				// Request fewer than available
				comments, err := post.GetCommentsAfter(ctx, nil, 2)
				Expect(err).NotTo(HaveOccurred())
				Expect(comments).To(HaveLen(2))
				Expect(comments[0].ID).To(Equal("c1"))
				Expect(comments[1].ID).To(Equal("c2"))
			})

			It("handles network errors mid-pagination", func() {
				// First page works
				firstPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c1",
										"author": "user1",
										"body":   "comment1",
									},
								},
							},
						},
					},
				}
				testMock.SetupComments(firstPageData)

				// Second page fails
				networkErr := errors.New("network timeout")
				testMock.SetupPageError("t1_c1", networkErr)

				comments, err := post.GetCommentsAfter(ctx, nil, 5)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("network timeout"))
				Expect(comments).To(BeNil())
			})

			It("handles very large limit values", func() {
				// Setup single comment
				singleCommentData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c1",
										"author": "user1",
										"body":   "comment1",
									},
								},
							},
						},
					},
				}
				testMock.SetupComments(singleCommentData)

				// Empty next page
				emptyPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{},
						},
					},
				}
				testMock.SetupPageResponse("t1_c1", emptyPageData)

				// Very large limit
				comments, err := post.GetCommentsAfter(ctx, nil, 1000000)
				Expect(err).NotTo(HaveOccurred())
				Expect(comments).To(HaveLen(1))
				Expect(comments[0].ID).To(Equal("c1"))
			})

			It("handles zero limit (fetch all)", func() {
				// First page
				firstPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c1",
										"author": "user1",
										"body":   "comment1",
									},
								},
							},
						},
					},
				}
				testMock.SetupComments(firstPageData)

				// Second page
				secondPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c2",
										"author": "user2",
										"body":   "comment2",
									},
								},
							},
						},
					},
				}
				testMock.SetupPageResponse("t1_c1", secondPageData)

				// Empty third page
				emptyPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{},
						},
					},
				}
				testMock.SetupPageResponse("t1_c2", emptyPageData)

				// Zero limit should fetch all
				comments, err := post.GetCommentsAfter(ctx, nil, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(comments).To(HaveLen(2))
				Expect(comments[0].ID).To(Equal("c1"))
				Expect(comments[1].ID).To(Equal("c2"))
			})

			It("verifies proper handling of duplicate items", func() {
				// First page with duplicated comment in API response
				firstPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":     "c1",
										"author": "user1",
										"body":   "comment1",
									},
								},
								map[string]any{
									"data": map[string]any{
										"id":     "c1", // Duplicate ID
										"author": "user1",
										"body":   "comment1 duplicate",
									},
								},
							},
						},
					},
				}
				testMock.SetupComments(firstPageData)

				// Set up empty page to stop pagination after duplicates
				emptyPageData := []any{
					map[string]any{}, // First element (post data)
					map[string]any{   // Second element (comments data)
						"data": map[string]any{
							"children": []any{},
						},
					},
				}
				testMock.SetupPageResponse("t1_c1", emptyPageData)

				comments, err := post.GetCommentsAfter(ctx, nil, 5)
				Expect(err).NotTo(HaveOccurred())
				// Should include all duplicates as returned by API (client doesn't deduplicate)
				Expect(comments).To(HaveLen(2))
				Expect(comments[0].ID).To(Equal("c1"))
				Expect(comments[1].ID).To(Equal("c1"))
				// Verify they have different content to confirm they're different responses
				Expect(comments[0].Body).To(Equal("comment1"))
				Expect(comments[1].Body).To(Equal("comment1 duplicate"))
			})
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
