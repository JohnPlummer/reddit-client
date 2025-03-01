package reddit

import (
	"net/http"
	"time"
)

// AuthOption represents a function that configures an Auth instance
type AuthOption func(*Auth)

// WithAuthUserAgent sets the user agent for auth requests
func WithAuthUserAgent(userAgent string) AuthOption {
	return func(a *Auth) {
		a.userAgent = userAgent
	}
}

// WithAuthTimeout sets the timeout for auth requests
func WithAuthTimeout(timeout time.Duration) AuthOption {
	return func(a *Auth) {
		a.timeout = timeout
		if a.client != nil {
			a.client.Timeout = timeout
		}
	}
}

// WithAuthTransport sets the transport for auth requests
func WithAuthTransport(transport http.RoundTripper) AuthOption {
	return func(a *Auth) {
		if a.client == nil {
			a.client = &http.Client{}
		}
		a.client.Transport = transport
		a.client.Timeout = a.timeout
	}
}

// WithAuthHTTPClient sets a custom HTTP client for auth requests
func WithAuthHTTPClient(client *http.Client) AuthOption {
	return func(a *Auth) {
		a.client = client
	}
}
