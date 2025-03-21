package reddit

import (
	"errors"
	"fmt"
	"net/http"
)

// Error types for the Reddit client
var (
	ErrMissingCredentials = fmt.Errorf("missing credentials")
	ErrInvalidCredentials = fmt.Errorf("invalid credentials")
	ErrRateLimited        = fmt.Errorf("rate limited")
	ErrNotFound           = fmt.Errorf("not found")
	ErrServerError        = fmt.Errorf("server error")
	ErrBadRequest         = fmt.Errorf("bad request")
)

// APIError represents an error returned by the Reddit API
type APIError struct {
	StatusCode int
	Message    string
	Response   []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("reddit API error: status=%d message=%s", e.StatusCode, e.Message)
}

// NewAPIError creates a new APIError from an HTTP response
func NewAPIError(resp *http.Response, body []byte) error {
	var baseErr error
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		baseErr = ErrInvalidCredentials
	case http.StatusTooManyRequests:
		baseErr = ErrRateLimited
	case http.StatusNotFound:
		baseErr = ErrNotFound
	case http.StatusBadRequest:
		baseErr = ErrBadRequest
	default:
		if resp.StatusCode >= 500 {
			baseErr = ErrServerError
		}
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    baseErr.Error(),
		Response:   body,
	}
}

// IsRateLimitError returns true if the error is a rate limit error
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	return err == ErrRateLimited || (errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests)
}

// IsNotFoundError returns true if the error is a not found error
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	return err == ErrNotFound || (errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound)
}

// IsServerError returns true if the error is a server error
func IsServerError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	return err == ErrServerError || (errors.As(err, &apiErr) && apiErr.StatusCode >= 500)
}
