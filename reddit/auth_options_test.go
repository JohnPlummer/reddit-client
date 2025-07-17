package reddit_test

import (
	"context"
	"net/http"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auth Options", func() {
	var (
		auth      *reddit.Auth
		transport *reddit.TestTransport
	)

	BeforeEach(func() {
		transport = reddit.NewTestTransport()
	})

	Describe("WithAuthHTTPClient", func() {
		It("sets a custom HTTP client", func() {
			customClient := &http.Client{
				Timeout: 30 * time.Second,
			}

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())
			Expect(auth).NotTo(BeNil())

			// WithAuthHTTPClient sets the client but doesn't update the auth.timeout field
			// The auth.timeout field shows the default timeout (10s), not the client's timeout
			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 10s"))
		})

		It("overwrites existing client configuration", func() {
			// First create auth with default transport
			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTransport(transport))
			Expect(err).NotTo(HaveOccurred())

			// Then apply custom client - this should overwrite the previous transport
			customClient := &http.Client{
				Timeout: 45 * time.Second,
			}

			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTransport(transport),
				reddit.WithAuthHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())

			// WithAuthHTTPClient overwrites the client but doesn't update auth.timeout
			// The auth.timeout field still shows the default timeout
			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 10s"))
		})

		It("accepts nil client", func() {
			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthHTTPClient(nil))
			Expect(err).NotTo(HaveOccurred())
			Expect(auth).NotTo(BeNil())

			// Even with nil client, NewAuth should create a default client
			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 10s")) // Default timeout
		})

		It("uses the custom HTTP client for requests", func() {
			// Create a custom transport to verify it's being used
			customTransport := reddit.NewTestTransport()
			customClient := &http.Client{
				Transport: customTransport,
				Timeout:   5 * time.Second,
			}

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())

			// Make an authentication request to verify the custom client is used
			// TestTransport automatically handles auth requests and returns valid responses
			ctx := context.Background()
			err = auth.Authenticate(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify that we got a token, which proves the custom client was used
			Expect(auth.Token).To(Equal("test_token"))
		})

		It("preserves client configuration when setting custom client", func() {
			customClient := &http.Client{
				Timeout: 25 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns: 100,
				},
			}

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())

			// WithAuthHTTPClient doesn't update the auth.timeout field
			// It shows the default timeout, not the client's timeout
			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 10s"))
		})
	})

	Describe("WithAuthTimeout", func() {
		It("sets timeout duration", func() {
			timeout := 15 * time.Second

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTimeout(timeout))
			Expect(err).NotTo(HaveOccurred())
			Expect(auth).NotTo(BeNil())

			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 15s"))
		})

		It("updates existing client timeout when client already exists", func() {
			// First create auth with custom transport (which creates a client)
			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTransport(transport),
				reddit.WithAuthTimeout(20*time.Second))
			Expect(err).NotTo(HaveOccurred())

			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 20s"))
		})

		It("handles zero timeout", func() {
			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTimeout(0))
			Expect(err).NotTo(HaveOccurred())
			Expect(auth).NotTo(BeNil())

			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 0s"))
		})

		It("handles negative timeout", func() {
			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTimeout(-5*time.Second))
			Expect(err).NotTo(HaveOccurred())
			Expect(auth).NotTo(BeNil())

			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: -5s"))
		})

		It("handles very large timeout", func() {
			largeTimeout := 24 * time.Hour

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTimeout(largeTimeout))
			Expect(err).NotTo(HaveOccurred())
			Expect(auth).NotTo(BeNil())

			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 24h0m0s"))
		})

		It("handles fractional timeout", func() {
			timeout := 1500 * time.Millisecond

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTimeout(timeout))
			Expect(err).NotTo(HaveOccurred())
			Expect(auth).NotTo(BeNil())

			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 1.5s"))
		})

		It("applies timeout to default client when no custom client is set", func() {
			timeout := 12 * time.Second

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTimeout(timeout))
			Expect(err).NotTo(HaveOccurred())

			// Verify timeout is applied to the default client created by NewAuth
			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 12s"))
		})

		It("overwrites previous timeout settings", func() {
			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTimeout(5*time.Second),
				reddit.WithAuthTimeout(10*time.Second))
			Expect(err).NotTo(HaveOccurred())

			// Should use the last timeout setting
			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 10s"))
		})
	})

	Describe("Combined Options", func() {
		It("applies timeout after setting custom client", func() {
			customClient := &http.Client{
				Timeout: 30 * time.Second,
			}

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthHTTPClient(customClient),
				reddit.WithAuthTimeout(15*time.Second))
			Expect(err).NotTo(HaveOccurred())

			// The timeout option should update the custom client's timeout
			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 15s"))
		})

		It("preserves timeout when setting client after timeout", func() {
			customClient := &http.Client{
				Timeout: 30 * time.Second,
			}

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthTimeout(15*time.Second),
				reddit.WithAuthHTTPClient(customClient))
			Expect(err).NotTo(HaveOccurred())

			// WithAuthHTTPClient doesn't update auth.timeout, so it keeps the previously set timeout
			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("Timeout: 15s"))
		})

		It("works with all auth options together", func() {
			customClient := &http.Client{
				Timeout: 25 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns: 50,
				},
			}

			var err error
			auth, err = reddit.NewAuth("test_id", "test_secret",
				reddit.WithAuthUserAgent("test-agent"),
				reddit.WithAuthTimeout(20*time.Second),
				reddit.WithAuthHTTPClient(customClient),
				reddit.WithAuthTransport(transport))
			Expect(err).NotTo(HaveOccurred())

			authStr := auth.String()
			Expect(authStr).To(ContainSubstring("UserAgent: \"test-agent\""))
			// WithAuthHTTPClient doesn't update auth.timeout, so it keeps the timeout set by WithAuthTimeout
			// WithAuthTransport comes after and modifies the client but preserves auth.timeout
			Expect(authStr).To(ContainSubstring("Timeout: 20s"))
		})
	})
})
