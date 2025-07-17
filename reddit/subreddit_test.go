package reddit_test

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Subreddit", func() {
	var (
		transport  *reddit.TestTransport
		client     *reddit.Client
		subreddit  *reddit.Subreddit
		ctx        context.Context
		mockClient *http.Client
	)

	BeforeEach(func() {
		transport = reddit.NewTestTransport()
		mockClient = &http.Client{Transport: transport}

		// Create auth with our mock transport
		auth, err := reddit.NewAuth("test_client_id", "test_client_secret",
			reddit.WithAuthTransport(transport))
		Expect(err).NotTo(HaveOccurred())

		// Create client with auth and custom transport
		client, err = reddit.NewClient(auth,
			reddit.WithHTTPClient(mockClient),
			reddit.WithUserAgent("test-bot/1.0"),
		)
		Expect(err).NotTo(HaveOccurred())

		subreddit = reddit.NewSubreddit("golang", client)
		ctx = context.Background()

		// Set up default post response
		transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
			"data": map[string]any{
				"children": []any{
					map[string]any{
						"data": map[string]any{
							"title":        "First Post",
							"selftext":     "Content 1",
							"url":          "https://example.com/1",
							"created_utc":  float64(time.Now().Unix()),
							"subreddit":    "golang",
							"id":           "post1",
							"score":        float64(100),
							"num_comments": float64(10),
						},
					},
					map[string]any{
						"data": map[string]any{
							"title":        "Second Post",
							"selftext":     "Content 2",
							"url":          "https://example.com/2",
							"created_utc":  float64(time.Now().Unix()),
							"subreddit":    "golang",
							"id":           "post2",
							"score":        float64(200),
							"num_comments": float64(20),
						},
					},
				},
				"after": "t3_post2",
			},
		}))
	})

	Describe("NewSubreddit", func() {
		It("creates a new subreddit instance", func() {
			sub := reddit.NewSubreddit("test", client)
			Expect(sub).NotTo(BeNil())
			Expect(sub.Name).To(Equal("test"))
		})
	})

	Describe("GetPosts", func() {
		BeforeEach(func() {
			// Mock response for /r/golang.json
			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{
						map[string]any{
							"data": map[string]any{
								"title":        "First Post",
								"selftext":     "Content 1",
								"url":          "https://example.com/1",
								"created_utc":  float64(time.Now().Unix()),
								"subreddit":    "golang",
								"id":           "post1",
								"score":        float64(100),
								"num_comments": float64(10),
							},
						},
						map[string]any{
							"data": map[string]any{
								"title":        "Second Post",
								"selftext":     "Content 2",
								"url":          "https://example.com/2",
								"created_utc":  float64(time.Now().Unix()),
								"subreddit":    "golang",
								"id":           "post2",
								"score":        float64(200),
								"num_comments": float64(20),
							},
						},
					},
					"after": "t3_post2",
				},
			}))
		})

		It("fetches posts with the specified parameters", func() {
			posts, err := subreddit.GetPosts(ctx, reddit.WithSort("new"), reddit.WithSubredditLimit(2))
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(2))
			Expect(posts[0].Title).To(Equal("First Post"))
			Expect(posts[1].Title).To(Equal("Second Post"))
		})

		It("respects the limit", func() {
			posts, err := subreddit.GetPosts(ctx, reddit.WithSort("new"), reddit.WithSubredditLimit(1))
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(1))
			Expect(posts[0].Title).To(Equal("First Post"))
		})

		It("applies subreddit options", func() {
			timestamp := time.Now().Unix()
			posts, err := subreddit.GetPosts(ctx, reddit.WithSort("new"), reddit.WithSubredditLimit(1), reddit.WithAfterTimestamp(timestamp))
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(1))
			Expect(posts[0].Title).To(Equal("First Post"))
		})
	})

	Describe("GetPostsAfter", func() {
		BeforeEach(func() {
			// Mock response for /r/golang.json?after=t3_post1
			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{
						map[string]any{
							"data": map[string]any{
								"title":        "Second Post",
								"selftext":     "Content 2",
								"url":          "https://example.com/2",
								"created_utc":  float64(time.Now().Unix()),
								"subreddit":    "golang",
								"id":           "post2",
								"score":        float64(200),
								"num_comments": float64(20),
							},
						},
						map[string]any{
							"data": map[string]any{
								"title":        "Third Post",
								"selftext":     "Content 3",
								"url":          "https://example.com/3",
								"created_utc":  float64(time.Now().Unix()),
								"subreddit":    "golang",
								"id":           "post3",
								"score":        float64(300),
								"num_comments": float64(30),
							},
						},
					},
					"after": "",
				},
			}))
		})

		It("fetches posts after the specified post", func() {
			afterPost := &reddit.Post{ID: "post1"}
			posts, err := subreddit.GetPostsAfter(ctx, afterPost, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(2))
			Expect(posts[0].Title).To(Equal("Second Post"))
			Expect(posts[1].Title).To(Equal("Third Post"))
		})

		It("stops fetching when a page has no posts", func() {
			// Clear existing responses
			transport = reddit.NewTestTransport()
			mockClient.Transport = transport

			// Set up response with empty page but with after token
			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    "t3_post5",
				},
			}))

			afterPost := &reddit.Post{ID: "post1"}
			posts, err := subreddit.GetPostsAfter(ctx, afterPost, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(0))
		})

		Context("GetPostsAfter edge cases", func() {
			BeforeEach(func() {
				transport.Reset()
			})

			It("handles pagination with nil after parameter", func() {
				// Setup first page
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "First Post",
									"selftext":     "Content 1",
									"url":          "https://example.com/1",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1",
									"score":        float64(100),
									"num_comments": float64(10),
								},
							},
							map[string]any{
								"data": map[string]any{
									"title":        "Second Post",
									"selftext":     "Content 2",
									"url":          "https://example.com/2",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post2",
									"score":        float64(200),
									"num_comments": float64(20),
								},
							},
						},
						"after": "",
					},
				}))

				posts, err := subreddit.GetPostsAfter(ctx, nil, 2)
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(2))
				Expect(posts[0].ID).To(Equal("post1"))
				Expect(posts[1].ID).To(Equal("post2"))
			})

			It("respects exact limit with pagination", func() {
				// First page with 2 posts
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "First Post",
									"selftext":     "Content 1",
									"url":          "https://example.com/1",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1",
									"score":        float64(100),
									"num_comments": float64(10),
								},
							},
							map[string]any{
								"data": map[string]any{
									"title":        "Second Post",
									"selftext":     "Content 2",
									"url":          "https://example.com/2",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post2",
									"score":        float64(200),
									"num_comments": float64(20),
								},
							},
						},
						"after": "t3_post2",
					},
				}))

				// Second page with 2 more posts
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "Third Post",
									"selftext":     "Content 3",
									"url":          "https://example.com/3",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post3",
									"score":        float64(300),
									"num_comments": float64(30),
								},
							},
							map[string]any{
								"data": map[string]any{
									"title":        "Fourth Post",
									"selftext":     "Content 4",
									"url":          "https://example.com/4",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post4",
									"score":        float64(400),
									"num_comments": float64(40),
								},
							},
						},
						"after": "",
					},
				}))

				// Request exactly 3 posts
				afterPost := &reddit.Post{ID: "post0"}
				posts, err := subreddit.GetPostsAfter(ctx, afterPost, 3)
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(3))
				Expect(posts[0].ID).To(Equal("post1"))
				Expect(posts[1].ID).To(Equal("post2"))
				Expect(posts[2].ID).To(Equal("post3"))
			})

			It("handles over limit pagination", func() {
				// Single post available
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "Only Post",
									"selftext":     "Content 1",
									"url":          "https://example.com/1",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1",
									"score":        float64(100),
									"num_comments": float64(10),
								},
							},
						},
						"after": "t3_post1",
					},
				}))

				// Empty second page
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    "",
					},
				}))

				// Request more than available
				afterPost := &reddit.Post{ID: "post0"}
				posts, err := subreddit.GetPostsAfter(ctx, afterPost, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(1)) // Should only return what's available
				Expect(posts[0].ID).To(Equal("post1"))
			})

			It("handles under limit pagination", func() {
				// Multiple posts available
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "First Post",
									"selftext":     "Content 1",
									"url":          "https://example.com/1",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1",
									"score":        float64(100),
									"num_comments": float64(10),
								},
							},
							map[string]any{
								"data": map[string]any{
									"title":        "Second Post",
									"selftext":     "Content 2",
									"url":          "https://example.com/2",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post2",
									"score":        float64(200),
									"num_comments": float64(20),
								},
							},
							map[string]any{
								"data": map[string]any{
									"title":        "Third Post",
									"selftext":     "Content 3",
									"url":          "https://example.com/3",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post3",
									"score":        float64(300),
									"num_comments": float64(30),
								},
							},
						},
						"after": "",
					},
				}))

				// Request fewer than available
				afterPost := &reddit.Post{ID: "post0"}
				posts, err := subreddit.GetPostsAfter(ctx, afterPost, 2)
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(2))
				Expect(posts[0].ID).To(Equal("post1"))
				Expect(posts[1].ID).To(Equal("post2"))
			})

			It("handles network errors mid-pagination", func() {
				// First page works
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "First Post",
									"selftext":     "Content 1",
									"url":          "https://example.com/1",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1",
									"score":        float64(100),
									"num_comments": float64(10),
								},
							},
						},
						"after": "t3_post1",
					},
				}))

				// Second call fails
				transport.SetErrorOnCall(3, errors.New("network timeout")) // Call 3 because call 1 is auth, call 2 is first page

				afterPost := &reddit.Post{ID: "post0"}
				posts, err := subreddit.GetPostsAfter(ctx, afterPost, 5)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("network timeout"))
				Expect(posts).To(BeNil())
			})

			It("handles very large limit values", func() {
				// Setup single post
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "Only Post",
									"selftext":     "Content 1",
									"url":          "https://example.com/1",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1",
									"score":        float64(100),
									"num_comments": float64(10),
								},
							},
						},
						"after": "t3_post1",
					},
				}))

				// Empty next page
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    "",
					},
				}))

				// Very large limit
				afterPost := &reddit.Post{ID: "post0"}
				posts, err := subreddit.GetPostsAfter(ctx, afterPost, 1000000)
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(1))
				Expect(posts[0].ID).To(Equal("post1"))
			})

			It("handles zero limit (fetch all)", func() {
				// First page
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "First Post",
									"selftext":     "Content 1",
									"url":          "https://example.com/1",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1",
									"score":        float64(100),
									"num_comments": float64(10),
								},
							},
						},
						"after": "t3_post1",
					},
				}))

				// Second page
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "Second Post",
									"selftext":     "Content 2",
									"url":          "https://example.com/2",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post2",
									"score":        float64(200),
									"num_comments": float64(20),
								},
							},
						},
						"after": "t3_post2",
					},
				}))

				// Empty third page
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    "",
					},
				}))

				// Zero limit should fetch all
				afterPost := &reddit.Post{ID: "post0"}
				posts, err := subreddit.GetPostsAfter(ctx, afterPost, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(2))
				Expect(posts[0].ID).To(Equal("post1"))
				Expect(posts[1].ID).To(Equal("post2"))
			})

			It("verifies proper handling of duplicate items", func() {
				// First page with duplicated post in API response
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "First Post",
									"selftext":     "Content 1",
									"url":          "https://example.com/1",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1",
									"score":        float64(100),
									"num_comments": float64(10),
								},
							},
							map[string]any{
								"data": map[string]any{
									"title":        "First Post Duplicate",
									"selftext":     "Content 1 duplicate",
									"url":          "https://example.com/1-dup",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1", // Duplicate ID
									"score":        float64(101),
									"num_comments": float64(11),
								},
							},
						},
						"after": "t3_post1",
					},
				}))

				// Second page returns the same post again
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "First Post Again",
									"selftext":     "Content 1 again",
									"url":          "https://example.com/1-again",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1", // Same ID again
									"score":        float64(102),
									"num_comments": float64(12),
								},
							},
						},
						"after": "",
					},
				}))

				afterPost := &reddit.Post{ID: "post0"}
				posts, err := subreddit.GetPostsAfter(ctx, afterPost, 5)
				Expect(err).NotTo(HaveOccurred())
				// Should include all duplicates as returned by API (client doesn't deduplicate)
				Expect(posts).To(HaveLen(3))
				Expect(posts[0].ID).To(Equal("post1"))
				Expect(posts[1].ID).To(Equal("post1"))
				Expect(posts[2].ID).To(Equal("post1"))
				// Verify they have different content to confirm they're different responses
				Expect(posts[0].Title).To(Equal("First Post"))
				Expect(posts[1].Title).To(Equal("First Post Duplicate"))
				Expect(posts[2].Title).To(Equal("First Post Again"))
			})

			It("handles pagination call count verification", func() {
				// First page
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "First Post",
									"selftext":     "Content 1",
									"url":          "https://example.com/1",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post1",
									"score":        float64(100),
									"num_comments": float64(10),
								},
							},
						},
						"after": "t3_post1",
					},
				}))

				// Second page
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"title":        "Second Post",
									"selftext":     "Content 2",
									"url":          "https://example.com/2",
									"created_utc":  float64(time.Now().Unix()),
									"subreddit":    "golang",
									"id":           "post2",
									"score":        float64(200),
									"num_comments": float64(20),
								},
							},
						},
						"after": "",
					},
				}))

				afterPost := &reddit.Post{ID: "post0"}
				posts, err := subreddit.GetPostsAfter(ctx, afterPost, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(2))

				// Should make 3 calls: 1 for auth, 2 for API requests
				Expect(transport.GetCallCount()).To(Equal(3))
				history := transport.GetCallHistory()
				Expect(len(history)).To(Equal(3))
				// Verify the API calls contain proper pagination parameters
				Expect(history[1]).To(ContainSubstring("/r/golang.json"))
				Expect(history[2]).To(ContainSubstring("/r/golang.json"))
				Expect(history[2]).To(ContainSubstring("after=t3_post1"))
			})
		})
	})
})
