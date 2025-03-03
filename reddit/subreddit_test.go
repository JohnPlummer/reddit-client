package reddit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockTransport implements http.RoundTripper for testing
type mockTransport struct {
	responses map[string]*http.Response
	err       error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Special handling for auth endpoint
	if req.URL.Host == "www.reddit.com" && req.URL.Path == "/api/v1/access_token" {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"access_token": "test_token",
				"token_type": "bearer",
				"expires_in": 3600
			}`))),
		}, nil
	}

	// For API endpoints, try to match the path
	if resp, ok := m.responses[req.URL.Path]; ok {
		// Return a new response with a fresh body for each request
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		resp.Body.Close()
		return &http.Response{
			StatusCode: resp.StatusCode,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}

	// Default response for unmatched paths
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}, nil
}

func jsonResponse(data interface{}) *http.Response {
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

var _ = Describe("Subreddit", func() {
	var (
		transport  *mockTransport
		client     *reddit.Client
		subreddit  *reddit.Subreddit
		ctx        context.Context
		mockClient *http.Client
	)

	BeforeEach(func() {
		transport = &mockTransport{
			responses: make(map[string]*http.Response),
		}
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
		transport.responses["/r/golang.json"] = jsonResponse(map[string]interface{}{
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
		})
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
			transport.responses["/r/golang.json"] = jsonResponse(map[string]interface{}{
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
			})
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
			transport.responses["/r/golang.json"] = jsonResponse(map[string]interface{}{
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
			})
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
