package reddit_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

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
			auth      *reddit.Auth
			client    *reddit.Client
			transport *MockTransport
		)

		BeforeEach(func() {
			auth = &reddit.Auth{
				Token:     "test_token",
				ExpiresAt: time.Now().Add(time.Hour),
			}

			// Setup mock transport with default success response for posts
			postsResp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data": {"children": [{"data": {"id": "123", "title": "Test Post", "subreddit": "golang"}}]}}`)),
			}

			transport = &MockTransport{
				Responses: []*http.Response{postsResp},
				Errors:    []error{nil},
			}

			var err error
			client, err = reddit.NewClient(auth)
			Expect(err).NotTo(HaveOccurred())
			client.SetHTTPClient(&http.Client{Transport: transport})
		})

		It("fetches comments for a post", func() {
			// Setup mock response for comments
			commentsResp := &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{},
					{"data": {"children": [{"data": {"author": "user1", "body": "comment1", "created_utc": 1234567890, "id": "c1"}}]}}
				]`)),
			}
			transport.Responses = append(transport.Responses, commentsResp)
			transport.Errors = append(transport.Errors, nil)

			// Create a post through the public interface
			posts, _, err := client.GetPosts(context.Background(), "golang", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(1))

			post := posts[0]
			comments, err := post.GetComments(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(comments).To(HaveLen(1))
			Expect(comments[0].Author).To(Equal("user1"))
			Expect(comments[0].Body).To(Equal("comment1"))
		})

		It("handles comment filtering by timestamp", func() {
			// Setup mock response for comments
			commentsResp := &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{},
					{"data": {"children": [{"data": {"author": "user1", "body": "comment1", "created_utc": 1000, "id": "c1"}}]}}
				]`)),
			}
			transport.Responses = append(transport.Responses, commentsResp)
			transport.Errors = append(transport.Errors, nil)

			// Create a post through the public interface
			posts, _, err := client.GetPosts(context.Background(), "golang", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(1))

			post := posts[0]
			comments, err := post.GetComments(context.Background(), reddit.CommentsSince(500))

			Expect(err).NotTo(HaveOccurred())
			Expect(comments).To(HaveLen(1))
		})

		It("handles errors", func() {
			// Setup mock response for comments with error
			commentsResp := &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"error": "API error"}`)),
			}
			transport.Responses = append(transport.Responses, commentsResp)
			transport.Errors = append(transport.Errors, nil)

			// Create a post through the public interface
			posts, _, err := client.GetPosts(context.Background(), "golang", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(posts).To(HaveLen(1))

			post := posts[0]
			_, err = post.GetComments(context.Background())

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("server error"))
		})
	})
})
