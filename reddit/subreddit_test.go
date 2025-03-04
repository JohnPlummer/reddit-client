package reddit_test

import (
	"context"
	"time"
	"net/http"

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
		client, err = reddit.NewClient(
			reddit.WithAuth(auth),
			reddit.WithHTTPClient(mockClient),
			reddit.WithUserAgent("test-bot/1.0"),
		)
		Expect(err).NotTo(HaveOccurred())

		subreddit = reddit.NewSubreddit("golang", client)
		ctx = context.Background()

		// Set up default post response
		transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]interface{}{
			"data": map[string]interface{}{
				"children": []interface{}{
					map[string]interface{}{
						"data": map[string]interface{}{
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
					map[string]interface{}{
						"data": map[string]interface{}{
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
			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]interface{}{
				"data": map[string]interface{}{
					"children": []interface{}{
						map[string]interface{}{
							"data": map[string]interface{}{
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
						map[string]interface{}{
							"data": map[string]interface{}{
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
			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]interface{}{
				"data": map[string]interface{}{
					"children": []interface{}{
						map[string]interface{}{
							"data": map[string]interface{}{
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
			Expect(posts).To(HaveLen(1))
			Expect(posts[0].Title).To(Equal("Third Post"))
		})
	})
})