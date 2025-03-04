package reddit_test

import (
	"net/http"

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
			client, err := reddit.NewClient(
				reddit.WithAuth(auth),
				reddit.WithHTTPClient(mockClient),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(client.String()).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
			Expect(client.String()).To(ContainSubstring("RateLimiter{requests_per_minute: 60.0, burst: 5}"))
		})

		It("creates a client with multiple custom options", func() {
			client, err := reddit.NewClient(
				reddit.WithAuth(auth),
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
			client, err := reddit.NewClient(
				reddit.WithAuth(auth),
				reddit.WithHTTPClient(mockClient),
				reddit.WithUserAgent("test-bot/1.0"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(client.String()).To(ContainSubstring("UserAgent: \"test-bot/1.0\""))
		})

		It("creates a client with custom rate limiting", func() {
			client, err := reddit.NewClient(
				reddit.WithAuth(auth),
				reddit.WithHTTPClient(mockClient),
				reddit.WithRateLimit(45, 4),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify rate limiter configuration through String method
			Expect(client.String()).To(ContainSubstring("RateLimiter{requests_per_minute: 45.0, burst: 4}"))
		})

		It("creates a client with default rate limiting", func() {
			client, err := reddit.NewClient(
				reddit.WithAuth(auth),
				reddit.WithHTTPClient(mockClient),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify default rate limiter configuration through String method
			Expect(client.String()).To(ContainSubstring("RateLimiter{requests_per_minute: 60.0, burst: 5}"))
		})

		It("returns error with nil auth option", func() {
			client, err := reddit.NewClient()
			Expect(err).To(MatchError("creating default auth client: missing credentials"))
			Expect(client).To(BeNil())
		})
	})
})
