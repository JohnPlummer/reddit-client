package reddit_test

import (
	"net/http"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	Describe("NewClient", func() {
		It("creates a client with default options", func() {
			client, err := reddit.NewClient(&reddit.Auth{}, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("creates a client with custom HTTP client", func() {
			customClient := &http.Client{Timeout: 30 * time.Second}
			client, err := reddit.NewClient(&reddit.Auth{}, customClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("applies custom options", func() {
			client, err := reddit.NewClient(&reddit.Auth{}, nil,
				reddit.WithUserAgent("custom-agent"),
				reddit.WithRateLimit(30, 3),
				reddit.WithTimeout(5*time.Second),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("returns error with nil auth", func() {
			client, err := reddit.NewClient(nil, nil)
			Expect(err).To(Equal(reddit.ErrMissingCredentials))
			Expect(client).To(BeNil())
		})
	})
})
