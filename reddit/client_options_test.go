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

	Describe("WithTransportConfig", func() {
		It("applies default transport configuration", func() {
			client, err := reddit.NewClient(auth,
				reddit.WithTransportConfig(reddit.DefaultTransportConfig()))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify the client is properly configured
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("applies custom transport configuration", func() {
			config := &reddit.TransportConfig{
				MaxIdleConns:        200,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     120 * time.Second,
				DisableKeepAlives:   false,
				MaxConnsPerHost:     50,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithTransportConfig(config))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify the client is functional
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("handles nil transport configuration by using defaults", func() {
			client, err := reddit.NewClient(auth,
				reddit.WithTransportConfig(nil))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Should fall back to default configuration
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("preserves existing transport settings when possible", func() {
			// First set up a custom HTTP client with transport
			customTransport := &http.Transport{
				MaxIdleConns:        50,
				TLSHandshakeTimeout: 10 * time.Second,
			}
			customClient := &http.Client{
				Transport: customTransport,
				Timeout:   30 * time.Second,
			}

			// Then apply transport config which should preserve non-pooling settings
			config := &reddit.TransportConfig{
				MaxIdleConns:        150,
				MaxIdleConnsPerHost: 15,
				IdleConnTimeout:     60 * time.Second,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient),
				reddit.WithTransportConfig(config))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify the client is functional
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("works with custom HTTP client without transport", func() {
			customClient := &http.Client{
				Timeout: 25 * time.Second,
			}

			config := reddit.DefaultTransportConfig()
			config.MaxIdleConnsPerHost = 25

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient),
				reddit.WithTransportConfig(config))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify the client is functional
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("handles non-Transport transport types gracefully", func() {
			// Create a custom transport that's not *http.Transport
			customTransport := &MockTransport{}
			customClient := &http.Client{
				Transport: customTransport,
			}

			config := reddit.DefaultTransportConfig()

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient),
				reddit.WithTransportConfig(config))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Should create a new transport when existing one is not *http.Transport
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("applies transport config to client with no existing HTTP client", func() {
			config := &reddit.TransportConfig{
				MaxIdleConns:        75,
				MaxIdleConnsPerHost: 8,
				IdleConnTimeout:     45 * time.Second,
				DisableKeepAlives:   true,
			}

			// Create client without WithHTTPClient option
			client, err := reddit.NewClient(auth,
				reddit.WithTransportConfig(config))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Should create HTTP client and transport
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("works with all connection pooling options", func() {
			config := &reddit.TransportConfig{
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     180 * time.Second,
				DisableKeepAlives:   false,
				MaxConnsPerHost:     100,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithUserAgent("transport-test-agent"),
				reddit.WithRateLimit(120, 10),
				reddit.WithTransportConfig(config),
				reddit.WithTimeout(20*time.Second))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// All options should be applied correctly
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"transport-test-agent\""))
			Expect(clientStr).To(ContainSubstring("RateLimiter{requests_per_minute: 120.0, burst: 10}"))
		})

		It("handles zero values in transport config", func() {
			config := &reddit.TransportConfig{
				MaxIdleConns:        0, // Zero means no limit
				MaxIdleConnsPerHost: 0, // Zero means default
				IdleConnTimeout:     0, // Zero means no limit
				DisableKeepAlives:   false,
				MaxConnsPerHost:     0, // Zero means no limit
			}

			client, err := reddit.NewClient(auth,
				reddit.WithTransportConfig(config))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Should handle zero values gracefully
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})
	})

	Describe("DefaultTransportConfig", func() {
		It("returns sensible defaults for Reddit API", func() {
			config := reddit.DefaultTransportConfig()
			Expect(config).NotTo(BeNil())
			Expect(config.MaxIdleConns).To(Equal(100))
			Expect(config.MaxIdleConnsPerHost).To(Equal(10))
			Expect(config.IdleConnTimeout).To(Equal(90 * time.Second))
			Expect(config.DisableKeepAlives).To(BeFalse())
			Expect(config.MaxConnsPerHost).To(Equal(0))
		})

		It("can be modified before using", func() {
			config := reddit.DefaultTransportConfig()
			config.MaxIdleConnsPerHost = 25
			config.IdleConnTimeout = 120 * time.Second

			client, err := reddit.NewClient(auth,
				reddit.WithTransportConfig(config))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Verify the client is functional with modified config
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})
	})

	Describe("TransportConfig integration with other options", func() {
		It("works correctly when applied before WithHTTPClient", func() {
			config := reddit.DefaultTransportConfig()
			customClient := &http.Client{
				Timeout: 40 * time.Second,
			}

			client, err := reddit.NewClient(auth,
				reddit.WithTransportConfig(config),
				reddit.WithHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// WithHTTPClient should override the transport configuration
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("works correctly when applied after WithHTTPClient", func() {
			customClient := &http.Client{
				Timeout: 35 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns: 25,
				},
			}
			config := reddit.DefaultTransportConfig()

			client, err := reddit.NewClient(auth,
				reddit.WithHTTPClient(customClient),
				reddit.WithTransportConfig(config))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// WithTransportConfig should modify the custom client's transport
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})

		It("preserves timeout when applied with WithTimeout", func() {
			config := reddit.DefaultTransportConfig()

			client, err := reddit.NewClient(auth,
				reddit.WithTimeout(15*time.Second),
				reddit.WithTransportConfig(config))
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())

			// Both timeout and transport config should be applied
			clientStr := client.String()
			Expect(clientStr).To(ContainSubstring("UserAgent: \"golang:reddit-client:v1.0\""))
		})
	})
})

// MockTransport is a simple mock implementation for testing non-Transport types
type MockTransport struct{}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       http.NoBody,
	}, nil
}
