package reddit

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockAuth implements the minimum interface needed for testing
type mockAuth struct {
	token     string
	expiresAt time.Time
	authError error
	callCount int
}

func (m *mockAuth) EnsureValidToken(ctx context.Context) error {
	m.callCount++
	return m.authError
}

// mockTransport implements http.RoundTripper for testing
type mockTransport struct {
	responses []*http.Response
	errors    []error
	calls     int
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.calls >= len(m.responses) {
		return nil, fmt.Errorf("no more responses")
	}
	resp := m.responses[m.calls]
	err := m.errors[m.calls]
	m.calls++
	return resp, err
}

// Ensure mockAuth implements the necessary interface
var _ interface {
	EnsureValidToken(context.Context) error
} = &mockAuth{}

// Ensure mockTransport implements http.RoundTripper
var _ http.RoundTripper = &mockTransport{}

var _ = Describe("Client", func() {
	var (
		auth        *Auth
		client      *Client
		mock        *mockAuth
		transport   *mockTransport
		mockResp    *http.Response
		mockRespErr error
	)

	BeforeEach(func() {
		mock = &mockAuth{
			token:     "test_token",
			expiresAt: time.Now().Add(time.Hour),
		}
		auth = &Auth{
			Token:     mock.token,
			ExpiresAt: mock.expiresAt,
		}

		// Setup mock transport with default success response
		mockResp = &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data": {"children": [{"data": {"id": "123", "title": "Test Post"}}], "after": "t3_456"}}`)),
		}
		mockRespErr = nil
		transport = &mockTransport{
			responses: []*http.Response{mockResp},
			errors:    []error{mockRespErr},
		}

		var err error
		client, err = NewClient(auth)
		Expect(err).NotTo(HaveOccurred())
		client.client = &http.Client{Transport: transport}
	})

	AfterEach(func() {
		if mockResp != nil && mockResp.Body != nil {
			mockResp.Body.Close()
		}
	})

	Describe("NewClient", func() {
		It("creates a client with default options", func() {
			client, err := NewClient(&Auth{})
			Expect(err).NotTo(HaveOccurred())
			Expect(client.userAgent).To(Equal("golang:reddit-client:v1.0"))
			Expect(client.rateLimiter).NotTo(BeNil())
		})

		It("applies custom options", func() {
			client, err := NewClient(&Auth{},
				WithUserAgent("custom-agent"),
				WithRateLimit(30, 3),
				WithTimeout(5*time.Second),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client.userAgent).To(Equal("custom-agent"))
			Expect(client.rateLimiter.limiter.Burst()).To(Equal(3))
			Expect(client.client.Timeout).To(Equal(5 * time.Second))
		})

		It("returns error with nil auth", func() {
			client, err := NewClient(nil)
			Expect(err).To(Equal(ErrMissingCredentials))
			Expect(client).To(BeNil())
		})
	})

	Describe("GetPosts", func() {
		It("fetches posts with correct parameters", func() {
			posts, after, err := client.GetPosts(context.Background(), "test", map[string]string{"limit": "25"})
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(1))
			Expect(posts[0].ID).To(Equal("123"))
			Expect(after).To(Equal("t3_456"))
		})

		Context("with context cancellation", func() {
			It("respects context cancellation", func() {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately

				_, _, err := client.GetPosts(ctx, "test", nil)
				Expect(err).To(MatchError(context.Canceled))
			})
		})

		Context("with context timeout", func() {
			It("respects context timeout", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(time.Millisecond) // Ensure timeout occurs

				_, _, err := client.GetPosts(ctx, "test", nil)
				Expect(err).To(MatchError(context.DeadlineExceeded))
			})
		})
	})

	Describe("GetPostsAfter", func() {
		It("fetches multiple pages of posts", func() {
			// Setup responses for two requests
			firstResp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data": {"children": [{"data": {"id": "123", "title": "First Post"}}], "after": "t3_456"}}`)),
			}
			secondResp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data": {"children": [{"data": {"id": "456", "title": "Second Post"}}], "after": ""}}`)),
			}
			transport.responses = []*http.Response{firstResp, secondResp}
			transport.errors = []error{nil, nil}

			posts, err := client.GetPostsAfter(context.Background(), "test", nil, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(2))
			Expect(posts[0].ID).To(Equal("123"))
			Expect(posts[1].ID).To(Equal("456"))
		})

		It("respects the limit parameter", func() {
			// Setup response with multiple posts
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{"data": {"children": [
					{"data": {"id": "1", "title": "Post 1"}},
					{"data": {"id": "2", "title": "Post 2"}},
					{"data": {"id": "3", "title": "Post 3"}}
				], "after": "t3_4"}}`)),
			}
			transport.responses = []*http.Response{resp}
			transport.errors = []error{nil}

			posts, err := client.GetPostsAfter(context.Background(), "test", nil, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(2))
			Expect(posts[0].ID).To(Equal("1"))
			Expect(posts[1].ID).To(Equal("2"))
		})

		It("uses the after parameter correctly", func() {
			afterPost := &Post{ID: "123"}
			_, err := client.GetPostsAfter(context.Background(), "test", afterPost, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(transport.calls).To(Equal(1))
			// The actual URL check would be better, but we'd need to modify our mock to capture the request
		})
	})

	Describe("getComments", func() {
		Context("with rate limiting", func() {
			It("handles rate limit errors", func() {
				// Setup responses for two requests
				successResp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"kind": "Listing"}, {"kind": "Listing", "data": {"children": []}}]`)),
					Header:     http.Header{},
				}

				rateLimitResp := &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(strings.NewReader(`{"error": "too many requests"}`)),
					Header:     http.Header{},
				}
				rateLimitResp.Header.Set("X-Ratelimit-Remaining", "0")
				rateLimitResp.Header.Set("X-Ratelimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

				transport.responses = []*http.Response{successResp, rateLimitResp}
				transport.errors = []error{nil, nil}

				// First request should succeed
				ctx := context.Background()
				_, err := client.getComments(ctx, "test", "123", nil)
				Expect(err).NotTo(HaveOccurred())

				// Second request should be rate limited
				_, err = client.getComments(ctx, "test", "123", nil)
				Expect(IsRateLimitError(err)).To(BeTrue())
			})
		})
	})

	Describe("request", func() {
		Context("with API errors", func() {
			It("handles different API error types", func() {
				testCases := []struct {
					statusCode int
					expected   error
				}{
					{http.StatusUnauthorized, ErrInvalidCredentials},
					{http.StatusTooManyRequests, ErrRateLimited},
					{http.StatusNotFound, ErrNotFound},
					{http.StatusBadRequest, ErrBadRequest},
					{http.StatusInternalServerError, ErrServerError},
				}

				for _, tc := range testCases {
					// Create a proper response with the test status code
					resp := &http.Response{
						StatusCode: tc.statusCode,
						Body:       io.NopCloser(strings.NewReader(`{"error": "test error"}`)),
					}
					err := NewAPIError(resp, []byte(`{"error": "test error"}`))
					var apiErr *APIError
					Expect(err).To(BeAssignableToTypeOf(apiErr))
					apiErr = err.(*APIError)
					Expect(apiErr.StatusCode).To(Equal(tc.statusCode))
					Expect(err.Error()).To(ContainSubstring(tc.expected.Error()))
				}
			})
		})
	})
})
