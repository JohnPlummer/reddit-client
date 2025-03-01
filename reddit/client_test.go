package reddit_test

import (
	"net/http"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		transport *mockTransport
		auth      *reddit.Auth
	)

	BeforeEach(func() {
		transport = &mockTransport{
			responses: make(map[string]*http.Response),
		}

		var err error
		auth, err = reddit.NewAuth("test_id", "test_secret",
			reddit.WithAuthTransport(transport))
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("NewClient", func() {
		It("creates a client with default options", func() {
			client, err := reddit.NewClient(
				reddit.WithAuth(auth),
				reddit.WithTransport(transport),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("creates a client with multiple custom options", func() {
			client, err := reddit.NewClient(
				reddit.WithAuth(auth),
				reddit.WithTransport(transport),
				reddit.WithUserAgent("custom-agent"),
				reddit.WithRateLimit(30, 3),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("creates a client with custom user agent", func() {
			client, err := reddit.NewClient(
				reddit.WithAuth(auth),
				reddit.WithTransport(transport),
				reddit.WithUserAgent("test-bot/1.0"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("creates a client with custom rate limiting", func() {
			client, err := reddit.NewClient(
				reddit.WithAuth(auth),
				reddit.WithTransport(transport),
				reddit.WithRateLimit(45, 4),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("returns error with nil auth option", func() {
			client, err := reddit.NewClient()
			Expect(err).To(MatchError("creating default auth client: missing credentials"))
			Expect(client).To(BeNil())
		})
	})
})
