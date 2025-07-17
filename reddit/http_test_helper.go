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
		responses:     make(map[string]*http.Response),
		callHistory:   make([]string, 0),
		errorOnCall:   make(map[int]error),
		responseQueue: make(map[string][]*http.Response),
	}
}

// TestTransport implements http.RoundTripper for testing
type TestTransport struct {
	responses     map[string]*http.Response
	err           error
	callCount     int                         // Track number of calls
	callHistory   []string                    // Track which paths were called
	errorOnCall   map[int]error               // Map from call number to error
	responseQueue map[string][]*http.Response // Queue of responses for a path
}

// Ensure TestTransport implements both interfaces
var _ HTTPTransport = (*TestTransport)(nil)

// RoundTrip implements the http.RoundTripper interface
func (m *TestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.callCount++
	m.callHistory = append(m.callHistory, req.URL.Path+"?"+req.URL.RawQuery)

	// Check for call-specific errors
	if err, hasErr := m.errorOnCall[m.callCount]; hasErr {
		return nil, err
	}

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

	// Check response queue first (for sequential responses)
	pathKey := req.URL.Path
	if queue, hasQueue := m.responseQueue[pathKey]; hasQueue && len(queue) > 0 {
		resp := queue[0]
		m.responseQueue[pathKey] = queue[1:] // Remove first response from queue

		// Return a new response with a fresh body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		resp.Body.Close()
		return &http.Response{
			StatusCode: resp.StatusCode,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     resp.Header,
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
			Header:     resp.Header,
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

// SetErrorOnCall sets an error to be returned on a specific call number
func (m *TestTransport) SetErrorOnCall(callNumber int, err error) {
	if m.errorOnCall == nil {
		m.errorOnCall = make(map[int]error)
	}
	m.errorOnCall[callNumber] = err
}

// AddResponseToQueue adds a response to the queue for a specific path
func (m *TestTransport) AddResponseToQueue(path string, resp *http.Response) {
	if m.responseQueue == nil {
		m.responseQueue = make(map[string][]*http.Response)
	}
	m.responseQueue[path] = append(m.responseQueue[path], resp)
}

// GetCallCount returns the number of calls made
func (m *TestTransport) GetCallCount() int {
	return m.callCount
}

// GetCallHistory returns the history of calls made
func (m *TestTransport) GetCallHistory() []string {
	return m.callHistory
}

// Reset resets the transport state
func (m *TestTransport) Reset() {
	m.responses = make(map[string]*http.Response)
	m.err = nil
	m.callCount = 0
	m.callHistory = make([]string, 0)
	m.errorOnCall = make(map[int]error)
	m.responseQueue = make(map[string][]*http.Response)
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
