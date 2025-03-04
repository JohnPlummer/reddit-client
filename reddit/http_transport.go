package reddit

import (
	"net/http"
)

// HTTPTransport defines an interface for the http.RoundTripper functionality
// This interface is used for easier testing of HTTP clients
//
//go:generate mockgen -destination=mocks/http_transport_mock.go -package=mocks github.com/JohnPlummer/reddit-client/reddit HTTPTransport
type HTTPTransport interface {
	http.RoundTripper
}