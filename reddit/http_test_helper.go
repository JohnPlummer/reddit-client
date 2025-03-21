package reddit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// TestResponse represents a pre-configured HTTP response
type TestResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// NewTestTransport creates a new transport for testing HTTP clients
func NewTestTransport() *TestTransport {
	return &TestTransport{
		responses: make(map[string]*http.Response),
	}
}

// TestTransport implements http.RoundTripper for testing
type TestTransport struct {
	responses map[string]*http.Response
	err       error
}

// Ensure TestTransport implements both interfaces
var _ HTTPTransport = (*TestTransport)(nil)

// RoundTrip implements the http.RoundTripper interface
func (m *TestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Special handling for auth endpoint
	if req.URL.Host == "www.reddit.com" && req.URL.Path == "/api/v1/access_token" {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(bytes.NewReader([]byte(`{
				"access_token": "test_token",
				"token_type": "bearer",
				"expires_in": 3600
			}`))),
		}, nil
	}

	// For API endpoints, try to match the path
	if resp, ok := m.responses[req.URL.Path]; ok {
		// Return a new response with a fresh body for each request
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		resp.Body.Close()
		return &http.Response{
			StatusCode: resp.StatusCode,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}

	// Default response for unmatched paths
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}, nil
}

// AddResponse adds a response for a specific path
func (m *TestTransport) AddResponse(path string, resp *http.Response) {
	m.responses[path] = resp
}

// SetError sets an error to be returned by the transport
func (m *TestTransport) SetError(err error) {
	m.err = err
}

// CreateJSONResponse creates an HTTP response with JSON body
func CreateJSONResponse(data any) *http.Response {
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}
