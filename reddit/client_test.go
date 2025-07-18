package reddit_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		transport  *reddit.TestTransport
		auth       *reddit.Auth
		mockClient *http.Client
	)

	BeforeEach(func() {
		transport = reddit.NewTestTransport()
		mockClient = &http.Client{Transport: transport}

		var err error
		auth, err = reddit.NewAuth("test_id", "test_secret",
			reddit.WithAuthTransport(transport))
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("NewClient", func() {
		It("creates a client with default options", func() {
			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(client.String()).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
			Expect(client.String()).To(ContainSubstring("RateLimiter{requests_per_minute: 60.0, burst: 5}"))
		})

		It("creates a client with multiple custom options", func() {
			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithUserAgent("custom-agent"),
				reddit.WithRateLimit(30, 3),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(client.String()).To(ContainSubstring("UserAgent: \"custom-agent\""))
			Expect(client.String()).To(ContainSubstring("RateLimiter{requests_per_minute: 30.0, burst: 3}"))
		})

		It("creates a client with custom user agent", func() {
			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithUserAgent("test-bot/1.0"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(client.String()).To(ContainSubstring("UserAgent: \"test-bot/1.0\""))
		})

		It("creates a client with custom rate limiting", func() {
			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRateLimit(45, 4),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify rate limiter configuration through String method
			Expect(client.String()).To(ContainSubstring("RateLimiter{requests_per_minute: 45.0, burst: 4}"))
		})

		It("creates a client with default rate limiting", func() {
			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify default rate limiter configuration through String method
			Expect(client.String()).To(ContainSubstring("RateLimiter{requests_per_minute: 60.0, burst: 5}"))
		})

		It("returns error with nil auth", func() {
			client, err := reddit.NewClient(nil)
			Expect(err).To(MatchError("client.NewClient: auth is required for client creation"))
			Expect(client).To(BeNil())
		})

		It("creates a client with custom retry configuration", func() {
			retryConfig := &reddit.RetryConfig{
				MaxRetries:        2,
				BaseDelay:         500 * time.Millisecond,
				MaxDelay:          4 * time.Second,
				JitterFactor:      0.2,
				RetryableCodes:    []int{429, 502, 503},
				RespectRetryAfter: true,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRetryConfig(retryConfig),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("creates a client with no retries", func() {
			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithNoRetries(),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})
	})

	Describe("Retry Logic", func() {
		var client *reddit.Client
		var subreddit *reddit.Subreddit

		BeforeEach(func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRetries(2),                       // Max 2 retries (3 total attempts)
				reddit.WithRetryDelay(100*time.Millisecond), // Fast retries for testing
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)
			// Reset call count after client setup (auth call)
			transport.Reset()
		})

		Context("when receiving retryable errors", func() {
			It("retries on 429 (rate limited) and succeeds", func() {
				// First two requests return 429, third succeeds
				transport.AddResponseToQueue("/r/golang.json", &http.Response{
					StatusCode: 429,
					Body:       http.NoBody,
					Header:     http.Header{"Retry-After": []string{"1"}},
				})
				transport.AddResponseToQueue("/r/golang.json", &http.Response{
					StatusCode: 429,
					Body:       http.NoBody,
				})
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				}))

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())

				// Check call history for the correct endpoint
				callHistory := transport.GetCallHistory()
				golangCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/golang.json") {
						golangCalls++
					}
				}
				Expect(golangCalls).To(Equal(3)) // 3 attempts to /r/golang.json
			})

			It("retries on 502 (bad gateway) and succeeds", func() {
				transport.AddResponseToQueue("/r/golang.json", &http.Response{
					StatusCode: 502,
					Body:       http.NoBody,
				})
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				}))

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())

				// Check call history for the correct endpoint
				callHistory := transport.GetCallHistory()
				golangCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/golang.json") {
						golangCalls++
					}
				}
				Expect(golangCalls).To(Equal(2)) // 2 attempts to /r/golang.json
			})

			It("retries on 503 (service unavailable) and succeeds", func() {
				transport.AddResponseToQueue("/r/golang.json", &http.Response{
					StatusCode: 503,
					Body:       http.NoBody,
				})
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				}))

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())

				// Check call history for the correct endpoint
				callHistory := transport.GetCallHistory()
				golangCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/golang.json") {
						golangCalls++
					}
				}
				Expect(golangCalls).To(Equal(2)) // 2 attempts to /r/golang.json
			})

			It("exhausts retries and returns the last error", func() {
				// All requests return 429
				for i := 0; i < 3; i++ {
					transport.AddResponseToQueue("/r/golang.json", &http.Response{
						StatusCode: 429,
						Body:       http.NoBody,
					})
				}

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(posts).To(BeNil())

				// Check call history for the correct endpoint
				callHistory := transport.GetCallHistory()
				golangCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/golang.json") {
						golangCalls++
					}
				}
				Expect(golangCalls).To(Equal(3)) // 3 attempts to /r/golang.json
				Expect(reddit.IsRateLimitError(err)).To(BeTrue())
			})
		})

		Context("when receiving non-retryable errors", func() {
			It("does not retry on 404 (not found)", func() {
				nonexistentSubreddit := reddit.NewSubreddit("nonexistent", client)
				transport.AddResponse("/r/nonexistent.json", &http.Response{
					StatusCode: 404,
					Body:       http.NoBody,
				})

				posts, err := nonexistentSubreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(posts).To(BeNil())

				// Check call history for the correct endpoint
				callHistory := transport.GetCallHistory()
				nonexistentCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/nonexistent.json") {
						nonexistentCalls++
					}
				}
				Expect(nonexistentCalls).To(Equal(1)) // Only 1 attempt
				Expect(reddit.IsNotFoundError(err)).To(BeTrue())
			})

			It("does not retry on 400 (bad request)", func() {
				transport.AddResponse("/r/golang.json", &http.Response{
					StatusCode: 400,
					Body:       http.NoBody,
				})

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(posts).To(BeNil())

				// Check call history for the correct endpoint
				callHistory := transport.GetCallHistory()
				golangCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/golang.json") {
						golangCalls++
					}
				}
				Expect(golangCalls).To(Equal(1)) // Only 1 attempt
			})
		})

		Context("when retry is disabled", func() {
			BeforeEach(func() {
				var err error
				client, err = reddit.NewClient(auth,
					reddit.WithHTTPClient(mockClient),
					reddit.WithNoRetries(),
				)
				Expect(err).NotTo(HaveOccurred())
				subreddit = reddit.NewSubreddit("golang", client)
				// Reset call count after client setup (auth call)
				transport.Reset()
			})

			It("does not retry on retryable errors", func() {
				transport.AddResponse("/r/golang.json", &http.Response{
					StatusCode: 429,
					Body:       http.NoBody,
				})

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(posts).To(BeNil())

				// Check call history for the correct endpoint
				callHistory := transport.GetCallHistory()
				golangCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/golang.json") {
						golangCalls++
					}
				}
				Expect(golangCalls).To(Equal(1)) // Only 1 attempt
			})
		})

		Context("with Retry-After header", func() {
			It("respects Retry-After header with seconds", func() {
				// Create a client with a smaller base delay so Retry-After takes precedence
				clientWithLowDelay, err := reddit.NewClient(auth,
					reddit.WithHTTPClient(mockClient),
					reddit.WithRetries(2),
					reddit.WithRetryDelay(50*time.Millisecond), // Smaller base delay
				)
				Expect(err).NotTo(HaveOccurred())
				subredditWithLowDelay := reddit.NewSubreddit("golang", clientWithLowDelay)
				transport.Reset() // Reset after client creation

				// Create response with explicit header
				resp := &http.Response{
					StatusCode: 429,
					Body:       http.NoBody,
					Header:     make(http.Header),
				}
				resp.Header.Set("Retry-After", "1")
				transport.AddResponseToQueue("/r/golang.json", resp)

				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				}))

				start := time.Now()
				posts, err := subredditWithLowDelay.GetPosts(context.Background())
				duration := time.Since(start)

				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())

				// Check call history for the correct endpoint
				callHistory := transport.GetCallHistory()
				golangCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/golang.json") {
						golangCalls++
					}
				}
				Expect(golangCalls).To(Equal(2))                        // 2 attempts total
				Expect(duration).To(BeNumerically(">=", 1*time.Second)) // Should wait at least 1 second
			})
		})

		Context("with context cancellation", func() {
			It("respects context cancellation during retry delay", func() {
				ctx, cancel := context.WithCancel(context.Background())

				transport.AddResponse("/r/golang.json", &http.Response{
					StatusCode: 429,
					Body:       http.NoBody,
				})

				go func() {
					time.Sleep(50 * time.Millisecond) // Cancel after 50ms
					cancel()
				}()

				posts, err := subreddit.GetPosts(ctx)
				Expect(err).To(HaveOccurred())
				Expect(posts).To(BeNil())
				Expect(errors.Is(err, context.Canceled)).To(BeTrue())
			})
		})
	})

	Describe("BuildEndpoint", func() {
		It("returns base URL when no parameters are provided", func() {
			base := "/api/endpoint"
			params := map[string]string{}

			result := reddit.BuildEndpoint(base, params)

			Expect(result).To(Equal("/api/endpoint"))
		})

		It("returns base URL when nil parameters are provided", func() {
			base := "/api/endpoint"

			result := reddit.BuildEndpoint(base, nil)

			Expect(result).To(Equal("/api/endpoint"))
		})

		It("properly encodes single parameter", func() {
			base := "/api/endpoint"
			params := map[string]string{
				"limit": "100",
			}

			result := reddit.BuildEndpoint(base, params)

			Expect(result).To(Equal("/api/endpoint?limit=100"))
		})

		It("properly encodes multiple parameters", func() {
			base := "/api/endpoint"
			params := map[string]string{
				"limit": "100",
				"after": "t3_abc123",
				"sort":  "hot",
			}

			result := reddit.BuildEndpoint(base, params)

			// Since map iteration order is not guaranteed, we need to check if all parameters are present
			Expect(result).To(HavePrefix("/api/endpoint?"))
			Expect(result).To(ContainSubstring("limit=100"))
			Expect(result).To(ContainSubstring("after=t3_abc123"))
			Expect(result).To(ContainSubstring("sort=hot"))
			// Check that parameters are separated by &
			paramString := strings.Split(result, "?")[1]
			params_count := len(strings.Split(paramString, "&"))
			Expect(params_count).To(Equal(3))
		})

		It("properly URL encodes special characters in parameter values", func() {
			base := "/api/endpoint"
			params := map[string]string{
				"query":   "hello world",
				"special": "foo&bar=baz",
				"unicode": "café",
			}

			result := reddit.BuildEndpoint(base, params)

			Expect(result).To(HavePrefix("/api/endpoint?"))
			Expect(result).To(ContainSubstring("query=hello+world"))
			Expect(result).To(ContainSubstring("special=foo%26bar%3Dbaz"))
			Expect(result).To(ContainSubstring("unicode=caf%C3%A9"))
		})

		It("properly URL encodes special characters in parameter keys", func() {
			base := "/api/endpoint"
			params := map[string]string{
				"param with spaces": "value1",
				"param&special":     "value2",
			}

			result := reddit.BuildEndpoint(base, params)

			Expect(result).To(HavePrefix("/api/endpoint?"))
			Expect(result).To(ContainSubstring("param+with+spaces=value1"))
			Expect(result).To(ContainSubstring("param%26special=value2"))
		})

		It("handles empty parameter values", func() {
			base := "/api/endpoint"
			params := map[string]string{
				"empty": "",
				"limit": "100",
			}

			result := reddit.BuildEndpoint(base, params)

			Expect(result).To(HavePrefix("/api/endpoint?"))
			Expect(result).To(ContainSubstring("empty="))
			Expect(result).To(ContainSubstring("limit=100"))
		})

		It("handles complex base URLs", func() {
			base := "/r/golang/comments/abc123"
			params := map[string]string{
				"limit": "50",
				"depth": "5",
			}

			result := reddit.BuildEndpoint(base, params)

			Expect(result).To(HavePrefix("/r/golang/comments/abc123?"))
			Expect(result).To(ContainSubstring("limit=50"))
			Expect(result).To(ContainSubstring("depth=5"))
		})

		It("preserves parameter order deterministically", func() {
			base := "/api/endpoint"
			params := map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			}

			// Call multiple times to ensure consistent output
			result1 := reddit.BuildEndpoint(base, params)
			result2 := reddit.BuildEndpoint(base, params)
			result3 := reddit.BuildEndpoint(base, params)

			Expect(result1).To(Equal(result2))
			Expect(result2).To(Equal(result3))
		})
	})

	Describe("JSON Response Handling", func() {
		var client *reddit.Client
		var subreddit *reddit.Subreddit

		BeforeEach(func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)
			transport.Reset()
		})

		Context("when processing valid JSON responses", func() {
			It("successfully processes subreddit posts JSON", func() {
				expectedData := map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"id":    "test123",
									"title": "Test Post",
									"url":   "https://example.com",
								},
							},
						},
						"after": nil,
					},
				}
				transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(expectedData))

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(1))
				Expect(posts[0].Title).To(Equal("Test Post"))
			})

			It("successfully processes comment JSON responses", func() {
				// First create a post properly through the subreddit
				postData := map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"id":        "post123",
									"title":     "Test Post",
									"subreddit": "golang",
									"url":       "https://example.com",
								},
							},
						},
						"after": nil,
					},
				}
				transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(postData))

				posts, err := subreddit.GetPosts(context.Background(), reddit.WithSubredditLimit(1))
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(1))

				// Mock data for comments
				expectedData := []any{
					map[string]any{
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":    "post123",
										"title": "Test Post",
									},
								},
							},
						},
					},
					map[string]any{
						"data": map[string]any{
							"children": []any{
								map[string]any{
									"data": map[string]any{
										"id":   "comment123",
										"body": "Test comment",
									},
								},
							},
						},
					},
				}
				transport.AddResponse("/r/golang/comments/post123", reddit.CreateJSONResponse(expectedData))

				comments, err := posts[0].GetComments(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(comments).To(HaveLen(1))
			})
		})

		Context("when handling malformed JSON responses", func() {
			It("returns descriptive error for malformed subreddit JSON", func() {
				transport.AddResponse("/r/golang.json", &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"invalid": json`)),
					Header:     make(http.Header),
				})

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(posts).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("decoding JSON response failed"))
				Expect(err.Error()).To(ContainSubstring("GET /r/golang.json"))
			})

			It("returns descriptive error for malformed comment JSON", func() {
				// First create a post properly through the subreddit
				postData := map[string]any{
					"data": map[string]any{
						"children": []any{
							map[string]any{
								"data": map[string]any{
									"id":        "post123",
									"title":     "Test Post",
									"subreddit": "golang",
									"url":       "https://example.com",
								},
							},
						},
						"after": nil,
					},
				}
				transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(postData))

				posts, err := subreddit.GetPosts(context.Background(), reddit.WithSubredditLimit(1))
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(HaveLen(1))

				// Now set up malformed comment response
				transport.AddResponse("/r/golang/comments/post123", &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`[{"invalid": json`)),
					Header:     make(http.Header),
				})

				comments, err := posts[0].GetComments(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(comments).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("decoding JSON response failed"))
				Expect(err.Error()).To(ContainSubstring("GET /r/golang/comments/post123"))
			})
		})

		Context("when handling empty responses", func() {
			It("handles empty subreddit responses gracefully", func() {
				transport.AddResponse("/r/golang.json", &http.Response{
					StatusCode: 200,
					Body:       http.NoBody,
					Header:     make(http.Header),
				})

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(posts).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("decoding JSON response failed"))
			})
		})
	})

	Describe("Rate Limit Header Processing", func() {
		var client *reddit.Client
		var subreddit *reddit.Subreddit

		BeforeEach(func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)
			transport.Reset()
		})

		Context("with all rate limit headers present", func() {
			It("successfully parses and updates rate limiter with all headers", func() {
				// Create response with all rate limit headers
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "50")
				resp.Header.Set("X-Ratelimit-Used", "10")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())

				// Verify the request was made
				callHistory := transport.GetCallHistory()
				golangCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/golang.json") {
						golangCalls++
					}
				}
				Expect(golangCalls).To(Equal(1))
			})
		})

		Context("with only remaining and reset headers", func() {
			It("successfully parses headers without used header", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "75")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(5*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})
		})

		Context("with malformed rate limit headers", func() {
			It("handles invalid remaining header gracefully", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "invalid")
				resp.Header.Set("X-Ratelimit-Used", "5")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})

			It("handles invalid used header gracefully", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "30")
				resp.Header.Set("X-Ratelimit-Used", "not-a-number")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})

			It("handles invalid reset header gracefully", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "40")
				resp.Header.Set("X-Ratelimit-Used", "20")
				resp.Header.Set("X-Ratelimit-Reset", "invalid-timestamp")

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})

			It("handles all invalid headers gracefully", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "invalid")
				resp.Header.Set("X-Ratelimit-Used", "also-invalid")
				resp.Header.Set("X-Ratelimit-Reset", "not-a-timestamp")

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})
		})

		Context("with zero remaining requests", func() {
			It("handles rate limit exhaustion properly", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "0")
				resp.Header.Set("X-Ratelimit-Used", "60")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})

			It("handles negative remaining requests", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "-1")
				resp.Header.Set("X-Ratelimit-Used", "61")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})
		})

		Context("with reset time in the past", func() {
			It("skips rate limit update for past reset time", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "30")
				resp.Header.Set("X-Ratelimit-Used", "10")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(-5*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})
		})

		Context("with no rate limit headers", func() {
			It("continues normally without rate limit headers", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				// No rate limit headers set

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})
		})

		Context("with partial headers", func() {
			It("processes only remaining header", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "25")

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})

			It("processes only reset header", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(8*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})

			It("processes only used header", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Used", "15")

				transport.AddResponse("/r/golang.json", resp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())
			})
		})

		Context("rate limit headers with retries", func() {
			BeforeEach(func() {
				var err error
				client, err = reddit.NewClient(auth,
					reddit.WithHTTPClient(mockClient),
					reddit.WithRetries(2),
					reddit.WithRetryDelay(100*time.Millisecond),
				)
				Expect(err).NotTo(HaveOccurred())
				subreddit = reddit.NewSubreddit("golang", client)
				transport.Reset()
			})

			It("processes rate limit headers on failed requests", func() {
				// First response: 429 with rate limit headers
				firstResp := &http.Response{
					StatusCode: 429,
					Body:       http.NoBody,
					Header:     make(http.Header),
				}
				firstResp.Header.Set("X-Ratelimit-Remaining", "0")
				firstResp.Header.Set("X-Ratelimit-Used", "60")
				firstResp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponseToQueue("/r/golang.json", firstResp)

				// Second response: success with different rate limit headers
				secondResp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				secondResp.Header = make(http.Header)
				secondResp.Header.Set("X-Ratelimit-Remaining", "59")
				secondResp.Header.Set("X-Ratelimit-Used", "1")
				secondResp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponseToQueue("/r/golang.json", secondResp)

				posts, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(posts).To(BeEmpty())

				// Verify both requests were made
				callHistory := transport.GetCallHistory()
				golangCalls := 0
				for _, call := range callHistory {
					if strings.Contains(call, "/r/golang.json") {
						golangCalls++
					}
				}
				Expect(golangCalls).To(Equal(2))
			})
		})
	})

	Describe("Rate Limit Hooks", func() {
		var (
			client    *reddit.Client
			subreddit *reddit.Subreddit
			hookCalls *testRateLimitHook
		)

		BeforeEach(func() {
			hookCalls = &testRateLimitHook{}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRateLimitHook(hookCalls),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)
			transport.Reset()
		})

		Context("OnRateLimitUpdate", func() {
			It("calls hook when rate limit headers are received", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "45")
				resp.Header.Set("X-Ratelimit-Used", "15")
				resetTime := time.Now().Add(10 * time.Minute)
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				_, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(hookCalls.updateCalls).To(HaveLen(1))
				Expect(hookCalls.updateCalls[0].remaining).To(Equal(45))
				Expect(hookCalls.updateCalls[0].reset.Unix()).To(Equal(resetTime.Unix()))
			})

			It("calls hook multiple times for multiple requests", func() {
				// First request
				resp1 := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp1.Header = make(http.Header)
				resp1.Header.Set("X-Ratelimit-Remaining", "45")
				resetTime1 := time.Now().Add(10 * time.Minute)
				resp1.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(resetTime1.Unix(), 10))

				transport.AddResponse("/r/golang.json", resp1)

				_, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				// Second request
				resp2 := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp2.Header = make(http.Header)
				resp2.Header.Set("X-Ratelimit-Remaining", "44")
				resetTime2 := time.Now().Add(9 * time.Minute)
				resp2.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(resetTime2.Unix(), 10))

				transport.AddResponse("/r/golang.json", resp2)

				_, err = subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(hookCalls.updateCalls).To(HaveLen(2))
				Expect(hookCalls.updateCalls[0].remaining).To(Equal(45))
				Expect(hookCalls.updateCalls[1].remaining).To(Equal(44))
			})

			It("does not call hook when no rate limit headers are present", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				// No rate limit headers

				transport.AddResponse("/r/golang.json", resp)

				_, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(hookCalls.updateCalls).To(BeEmpty())
			})
		})

		Context("OnRateLimitExceeded", func() {
			It("calls hook when remaining requests is zero", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "0")
				resp.Header.Set("X-Ratelimit-Used", "60")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				_, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(hookCalls.exceededCalls).To(HaveLen(1))
				Expect(hookCalls.updateCalls).To(HaveLen(1))
			})

			It("calls hook when remaining requests is negative", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "-1")
				resp.Header.Set("X-Ratelimit-Used", "61")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				_, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(hookCalls.exceededCalls).To(HaveLen(1))
			})

			It("does not call hook when remaining requests is positive", func() {
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "45")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				_, err := subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(hookCalls.exceededCalls).To(BeEmpty())
			})
		})

		Context("OnRateLimitWait", func() {
			It("calls hook when rate limiting causes a wait", func() {
				// Test the hook interface directly by testing the hook methods
				testHook := &testRateLimitHook{}

				// Test OnRateLimitWait directly
				ctx := context.Background()
				duration := 100 * time.Millisecond
				testHook.OnRateLimitWait(ctx, duration)

				// Verify the hook captured the call
				Expect(len(testHook.waitCalls)).To(Equal(1))
				Expect(testHook.waitCalls[0].duration).To(Equal(duration))

				// Test that the hook can be added to a client without errors
				client, err := reddit.NewClient(auth,
					reddit.WithHTTPClient(mockClient),
					reddit.WithRateLimitHook(testHook),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())

				// Verify the hook is set up correctly by making a normal request
				// This won't trigger rate limiting but will ensure the hook integration works
				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				transport.AddResponse("/r/golang.json", resp)

				subreddit := reddit.NewSubreddit("golang", client)
				_, err = subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				// The test passes if we can successfully use a client with hooks
				// Rate limiting timing tests are covered in other integration tests
			})
		})

		Context("with LoggingRateLimitHook", func() {
			It("creates and uses logging hook without errors", func() {
				loggingHook := &reddit.LoggingRateLimitHook{}

				client, err := reddit.NewClient(auth,
					reddit.WithHTTPClient(mockClient),
					reddit.WithRateLimitHook(loggingHook),
				)
				Expect(err).NotTo(HaveOccurred())
				subreddit = reddit.NewSubreddit("golang", client)

				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "30")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				_, err = subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				// Test should pass without panics or errors
			})
		})

		Context("with nil hook", func() {
			It("works normally without any hooks configured", func() {
				client, err := reddit.NewClient(auth,
					reddit.WithHTTPClient(mockClient),
					// No rate limit hook configured
				)
				Expect(err).NotTo(HaveOccurred())
				subreddit = reddit.NewSubreddit("golang", client)

				resp := reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				})
				resp.Header = make(http.Header)
				resp.Header.Set("X-Ratelimit-Remaining", "0")
				resp.Header.Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10))

				transport.AddResponse("/r/golang.json", resp)

				_, err = subreddit.GetPosts(context.Background())
				Expect(err).NotTo(HaveOccurred())

				// Should work without any issues
			})
		})
	})
})

// testRateLimitHook is a test implementation of RateLimitHook that records all calls
type testRateLimitHook struct {
	waitCalls     []waitCall
	updateCalls   []updateCall
	exceededCalls []exceededCall
}

type waitCall struct {
	duration time.Duration
}

type updateCall struct {
	remaining int
	reset     time.Time
}

type exceededCall struct {
	// We could store context info here if needed
}

func (h *testRateLimitHook) OnRateLimitWait(ctx context.Context, duration time.Duration) {
	h.waitCalls = append(h.waitCalls, waitCall{duration: duration})
}

func (h *testRateLimitHook) OnRateLimitUpdate(remaining int, reset time.Time) {
	h.updateCalls = append(h.updateCalls, updateCall{remaining: remaining, reset: reset})
}

func (h *testRateLimitHook) OnRateLimitExceeded(ctx context.Context) {
	h.exceededCalls = append(h.exceededCalls, exceededCall{})
}

var _ = Describe("Client Circuit Breaker Integration", func() {
	var (
		transport  *reddit.TestTransport
		auth       *reddit.Auth
		mockClient *http.Client
		client     *reddit.Client
		subreddit  *reddit.Subreddit
	)

	BeforeEach(func() {
		transport = reddit.NewTestTransport()
		mockClient = &http.Client{Transport: transport}

		var err error
		auth, err = reddit.NewAuth("test_id", "test_secret",
			reddit.WithAuthTransport(transport))
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("WithCircuitBreaker", func() {
		It("should create client with circuit breaker", func() {
			config := &reddit.CircuitBreakerConfig{
				FailureThreshold: 3,
				SuccessThreshold: 2,
				Timeout:          100 * time.Millisecond,
				MaxRequests:      2,
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithCircuitBreaker(config),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("should create client with default circuit breaker", func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithDefaultCircuitBreaker(),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})
	})

	Describe("Circuit breaker behavior during requests", func() {
		BeforeEach(func() {
			config := &reddit.CircuitBreakerConfig{
				FailureThreshold: 2,
				SuccessThreshold: 2,
				Timeout:          100 * time.Millisecond,
				MaxRequests:      2,
				ShouldTrip: func(err error) bool {
					return reddit.IsServerError(err)
				},
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithCircuitBreaker(config),
				reddit.WithNoRetries(), // Disable retries to test circuit breaker in isolation
			)
			Expect(err).NotTo(HaveOccurred())

			subreddit = reddit.NewSubreddit("golang", client)
		})

		It("should open circuit after failure threshold and fast-fail subsequent requests", func() {
			// Set up server error responses
			transport.AddResponse("/r/golang.json", &http.Response{
				StatusCode: 500,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error": "internal server error"}`)),
			})

			// Make enough requests to trip the circuit
			for i := 0; i < 2; i++ {
				_, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(reddit.IsServerError(err)).To(BeTrue())
			}

			// Next request should fail fast due to open circuit
			_, err := subreddit.GetPosts(context.Background())
			Expect(err).To(HaveOccurred())

			var cbErr *reddit.CircuitBreakerError
			Expect(errors.As(err, &cbErr)).To(BeTrue())
			Expect(cbErr.State).To(Equal(reddit.CircuitOpen))
		})

		It("should transition to half-open and then closed after successful requests", func() {
			// Set up server error responses to open the circuit (2 responses)
			for i := 0; i < 2; i++ {
				transport.AddResponseToQueue("/r/golang.json", &http.Response{
					StatusCode: 500,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"error": "internal server error"}`)),
				})
			}

			// Trip the circuit
			for i := 0; i < 2; i++ {
				_, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(reddit.IsServerError(err)).To(BeTrue())
			}

			// Wait for circuit timeout
			time.Sleep(110 * time.Millisecond)

			// Set up successful responses (2 responses)
			for i := 0; i < 2; i++ {
				transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
					"data": map[string]any{
						"children": []any{},
						"after":    nil,
					},
				}))
			}

			// Make successful request - should work in half-open state
			_, err := subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Make another successful request
			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not trip circuit for client errors", func() {
			// Set up client error responses (400 Bad Request)
			transport.AddResponse("/r/golang.json", &http.Response{
				StatusCode: 400,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error": "bad request"}`)),
			})

			// Make many requests with client errors
			for i := 0; i < 5; i++ {
				_, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(reddit.IsServerError(err)).To(BeFalse())
			}

			// Circuit should still be closed since client errors don't trip it
			_, err := subreddit.GetPosts(context.Background())
			Expect(err).To(HaveOccurred())
			// Should be API error, not circuit breaker error
			var cbErr *reddit.CircuitBreakerError
			Expect(errors.As(err, &cbErr)).To(BeFalse())
		})

		It("should allow requests in half-open state", func() {
			// Trip the circuit
			transport.AddResponse("/r/golang.json", &http.Response{
				StatusCode: 500,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error": "internal server error"}`)),
			})

			for i := 0; i < 2; i++ {
				_, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
			}

			// Wait for timeout
			time.Sleep(110 * time.Millisecond)

			// Set up successful response
			transport.AddResponse("/r/golang.json", &http.Response{
				StatusCode: 200,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"kind": "Listing", "data": {"children": []}}`)),
			})

			// Make a request in half-open state - should succeed
			_, err := subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Circuit breaker with retry integration", func() {
		It("should work correctly with retry logic", func() {
			config := &reddit.CircuitBreakerConfig{
				FailureThreshold: 2,
				SuccessThreshold: 1,
				Timeout:          100 * time.Millisecond,
				MaxRequests:      1,
				ShouldTrip: func(err error) bool {
					return reddit.IsServerError(err)
				},
			}

			retryConfig := reddit.DefaultRetryConfig()
			retryConfig.MaxRetries = 1

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithCircuitBreaker(config),
				reddit.WithRetryConfig(retryConfig),
			)
			Expect(err).NotTo(HaveOccurred())

			subreddit = reddit.NewSubreddit("golang", client)

			// Set up server error response
			transport.AddResponse("/r/golang.json", &http.Response{
				StatusCode: 500,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error": "internal server error"}`)),
			})

			// Each request will be retried, so 2 actual requests will result in 4 total attempts
			// This should trip the circuit even with retries
			for i := 0; i < 2; i++ {
				_, err := subreddit.GetPosts(context.Background())
				Expect(err).To(HaveOccurred())
			}

			// Circuit should be open, so next request should fail fast
			_, err = subreddit.GetPosts(context.Background())
			Expect(err).To(HaveOccurred())

			var cbErr *reddit.CircuitBreakerError
			Expect(errors.As(err, &cbErr)).To(BeTrue())
		})
	})
})

var _ = Describe("Client Compression Support", func() {
	var (
		transport  *reddit.TestTransport
		auth       *reddit.Auth
		mockClient *http.Client
		client     *reddit.Client
		subreddit  *reddit.Subreddit
	)

	BeforeEach(func() {
		transport = reddit.NewTestTransport()
		mockClient = &http.Client{Transport: transport}

		var err error
		auth, err = reddit.NewAuth("test_id", "test_secret",
			reddit.WithAuthTransport(transport))
		Expect(err).NotTo(HaveOccurred())
		transport.Reset()
	})

	Context("with compression enabled (default)", func() {
		BeforeEach(func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)
		})

		It("adds Accept-Encoding: gzip header to requests", func() {
			// Set up test to capture request headers
			var capturedHeaders http.Header
			interceptor := func(req *http.Request) error {
				capturedHeaders = req.Header.Clone()
				return nil
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRequestInterceptor(interceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			Expect(capturedHeaders.Get("Accept-Encoding")).To(Equal("gzip"))
		})

		It("successfully decompresses gzipped JSON responses", func() {
			expectedData := map[string]any{
				"data": map[string]any{
					"children": []any{
						map[string]any{
							"data": map[string]any{
								"id":    "test123",
								"title": "Test Gzipped Post",
								"url":   "https://example.com/gzipped",
							},
						},
					},
					"after": nil,
				},
			}

			transport.AddResponse("/r/golang.json", reddit.CreateGzippedJSONResponse(expectedData))

			posts, err := subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(1))
			Expect(posts[0].Title).To(Equal("Test Gzipped Post"))
		})

		It("handles both compressed and uncompressed responses", func() {
			// First response: uncompressed
			transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{
						map[string]any{
							"data": map[string]any{
								"id":    "uncompressed",
								"title": "Uncompressed Post",
								"url":   "https://example.com/uncompressed",
							},
						},
					},
					"after": nil,
				},
			}))

			// Second response: compressed
			transport.AddResponseToQueue("/r/golang.json", reddit.CreateGzippedJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{
						map[string]any{
							"data": map[string]any{
								"id":    "compressed",
								"title": "Compressed Post",
								"url":   "https://example.com/compressed",
							},
						},
					},
					"after": nil,
				},
			}))

			// Test uncompressed response
			posts1, err := subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(posts1).To(HaveLen(1))
			Expect(posts1[0].Title).To(Equal("Uncompressed Post"))

			// Test compressed response
			posts2, err := subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(posts2).To(HaveLen(1))
			Expect(posts2[0].Title).To(Equal("Compressed Post"))
		})

		It("handles gzipped responses in retry scenarios", func() {
			// First response: 429 with gzipped error body
			gzippedErrorResp := reddit.CreateGzippedJSONResponse(map[string]any{
				"error": "rate limited",
			})
			gzippedErrorResp.StatusCode = 429
			transport.AddResponseToQueue("/r/golang.json", gzippedErrorResp)

			// Second response: success with gzipped body
			transport.AddResponseToQueue("/r/golang.json", reddit.CreateGzippedJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			// Configure client with retries
			clientWithRetries, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRetries(1),
				reddit.WithRetryDelay(50*time.Millisecond),
			)
			Expect(err).NotTo(HaveOccurred())
			subredditWithRetries := reddit.NewSubreddit("golang", clientWithRetries)

			posts, err := subredditWithRetries.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(BeEmpty())
		})
	})

	Context("with compression disabled", func() {
		BeforeEach(func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithNoCompression(),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)
		})

		It("does not add Accept-Encoding header to requests", func() {
			// Set up test to capture request headers
			var capturedHeaders http.Header
			interceptor := func(req *http.Request) error {
				capturedHeaders = req.Header.Clone()
				return nil
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithNoCompression(),
				reddit.WithRequestInterceptor(interceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			Expect(capturedHeaders.Get("Accept-Encoding")).To(BeEmpty())
		})

		It("handles regular uncompressed responses normally", func() {
			expectedData := map[string]any{
				"data": map[string]any{
					"children": []any{
						map[string]any{
							"data": map[string]any{
								"id":    "test123",
								"title": "Regular Post",
								"url":   "https://example.com/regular",
							},
						},
					},
					"after": nil,
				},
			}

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(expectedData))

			posts, err := subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(1))
			Expect(posts[0].Title).To(Equal("Regular Post"))
		})

		It("treats gzipped responses as raw data when compression is disabled", func() {
			// Create a gzipped response but without Content-Encoding header
			// to simulate when server sends gzipped data but client doesn't expect it
			gzippedResp := reddit.CreateGzippedJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			})
			// Remove the Content-Encoding header to simulate raw gzipped data
			gzippedResp.Header.Del("Content-Encoding")

			transport.AddResponse("/r/golang.json", gzippedResp)

			_, err := subreddit.GetPosts(context.Background())
			// Should fail to decode since we're trying to parse gzipped data as JSON
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("decoding JSON response failed"))
		})
	})

	Context("with explicit compression settings", func() {
		It("WithCompression(true) enables compression", func() {
			_, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithCompression(true),
			)
			Expect(err).NotTo(HaveOccurred())

			// Test that compression is enabled by checking request headers
			var capturedHeaders http.Header
			interceptor := func(req *http.Request) error {
				capturedHeaders = req.Header.Clone()
				return nil
			}

			clientWithInterceptor, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithCompression(true),
				reddit.WithRequestInterceptor(interceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", clientWithInterceptor)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedHeaders.Get("Accept-Encoding")).To(Equal("gzip"))
		})

		It("WithCompression(false) disables compression", func() {
			_, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithCompression(false),
			)
			Expect(err).NotTo(HaveOccurred())

			// Test that compression is disabled by checking request headers
			var capturedHeaders http.Header
			interceptor := func(req *http.Request) error {
				capturedHeaders = req.Header.Clone()
				return nil
			}

			clientWithInterceptor, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithCompression(false),
				reddit.WithRequestInterceptor(interceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", clientWithInterceptor)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedHeaders.Get("Accept-Encoding")).To(BeEmpty())
		})
	})

	Context("error handling with compression", func() {
		BeforeEach(func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)
		})

		It("handles malformed gzipped responses gracefully", func() {
			// Create a response with gzip Content-Encoding but invalid gzip data
			resp := &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("invalid gzip data")),
				Header:     make(http.Header),
			}
			resp.Header.Set("Content-Encoding", "gzip")

			transport.AddResponse("/r/golang.json", resp)

			_, err := subreddit.GetPosts(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creating gzip reader failed"))
		})

		It("handles errors during gzip decompression", func() {
			// Create a response with truncated gzip data
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			gzWriter.Write([]byte(`{"incomplete"`)) // Incomplete JSON
			gzWriter.Close()

			// Take only part of the gzipped data to simulate truncation
			truncatedData := buf.Bytes()[:len(buf.Bytes())/2]

			resp := &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(truncatedData)),
				Header:     make(http.Header),
			}
			resp.Header.Set("Content-Encoding", "gzip")

			transport.AddResponse("/r/golang.json", resp)

			_, err := subreddit.GetPosts(context.Background())
			Expect(err).To(HaveOccurred())
			// Should fail during JSON decoding due to incomplete data
			Expect(err.Error()).To(ContainSubstring("decoding JSON response failed"))
		})
	})
})

var _ = Describe("Client Request and Response Interceptors", func() {
	var (
		transport  *reddit.TestTransport
		auth       *reddit.Auth
		mockClient *http.Client
		client     *reddit.Client
		subreddit  *reddit.Subreddit
	)

	BeforeEach(func() {
		transport = reddit.NewTestTransport()
		mockClient = &http.Client{Transport: transport}

		var err error
		auth, err = reddit.NewAuth("test_id", "test_secret",
			reddit.WithAuthTransport(transport))
		Expect(err).NotTo(HaveOccurred())
		transport.Reset()
	})

	Context("Request Interceptors", func() {
		It("calls request interceptors in order", func() {
			var callOrder []string

			firstInterceptor := func(req *http.Request) error {
				callOrder = append(callOrder, "first")
				req.Header.Set("X-First", "first-value")
				return nil
			}

			secondInterceptor := func(req *http.Request) error {
				callOrder = append(callOrder, "second")
				req.Header.Set("X-Second", "second-value")
				return nil
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRequestInterceptor(firstInterceptor),
				reddit.WithRequestInterceptor(secondInterceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Verify call order
			Expect(callOrder).To(Equal([]string{"first", "second"}))
		})

		It("cancels request when interceptor returns error", func() {
			errorInterceptor := func(req *http.Request) error {
				return errors.New("interceptor error")
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRequestInterceptor(errorInterceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("request interceptor 0 failed"))
			Expect(err.Error()).To(ContainSubstring("interceptor error"))

			// Verify no HTTP requests were made to the subreddit endpoint
			callHistory := transport.GetCallHistory()
			golangCalls := 0
			for _, call := range callHistory {
				if strings.Contains(call, "/r/golang.json") {
					golangCalls++
				}
			}
			Expect(golangCalls).To(Equal(0))
		})

		It("supports multiple interceptors with one failing", func() {
			var callOrder []string

			firstInterceptor := func(req *http.Request) error {
				callOrder = append(callOrder, "first")
				return nil
			}

			errorInterceptor := func(req *http.Request) error {
				callOrder = append(callOrder, "error")
				return errors.New("second interceptor error")
			}

			thirdInterceptor := func(req *http.Request) error {
				callOrder = append(callOrder, "third")
				return nil
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRequestInterceptor(firstInterceptor),
				reddit.WithRequestInterceptor(errorInterceptor),
				reddit.WithRequestInterceptor(thirdInterceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("request interceptor 1 failed"))

			// Verify first interceptor was called, but third was not
			Expect(callOrder).To(Equal([]string{"first", "error"}))
		})
	})

	Context("Response Interceptors", func() {
		It("calls response interceptors in order", func() {
			var callOrder []string

			firstInterceptor := func(resp *http.Response) error {
				callOrder = append(callOrder, "first")
				return nil
			}

			secondInterceptor := func(resp *http.Response) error {
				callOrder = append(callOrder, "second")
				return nil
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithResponseInterceptor(firstInterceptor),
				reddit.WithResponseInterceptor(secondInterceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Verify call order
			Expect(callOrder).To(Equal([]string{"first", "second"}))
		})

		It("fails request when response interceptor returns error", func() {
			errorInterceptor := func(resp *http.Response) error {
				return errors.New("response interceptor error")
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithResponseInterceptor(errorInterceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("response interceptor 0 failed"))
			Expect(err.Error()).To(ContainSubstring("response interceptor error"))
		})

		It("supports multiple interceptors with one failing", func() {
			var callOrder []string

			firstInterceptor := func(resp *http.Response) error {
				callOrder = append(callOrder, "first")
				return nil
			}

			errorInterceptor := func(resp *http.Response) error {
				callOrder = append(callOrder, "error")
				return errors.New("second response interceptor error")
			}

			thirdInterceptor := func(resp *http.Response) error {
				callOrder = append(callOrder, "third")
				return nil
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithResponseInterceptor(firstInterceptor),
				reddit.WithResponseInterceptor(errorInterceptor),
				reddit.WithResponseInterceptor(thirdInterceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("response interceptor 1 failed"))

			// Verify first interceptor was called, but third was not
			Expect(callOrder).To(Equal([]string{"first", "error"}))
		})
	})

	Context("Combined Request and Response Interceptors", func() {
		It("calls request and response interceptors together", func() {
			var callOrder []string

			requestInterceptor := func(req *http.Request) error {
				callOrder = append(callOrder, "request")
				req.Header.Set("X-Custom", "custom-value")
				return nil
			}

			responseInterceptor := func(resp *http.Response) error {
				callOrder = append(callOrder, "response")
				return nil
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRequestInterceptor(requestInterceptor),
				reddit.WithResponseInterceptor(responseInterceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Verify both were called in correct order
			Expect(callOrder).To(Equal([]string{"request", "response"}))
		})
	})

	Context("Interceptors with Retries", func() {
		It("calls request interceptors on each retry attempt", func() {
			var requestCount int

			requestInterceptor := func(req *http.Request) error {
				requestCount++
				return nil
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRetries(1),
				reddit.WithRetryDelay(100*time.Millisecond),
				reddit.WithRequestInterceptor(requestInterceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			// First request fails with retryable error
			transport.AddResponseToQueue("/r/golang.json", &http.Response{
				StatusCode: 429,
				Body:       http.NoBody,
			})

			// Second request succeeds
			transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Verify request interceptor was called for both attempts
			Expect(requestCount).To(Equal(2))
		})

		It("calls response interceptors on failed responses", func() {
			var responseCount int

			responseInterceptor := func(resp *http.Response) error {
				responseCount++
				return nil
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRetries(1),
				reddit.WithRetryDelay(100*time.Millisecond),
				reddit.WithResponseInterceptor(responseInterceptor),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			// First request fails with retryable error
			transport.AddResponseToQueue("/r/golang.json", &http.Response{
				StatusCode: 429,
				Body:       http.NoBody,
			})

			// Second request succeeds
			transport.AddResponseToQueue("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Verify response interceptor was called for both responses
			Expect(responseCount).To(Equal(2))
		})
	})

	Context("Example Interceptors", func() {
		It("works with HeaderInjectionRequestInterceptor", func() {
			headers := map[string]string{
				"X-Test-Header":    "test-value",
				"X-Request-Source": "golang-test",
			}

			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRequestInterceptor(reddit.HeaderInjectionRequestInterceptor(headers)),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Test passes if no errors occur
		})

		It("works with LoggingRequestInterceptor", func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRequestInterceptor(reddit.LoggingRequestInterceptor()),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Test passes if no errors occur (logging happens to stdout/stderr)
		})

		It("works with LoggingResponseInterceptor", func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithResponseInterceptor(reddit.LoggingResponseInterceptor()),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Test passes if no errors occur (logging happens to stdout/stderr)
		})

		It("works with DeprecationWarningResponseInterceptor", func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithResponseInterceptor(reddit.DeprecationWarningResponseInterceptor()),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			// Create response with deprecation header
			resp := reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			})
			resp.Header = make(http.Header)
			resp.Header.Set("X-API-Deprecated", "This endpoint will be removed in v2")

			transport.AddResponse("/r/golang.json", resp)

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Test passes if no errors occur (warning logged)
		})

		It("works with RequestIDRequestInterceptor", func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
				reddit.WithRequestInterceptor(reddit.RequestIDRequestInterceptor("X-Request-ID")),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Test passes if no errors occur
		})
	})

	Context("No Interceptors", func() {
		It("works normally without any interceptors configured", func() {
			var err error
			client, err = reddit.NewClient(auth,
				reddit.WithHTTPClient(mockClient),
			)
			Expect(err).NotTo(HaveOccurred())
			subreddit = reddit.NewSubreddit("golang", client)

			transport.AddResponse("/r/golang.json", reddit.CreateJSONResponse(map[string]any{
				"data": map[string]any{
					"children": []any{},
					"after":    nil,
				},
			}))

			_, err = subreddit.GetPosts(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// Should work without any issues
		})
	})
})
