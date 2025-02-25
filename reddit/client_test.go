package reddit_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
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

// MockTransport implements http.RoundTripper for testing
type MockTransport struct {
	Responses []*http.Response
	Errors    []error
	Calls     int
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.Calls >= len(m.Responses) {
		return nil, fmt.Errorf("no more responses")
	}
	resp := m.Responses[m.Calls]
	err := m.Errors[m.Calls]
	m.Calls++
	return resp, err
}

// Ensure mockAuth implements the necessary interface
var _ interface {
	EnsureValidToken(context.Context) error
} = &mockAuth{}

// Ensure MockTransport implements http.RoundTripper
var _ http.RoundTripper = &MockTransport{}

var _ = Describe("Client", func() {
	var (
		auth        *reddit.Auth
		client      *reddit.Client
		mock        *mockAuth
		transport   *MockTransport
		mockResp    *http.Response
		mockRespErr error
	)

	BeforeEach(func() {
		mock = &mockAuth{
			token:     "test_token",
			expiresAt: time.Now().Add(time.Hour),
		}
		auth = &reddit.Auth{
			Token:     mock.token,
			ExpiresAt: mock.expiresAt,
		}

		// Setup mock transport with default success response
		mockResp = &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data": {"children": [{"data": {"id": "123", "title": "Test Post"}}], "after": "t3_456"}}`)),
		}
		mockRespErr = nil
		transport = &MockTransport{
			Responses: []*http.Response{mockResp},
			Errors:    []error{mockRespErr},
		}

		var err error
		client, err = reddit.NewClient(auth)
		Expect(err).NotTo(HaveOccurred())
		client.SetHTTPClient(&http.Client{Transport: transport})
	})

	AfterEach(func() {
		if mockResp != nil && mockResp.Body != nil {
			mockResp.Body.Close()
		}
	})

	Describe("NewClient", func() {
		It("creates a client with default options", func() {
			client, err := reddit.NewClient(&reddit.Auth{})
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("applies custom options", func() {
			client, err := reddit.NewClient(&reddit.Auth{},
				reddit.WithUserAgent("custom-agent"),
				reddit.WithRateLimit(30, 3),
				reddit.WithTimeout(5*time.Second),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("returns error with nil auth", func() {
			client, err := reddit.NewClient(nil)
			Expect(err).To(Equal(reddit.ErrMissingCredentials))
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
			transport.Responses = []*http.Response{firstResp, secondResp}
			transport.Errors = []error{nil, nil}

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
			transport.Responses = []*http.Response{resp}
			transport.Errors = []error{nil}

			posts, err := client.GetPostsAfter(context.Background(), "test", nil, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(2))
			Expect(posts[0].ID).To(Equal("1"))
			Expect(posts[1].ID).To(Equal("2"))
		})

		It("uses the after parameter correctly", func() {
			afterPost := &reddit.Post{ID: "123"}
			_, err := client.GetPostsAfter(context.Background(), "test", afterPost, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(transport.Calls).To(Equal(1))
		})
	})
})
