package reddit_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CircuitBreaker", func() {
	var (
		circuitBreaker *reddit.CircuitBreaker
		config         *reddit.CircuitBreakerConfig
	)

	BeforeEach(func() {
		config = &reddit.CircuitBreakerConfig{
			FailureThreshold: 3,
			SuccessThreshold: 5, // Higher than the number of concurrent requests we'll test
			Timeout:          100 * time.Millisecond,
			MaxRequests:      2,
			ShouldTrip: func(err error) bool {
				return err != nil
			},
		}
		circuitBreaker = reddit.NewCircuitBreaker(config)
	})

	Describe("NewCircuitBreaker", func() {
		It("should create a circuit breaker with default config when nil is passed", func() {
			cb := reddit.NewCircuitBreaker(nil)
			Expect(cb).NotTo(BeNil())
			Expect(cb.State()).To(Equal(reddit.CircuitClosed))
		})

		It("should create a circuit breaker with provided config", func() {
			Expect(circuitBreaker).NotTo(BeNil())
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitClosed))
		})

		It("should set defaults for zero values in config", func() {
			invalidConfig := &reddit.CircuitBreakerConfig{
				FailureThreshold: 0,
				SuccessThreshold: 0,
				Timeout:          0,
				MaxRequests:      -1,
			}
			cb := reddit.NewCircuitBreaker(invalidConfig)
			Expect(cb).NotTo(BeNil())
		})
	})

	Describe("State transitions", func() {
		It("should start in closed state", func() {
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitClosed))
		})

		It("should transition to open state after failure threshold", func() {
			// Cause enough failures to open the circuit
			for i := 0; i < config.FailureThreshold; i++ {
				err := circuitBreaker.Execute(func() error {
					return errors.New("test error")
				})
				Expect(err).To(HaveOccurred())
			}

			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitOpen))
		})

		It("should transition to half-open state after timeout", func() {
			// Open the circuit
			for i := 0; i < config.FailureThreshold; i++ {
				circuitBreaker.Execute(func() error {
					return errors.New("test error")
				})
			}
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitOpen))

			// Wait for timeout
			time.Sleep(config.Timeout + 10*time.Millisecond)

			// Next request should transition to half-open
			err := circuitBreaker.Execute(func() error {
				return nil // Success
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitHalfOpen))
		})

		It("should transition from half-open to closed after enough successes", func() {
			// Open the circuit
			for i := 0; i < config.FailureThreshold; i++ {
				circuitBreaker.Execute(func() error {
					return errors.New("test error")
				})
			}

			// Wait for timeout
			time.Sleep(config.Timeout + 10*time.Millisecond)

			// Make enough successful requests to close the circuit
			for i := 0; i < config.SuccessThreshold; i++ {
				err := circuitBreaker.Execute(func() error {
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitClosed))
		})

		It("should transition from half-open to open on any failure", func() {
			// Open the circuit
			for i := 0; i < config.FailureThreshold; i++ {
				circuitBreaker.Execute(func() error {
					return errors.New("test error")
				})
			}

			// Wait for timeout
			time.Sleep(config.Timeout + 10*time.Millisecond)

			// Make one successful request (transitions to half-open)
			err := circuitBreaker.Execute(func() error {
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitHalfOpen))

			// Make a failing request (should transition back to open)
			err = circuitBreaker.Execute(func() error {
				return errors.New("test error")
			})
			Expect(err).To(HaveOccurred())
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitOpen))
		})
	})

	Describe("Execute", func() {
		It("should execute function successfully when circuit is closed", func() {
			executed := false
			err := circuitBreaker.Execute(func() error {
				executed = true
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(executed).To(BeTrue())
		})

		It("should return circuit breaker error when circuit is open", func() {
			// Open the circuit
			for i := 0; i < config.FailureThreshold; i++ {
				circuitBreaker.Execute(func() error {
					return errors.New("test error")
				})
			}

			executed := false
			err := circuitBreaker.Execute(func() error {
				executed = true
				return nil
			})

			Expect(err).To(HaveOccurred())
			Expect(executed).To(BeFalse())
			var cbErr *reddit.CircuitBreakerError
			Expect(errors.As(err, &cbErr)).To(BeTrue())
			Expect(cbErr.State).To(Equal(reddit.CircuitOpen))
		})

		It("should allow limited requests in half-open state", func() {
			// Open the circuit
			for i := 0; i < config.FailureThreshold; i++ {
				circuitBreaker.Execute(func() error {
					return errors.New("test error")
				})
			}

			// Verify circuit is open
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitOpen))

			// Wait for timeout to allow transition to half-open
			time.Sleep(config.Timeout + 10*time.Millisecond)

			// First request after timeout should transition to half-open and succeed
			err := circuitBreaker.Execute(func() error {
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitHalfOpen))

			// Subsequent successful requests should work but keep us in half-open
			err = circuitBreaker.Execute(func() error {
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			// Note: The actual number of concurrent requests that can be processed
			// depends on the timing of completion, but the circuit should remain
			// in half-open state until SuccessThreshold is reached
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitHalfOpen))
		})
	})

	Describe("ShouldTrip function", func() {
		It("should not trip circuit for errors that don't match ShouldTrip", func() {
			config.ShouldTrip = func(err error) bool {
				return err.Error() == "trip-worthy"
			}
			circuitBreaker = reddit.NewCircuitBreaker(config)

			// Make many requests with non-trip-worthy errors
			for i := 0; i < config.FailureThreshold*2; i++ {
				err := circuitBreaker.Execute(func() error {
					return errors.New("not-trip-worthy")
				})
				Expect(err).To(HaveOccurred())
			}

			// Circuit should still be closed
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitClosed))
		})

		It("should trip circuit for errors that match ShouldTrip", func() {
			config.ShouldTrip = func(err error) bool {
				return err.Error() == "trip-worthy"
			}
			circuitBreaker = reddit.NewCircuitBreaker(config)

			// Make enough requests with trip-worthy errors
			for i := 0; i < config.FailureThreshold; i++ {
				err := circuitBreaker.Execute(func() error {
					return errors.New("trip-worthy")
				})
				Expect(err).To(HaveOccurred())
			}

			// Circuit should be open
			Expect(circuitBreaker.State()).To(Equal(reddit.CircuitOpen))
		})
	})

	Describe("State callbacks", func() {
		It("should call OnStateChange callback when state changes", func() {
			var fromStates []reddit.CircuitState
			var toStates []reddit.CircuitState
			var mu sync.Mutex

			config.OnStateChange = func(from, to reddit.CircuitState) {
				mu.Lock()
				defer mu.Unlock()
				fromStates = append(fromStates, from)
				toStates = append(toStates, to)
			}
			circuitBreaker = reddit.NewCircuitBreaker(config)

			// Open the circuit
			for i := 0; i < config.FailureThreshold; i++ {
				circuitBreaker.Execute(func() error {
					return errors.New("test error")
				})
			}

			// Wait for callback to be called
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(fromStates)
			}).Should(Equal(1))

			mu.Lock()
			Expect(fromStates[0]).To(Equal(reddit.CircuitClosed))
			Expect(toStates[0]).To(Equal(reddit.CircuitOpen))
			mu.Unlock()
		})
	})

	Describe("Counts", func() {
		It("should track failure and success counts correctly", func() {
			failures, successes := circuitBreaker.Counts()
			Expect(failures).To(Equal(0))
			Expect(successes).To(Equal(0))

			// Make a few failures
			for i := 0; i < 2; i++ {
				circuitBreaker.Execute(func() error {
					return errors.New("test error")
				})
			}

			failures, successes = circuitBreaker.Counts()
			Expect(failures).To(Equal(2))
			Expect(successes).To(Equal(0))

			// Make a success (should reset failure count in closed state)
			err := circuitBreaker.Execute(func() error {
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			failures, successes = circuitBreaker.Counts()
			Expect(failures).To(Equal(0))
			Expect(successes).To(Equal(0))
		})
	})

	Describe("String representation", func() {
		It("should return a meaningful string representation", func() {
			str := circuitBreaker.String()
			Expect(str).To(ContainSubstring("CircuitBreaker"))
			Expect(str).To(ContainSubstring("state: closed"))
			Expect(str).To(ContainSubstring("failures: 0"))
			Expect(str).To(ContainSubstring("successes: 0"))
			Expect(str).To(ContainSubstring("threshold: 3"))
		})
	})

	Describe("CircuitState String method", func() {
		It("should return correct string representations", func() {
			Expect(reddit.CircuitClosed.String()).To(Equal("closed"))
			Expect(reddit.CircuitOpen.String()).To(Equal("open"))
			Expect(reddit.CircuitHalfOpen.String()).To(Equal("half-open"))
		})
	})

	Describe("CircuitBreakerError", func() {
		It("should implement error interface correctly", func() {
			err := &reddit.CircuitBreakerError{State: reddit.CircuitOpen}
			Expect(err.Error()).To(Equal("circuit breaker is open"))
		})
	})

	Describe("DefaultCircuitBreakerConfig", func() {
		It("should return sensible defaults", func() {
			config := reddit.DefaultCircuitBreakerConfig()
			Expect(config.FailureThreshold).To(Equal(5))
			Expect(config.SuccessThreshold).To(Equal(3))
			Expect(config.Timeout).To(Equal(30 * time.Second))
			Expect(config.MaxRequests).To(Equal(5))
			Expect(config.ShouldTrip).NotTo(BeNil())
			Expect(config.OnStateChange).NotTo(BeNil())
		})

		It("should have ShouldTrip function that handles Reddit errors correctly", func() {
			config := reddit.DefaultCircuitBreakerConfig()

			// Test with server error
			serverErr := &reddit.APIError{StatusCode: 500}
			Expect(config.ShouldTrip(serverErr)).To(BeTrue())

			// Test with timeout error
			timeoutErr := context.DeadlineExceeded
			Expect(config.ShouldTrip(timeoutErr)).To(BeTrue())

			// Test with client error (should not trip)
			clientErr := &reddit.APIError{StatusCode: 400}
			Expect(config.ShouldTrip(clientErr)).To(BeFalse())
		})
	})
})
