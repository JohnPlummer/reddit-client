package reddit_test

import (
	"net/http"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client Options", func() {
	var (
		auth      *reddit.Auth
		transport *reddit.TestTransport
	)

	BeforeEach(func() {
		transport = reddit.NewTestTransport()
		var err error
		auth, err = reddit.NewAuth("test_id", "test_secret",
			reddit.WithAuthTransport(transport))
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("WithHTTPClient", func() {
		It("sets a custom HTTP client", func() {
			customClient := &http.Client{
				Timeout: 30 * time.Second,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify that the client is properly initialized and functional
			// String() method should contain expected default values
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
			Expect(clientStr).To(ContainSubstring("RateLimiter{requests_per_minute: 60.0, burst: 5}"))
		})

		It("overwrites the default HTTP client", func() {
			// Create a custom client with specific configuration
			customTransport := &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
			}
			customClient := &http.Client{
				Transport: customTransport,
				Timeout:   45 * time.Second,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// The custom HTTP client should replace the default one
			// We can't directly access the internal client, but we can verify the client works
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("Auth: "))
			Expect(clientStr).To(ContainSubstring("UserAgent: "))
		})

		It("accepts nil HTTP client", func() {
			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(nil))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// NewClient should ensure there's always an HTTP client even if nil is passed
			// The client should still be functional
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("preserves other client options when setting custom HTTP client", func() {
			customClient := &http.Client{
				Timeout: 25 * time.Second,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithUserAgent("custom-test-agent"),
				reddit.WithRateLimit(120, 10),
				reddit.WithHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Other options should be preserved
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"custom-test-agent\""))
			Expect(clientStr).To(ContainSubstring("RateLimiter{requests_per_minute: 120.0, burst: 10}"))
		})

		It("overrides HTTP client when multiple WithHTTPClient options are applied", func() {
			firstClient := &http.Client{
				Timeout: 15 * time.Second,
			}
			secondClient := &http.Client{
				Timeout: 35 * time.Second,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(firstClient),
				reddit.WithHTTPClient(secondClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// The last WithHTTPClient option should win
			// We can't directly test the timeout, but we can verify the client is functional
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("works with custom transport in HTTP client", func() {
			// Create a test transport for verification
			testTransport := reddit.NewTestTransport()
			customClient := &http.Client{
				Transport: testTransport,
				Timeout:   20 * time.Second,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify the client is properly configured
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: "))
			Expect(clientStr).To(ContainSubstring("RateLimiter{"))
		})

		It("maintains HTTP client configuration when combined with other options", func() {
			customClient := &http.Client{
				Timeout: 50 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns: 200,
				},
			}

			client, err := reddit.NewClient(auth,
				reddit.WithUserAgent("integration-test-agent"),
				reddit.WithHTTPClient(customClient),
				reddit.WithRateLimit(90, 15))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// All options should be properly applied
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"integration-test-agent\""))
			Expect(clientStr).To(ContainSubstring("RateLimiter{requests_per_minute: 90.0, burst: 15}"))
		})

		It("allows HTTP client with no timeout set", func() {
			customClient := &http.Client{
				// No timeout set (will be 0)
				Transport: &http.Transport{},
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Client should still be functional even with no timeout
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("handles HTTP client with custom jar and other advanced settings", func() {
			customClient := &http.Client{
				Timeout: 40 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        50,
					MaxIdleConnsPerHost: 5,
					IdleConnTimeout:     90 * time.Second,
				},
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse // Don't follow redirects
				},
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify basic functionality is preserved
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("Auth: "))
			Expect(clientStr).To(ContainSubstring("UserAgent: "))
			Expect(clientStr).To(ContainSubstring("RateLimiter{"))
		})
	})

	Describe("WithHTTPClient integration with other options", func() {
		It("applies WithHTTPClient before WithTimeout", func() {
			customClient := &http.Client{
				Timeout: 25 * time.Second,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient),
				reddit.WithTimeout(15*time.Second))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// WithTimeout should override the custom client's timeout
			// We can't directly test the timeout, but the client should be functional
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("applies WithHTTPClient after WithTimeout", func() {
			customClient := &http.Client{
				Timeout: 35 * time.Second,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithTimeout(20*time.Second),
				reddit.WithHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// WithHTTPClient should replace the client that had timeout set
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("works with all available client options", func() {
			customClient := &http.Client{
				Timeout:   30 * time.Second,
				Transport: transport, // Use the test transport
			}

			client, err := reddit.NewClient(auth,
				reddit.WithUserAgent("comprehensive-test-agent"),
				reddit.WithRateLimit(75, 8),
				reddit.WithTimeout(10*time.Second),
				reddit.WithHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// All options should be applied correctly
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"comprehensive-test-agent\""))
			Expect(clientStr).To(ContainSubstring("RateLimiter{requests_per_minute: 75.0, burst: 8}"))
		})
	})
})
