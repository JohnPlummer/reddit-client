package reddit_test

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Errors", func() {
	Describe("APIError", func() {
		It("implements the error interface", func() {
			apiErr := &reddit.APIError{
				StatusCode: 400,
				Message:    "bad request",
				Response:   []byte("error response"),
			}

			expectedMessage := "reddit API error: status=400 message=bad request"
			Expect(apiErr.Error()).To(Equal(expectedMessage))
		})

		It("formats error message correctly with status code and message", func() {
			apiErr := &reddit.APIError{
				StatusCode: 500,
				Message:    "internal server error",
				Response:   []byte("detailed error"),
			}

			Expect(apiErr.Error()).To(ContainSubstring("status=500"))
			Expect(apiErr.Error()).To(ContainSubstring("message=internal server error"))
		})
	})

	Describe("NewAPIError", func() {
		var (
			responseBody []byte
		)

		BeforeEach(func() {
			responseBody = []byte(`{"error": "test error response"}`)
		})

		Context("with 401 Unauthorized", func() {
			It("creates APIError with invalid credentials message", func() {
				resp := &http.Response{StatusCode: http.StatusUnauthorized}
				err := reddit.NewAPIError(resp, responseBody)

				Expect(err).To(BeAssignableToTypeOf(&reddit.APIError{}))
				apiErr := err.(*reddit.APIError)
				Expect(apiErr.StatusCode).To(Equal(http.StatusUnauthorized))
				Expect(apiErr.Message).To(Equal("invalid credentials"))
				Expect(apiErr.Response).To(Equal(responseBody))
			})
		})

		Context("with 429 Too Many Requests", func() {
			It("creates APIError with rate limited message", func() {
				resp := &http.Response{StatusCode: http.StatusTooManyRequests}
				err := reddit.NewAPIError(resp, responseBody)

				Expect(err).To(BeAssignableToTypeOf(&reddit.APIError{}))
				apiErr := err.(*reddit.APIError)
				Expect(apiErr.StatusCode).To(Equal(http.StatusTooManyRequests))
				Expect(apiErr.Message).To(Equal("rate limited"))
				Expect(apiErr.Response).To(Equal(responseBody))
			})
		})

		Context("with 404 Not Found", func() {
			It("creates APIError with not found message", func() {
				resp := &http.Response{StatusCode: http.StatusNotFound}
				err := reddit.NewAPIError(resp, responseBody)

				Expect(err).To(BeAssignableToTypeOf(&reddit.APIError{}))
				apiErr := err.(*reddit.APIError)
				Expect(apiErr.StatusCode).To(Equal(http.StatusNotFound))
				Expect(apiErr.Message).To(Equal("not found"))
				Expect(apiErr.Response).To(Equal(responseBody))
			})
		})

		Context("with 400 Bad Request", func() {
			It("creates APIError with bad request message", func() {
				resp := &http.Response{StatusCode: http.StatusBadRequest}
				err := reddit.NewAPIError(resp, responseBody)

				Expect(err).To(BeAssignableToTypeOf(&reddit.APIError{}))
				apiErr := err.(*reddit.APIError)
				Expect(apiErr.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(apiErr.Message).To(Equal("bad request"))
				Expect(apiErr.Response).To(Equal(responseBody))
			})
		})

		Context("with 500 Internal Server Error", func() {
			It("creates APIError with server error message", func() {
				resp := &http.Response{StatusCode: http.StatusInternalServerError}
				err := reddit.NewAPIError(resp, responseBody)

				Expect(err).To(BeAssignableToTypeOf(&reddit.APIError{}))
				apiErr := err.(*reddit.APIError)
				Expect(apiErr.StatusCode).To(Equal(http.StatusInternalServerError))
				Expect(apiErr.Message).To(Equal("server error"))
				Expect(apiErr.Response).To(Equal(responseBody))
			})
		})

		Context("with 502 Bad Gateway", func() {
			It("creates APIError with server error message for 5xx status", func() {
				resp := &http.Response{StatusCode: http.StatusBadGateway}
				err := reddit.NewAPIError(resp, responseBody)

				Expect(err).To(BeAssignableToTypeOf(&reddit.APIError{}))
				apiErr := err.(*reddit.APIError)
				Expect(apiErr.StatusCode).To(Equal(http.StatusBadGateway))
				Expect(apiErr.Message).To(Equal("server error"))
				Expect(apiErr.Response).To(Equal(responseBody))
			})
		})

		Context("with 503 Service Unavailable", func() {
			It("creates APIError with server error message for 5xx status", func() {
				resp := &http.Response{StatusCode: http.StatusServiceUnavailable}
				err := reddit.NewAPIError(resp, responseBody)

				Expect(err).To(BeAssignableToTypeOf(&reddit.APIError{}))
				apiErr := err.(*reddit.APIError)
				Expect(apiErr.StatusCode).To(Equal(http.StatusServiceUnavailable))
				Expect(apiErr.Message).To(Equal("server error"))
				Expect(apiErr.Response).To(Equal(responseBody))
			})
		})

		Context("with unhandled status codes", func() {
			It("panics for 2xx status when trying to call Error() on nil baseErr", func() {
				resp := &http.Response{StatusCode: http.StatusOK}
				Expect(func() {
					reddit.NewAPIError(resp, responseBody)
				}).To(Panic())
			})

			It("panics for 3xx status when trying to call Error() on nil baseErr", func() {
				resp := &http.Response{StatusCode: http.StatusMovedPermanently}
				Expect(func() {
					reddit.NewAPIError(resp, responseBody)
				}).To(Panic())
			})

			It("panics for 4xx status (not handled) when trying to call Error() on nil baseErr", func() {
				resp := &http.Response{StatusCode: http.StatusForbidden}
				Expect(func() {
					reddit.NewAPIError(resp, responseBody)
				}).To(Panic())
			})
		})

		Context("with empty response body", func() {
			It("creates APIError with empty response", func() {
				resp := &http.Response{StatusCode: http.StatusUnauthorized}
				err := reddit.NewAPIError(resp, nil)

				Expect(err).To(BeAssignableToTypeOf(&reddit.APIError{}))
				apiErr := err.(*reddit.APIError)
				Expect(apiErr.StatusCode).To(Equal(http.StatusUnauthorized))
				Expect(apiErr.Message).To(Equal("invalid credentials"))
				Expect(apiErr.Response).To(BeNil())
			})

			It("creates APIError with empty byte slice", func() {
				resp := &http.Response{StatusCode: http.StatusTooManyRequests}
				err := reddit.NewAPIError(resp, []byte{})

				Expect(err).To(BeAssignableToTypeOf(&reddit.APIError{}))
				apiErr := err.(*reddit.APIError)
				Expect(apiErr.StatusCode).To(Equal(http.StatusTooManyRequests))
				Expect(apiErr.Message).To(Equal("rate limited"))
				Expect(apiErr.Response).To(Equal([]byte{}))
			})
		})
	})

	Describe("IsRateLimitError", func() {
		Context("with nil error", func() {
			It("returns false", func() {
				Expect(reddit.IsRateLimitError(nil)).To(BeFalse())
			})
		})

		Context("with ErrRateLimited", func() {
			It("returns true for direct error", func() {
				Expect(reddit.IsRateLimitError(reddit.ErrRateLimited)).To(BeTrue())
			})

			It("returns false for wrapped error (direct equality check only)", func() {
				wrappedErr := fmt.Errorf("wrapped: %w", reddit.ErrRateLimited)
				Expect(reddit.IsRateLimitError(wrappedErr)).To(BeFalse())
			})
		})

		Context("with APIError", func() {
			It("returns true for 429 status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusTooManyRequests,
					Message:    "rate limited",
					Response:   []byte("rate limit exceeded"),
				}
				Expect(reddit.IsRateLimitError(apiErr)).To(BeTrue())
			})

			It("returns true for wrapped APIError with 429 status", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusTooManyRequests,
					Message:    "rate limited",
					Response:   []byte("rate limit exceeded"),
				}
				wrappedErr := fmt.Errorf("API call failed: %w", apiErr)
				Expect(reddit.IsRateLimitError(wrappedErr)).To(BeTrue())
			})

			It("returns false for APIError with different status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusNotFound,
					Message:    "not found",
					Response:   []byte("resource not found"),
				}
				Expect(reddit.IsRateLimitError(apiErr)).To(BeFalse())
			})
		})

		Context("with other errors", func() {
			It("returns false for unrelated error", func() {
				err := errors.New("some random error")
				Expect(reddit.IsRateLimitError(err)).To(BeFalse())
			})

			It("returns false for other predefined errors", func() {
				Expect(reddit.IsRateLimitError(reddit.ErrNotFound)).To(BeFalse())
				Expect(reddit.IsRateLimitError(reddit.ErrServerError)).To(BeFalse())
				Expect(reddit.IsRateLimitError(reddit.ErrInvalidCredentials)).To(BeFalse())
			})
		})
	})

	Describe("IsNotFoundError", func() {
		Context("with nil error", func() {
			It("returns false", func() {
				Expect(reddit.IsNotFoundError(nil)).To(BeFalse())
			})
		})

		Context("with ErrNotFound", func() {
			It("returns true for direct error", func() {
				Expect(reddit.IsNotFoundError(reddit.ErrNotFound)).To(BeTrue())
			})

			It("returns false for wrapped error (direct equality check only)", func() {
				wrappedErr := fmt.Errorf("wrapped: %w", reddit.ErrNotFound)
				Expect(reddit.IsNotFoundError(wrappedErr)).To(BeFalse())
			})
		})

		Context("with APIError", func() {
			It("returns true for 404 status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusNotFound,
					Message:    "not found",
					Response:   []byte("resource not found"),
				}
				Expect(reddit.IsNotFoundError(apiErr)).To(BeTrue())
			})

			It("returns true for wrapped APIError with 404 status", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusNotFound,
					Message:    "not found",
					Response:   []byte("resource not found"),
				}
				wrappedErr := fmt.Errorf("API call failed: %w", apiErr)
				Expect(reddit.IsNotFoundError(wrappedErr)).To(BeTrue())
			})

			It("returns false for APIError with different status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusTooManyRequests,
					Message:    "rate limited",
					Response:   []byte("rate limit exceeded"),
				}
				Expect(reddit.IsNotFoundError(apiErr)).To(BeFalse())
			})
		})

		Context("with other errors", func() {
			It("returns false for unrelated error", func() {
				err := errors.New("some random error")
				Expect(reddit.IsNotFoundError(err)).To(BeFalse())
			})

			It("returns false for other predefined errors", func() {
				Expect(reddit.IsNotFoundError(reddit.ErrRateLimited)).To(BeFalse())
				Expect(reddit.IsNotFoundError(reddit.ErrServerError)).To(BeFalse())
				Expect(reddit.IsNotFoundError(reddit.ErrInvalidCredentials)).To(BeFalse())
			})
		})
	})

	Describe("IsServerError", func() {
		Context("with nil error", func() {
			It("returns false", func() {
				Expect(reddit.IsServerError(nil)).To(BeFalse())
			})
		})

		Context("with ErrServerError", func() {
			It("returns true for direct error", func() {
				Expect(reddit.IsServerError(reddit.ErrServerError)).To(BeTrue())
			})

			It("returns false for wrapped error (direct equality check only)", func() {
				wrappedErr := fmt.Errorf("wrapped: %w", reddit.ErrServerError)
				Expect(reddit.IsServerError(wrappedErr)).To(BeFalse())
			})
		})

		Context("with APIError", func() {
			It("returns true for 500 status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusInternalServerError,
					Message:    "server error",
					Response:   []byte("internal server error"),
				}
				Expect(reddit.IsServerError(apiErr)).To(BeTrue())
			})

			It("returns true for 502 status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusBadGateway,
					Message:    "server error",
					Response:   []byte("bad gateway"),
				}
				Expect(reddit.IsServerError(apiErr)).To(BeTrue())
			})

			It("returns true for 503 status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusServiceUnavailable,
					Message:    "server error",
					Response:   []byte("service unavailable"),
				}
				Expect(reddit.IsServerError(apiErr)).To(BeTrue())
			})

			It("returns true for wrapped APIError with 5xx status", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusInternalServerError,
					Message:    "server error",
					Response:   []byte("internal server error"),
				}
				wrappedErr := fmt.Errorf("API call failed: %w", apiErr)
				Expect(reddit.IsServerError(wrappedErr)).To(BeTrue())
			})

			It("returns false for APIError with 4xx status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusBadRequest,
					Message:    "bad request",
					Response:   []byte("invalid parameters"),
				}
				Expect(reddit.IsServerError(apiErr)).To(BeFalse())
			})

			It("returns false for APIError with 2xx status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusOK,
					Message:    "",
					Response:   []byte("success"),
				}
				Expect(reddit.IsServerError(apiErr)).To(BeFalse())
			})

			It("returns false for APIError with 3xx status code", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusMovedPermanently,
					Message:    "",
					Response:   []byte("moved"),
				}
				Expect(reddit.IsServerError(apiErr)).To(BeFalse())
			})
		})

		Context("with other errors", func() {
			It("returns false for unrelated error", func() {
				err := errors.New("some random error")
				Expect(reddit.IsServerError(err)).To(BeFalse())
			})

			It("returns false for other predefined errors", func() {
				Expect(reddit.IsServerError(reddit.ErrRateLimited)).To(BeFalse())
				Expect(reddit.IsServerError(reddit.ErrNotFound)).To(BeFalse())
				Expect(reddit.IsServerError(reddit.ErrInvalidCredentials)).To(BeFalse())
			})
		})
	})

	Describe("Error wrapping behavior", func() {
		Context("with multiply wrapped errors", func() {
			It("correctly identifies rate limit errors through multiple wraps", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusTooManyRequests,
					Message:    "rate limited",
					Response:   []byte("rate limit exceeded"),
				}
				wrappedOnce := fmt.Errorf("level 1: %w", apiErr)
				wrappedTwice := fmt.Errorf("level 2: %w", wrappedOnce)

				Expect(reddit.IsRateLimitError(wrappedTwice)).To(BeTrue())
			})

			It("correctly identifies not found errors through multiple wraps", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusNotFound,
					Message:    "not found",
					Response:   []byte("resource not found"),
				}
				wrappedOnce := fmt.Errorf("level 1: %w", apiErr)
				wrappedTwice := fmt.Errorf("level 2: %w", wrappedOnce)

				Expect(reddit.IsNotFoundError(wrappedTwice)).To(BeTrue())
			})

			It("correctly identifies server errors through multiple wraps", func() {
				apiErr := &reddit.APIError{
					StatusCode: http.StatusInternalServerError,
					Message:    "server error",
					Response:   []byte("internal server error"),
				}
				wrappedOnce := fmt.Errorf("level 1: %w", apiErr)
				wrappedTwice := fmt.Errorf("level 2: %w", wrappedOnce)

				Expect(reddit.IsServerError(wrappedTwice)).To(BeTrue())
			})
		})

		Context("with mixed error types", func() {
			It("does not confuse different error types when wrapped", func() {
				rateLimitErr := &reddit.APIError{
					StatusCode: http.StatusTooManyRequests,
					Message:    "rate limited",
					Response:   []byte("rate limit exceeded"),
				}
				wrappedRateLimit := fmt.Errorf("wrapped rate limit: %w", rateLimitErr)

				Expect(reddit.IsRateLimitError(wrappedRateLimit)).To(BeTrue())
				Expect(reddit.IsNotFoundError(wrappedRateLimit)).To(BeFalse())
				Expect(reddit.IsServerError(wrappedRateLimit)).To(BeFalse())
			})
		})
	})

	Describe("Error constants", func() {
		It("verifies all predefined error constants exist and have correct messages", func() {
			Expect(reddit.ErrMissingCredentials.Error()).To(Equal("missing credentials"))
			Expect(reddit.ErrInvalidCredentials.Error()).To(Equal("invalid credentials"))
			Expect(reddit.ErrRateLimited.Error()).To(Equal("rate limited"))
			Expect(reddit.ErrNotFound.Error()).To(Equal("not found"))
			Expect(reddit.ErrServerError.Error()).To(Equal("server error"))
			Expect(reddit.ErrBadRequest.Error()).To(Equal("bad request"))
		})
	})
})
