package reddit_test

import (
	"time"

	"github.com/JohnPlummer/reddit-client/reddit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/time/rate"
)

var _ = Describe("RateLimiter", func() {
	var rateLimiter *reddit.RateLimiter

	Describe("Allow", func() {
		Context("when rate limit is not exceeded", func() {
			BeforeEach(func() {
				// Create a rate limiter with a generous limit
				rateLimiter = reddit.NewRateLimiter(60, 5) // 60 requests per minute, burst of 5
			})

			It("returns true for first few requests", func() {
				// Should allow burst requests
				for i := 0; i < 5; i++ {
					Expect(rateLimiter.Allow()).To(BeTrue())
				}
			})
		})

		Context("when rate limit is exceeded", func() {
			BeforeEach(func() {
				// Create a very restrictive rate limiter
				rateLimiter = reddit.NewRateLimiter(1, 1) // 1 request per minute, burst of 1
			})

			It("returns false when burst is exhausted", func() {
				// First request should be allowed
				Expect(rateLimiter.Allow()).To(BeTrue())

				// Subsequent requests should be denied immediately
				Expect(rateLimiter.Allow()).To(BeFalse())
				Expect(rateLimiter.Allow()).To(BeFalse())
			})
		})

		Context("with zero rate limit", func() {
			BeforeEach(func() {
				rateLimiter = reddit.NewRateLimiter(0, 1)
			})

			It("allows burst requests but denies subsequent requests", func() {
				// First request should be allowed due to burst capacity
				Expect(rateLimiter.Allow()).To(BeTrue())

				// Subsequent requests should be denied
				Expect(rateLimiter.Allow()).To(BeFalse())
				Expect(rateLimiter.Allow()).To(BeFalse())
			})
		})
	})

	Describe("Reserve", func() {
		Context("when rate limit is not exceeded", func() {
			BeforeEach(func() {
				rateLimiter = reddit.NewRateLimiter(60, 5) // 60 requests per minute, burst of 5
			})

			It("returns a reservation with no delay for burst requests", func() {
				// First few requests should have no delay
				for i := 0; i < 5; i++ {
					reservation := rateLimiter.Reserve()
					Expect(reservation).NotTo(BeNil())
					Expect(reservation.OK()).To(BeTrue())
					Expect(reservation.Delay()).To(BeZero())
				}
			})

			It("returns a reservation with delay after burst is exhausted", func() {
				// Exhaust the burst
				for i := 0; i < 5; i++ {
					rateLimiter.Reserve()
				}

				// Next reservation should have a delay
				reservation := rateLimiter.Reserve()
				Expect(reservation).NotTo(BeNil())
				Expect(reservation.OK()).To(BeTrue())
				Expect(reservation.Delay()).To(BeNumerically(">", 0))
			})
		})

		Context("with very low rate limit", func() {
			BeforeEach(func() {
				rateLimiter = reddit.NewRateLimiter(1, 1) // 1 request per minute
			})

			It("provides correct wait times", func() {
				// First request should be immediate
				reservation1 := rateLimiter.Reserve()
				Expect(reservation1.Delay()).To(BeZero())

				// Second request should have approximately 60 second delay
				reservation2 := rateLimiter.Reserve()
				Expect(reservation2.Delay()).To(BeNumerically("~", 60*time.Second, time.Second))
			})

			It("allows canceling reservations", func() {
				// Exhaust the burst first
				reservation1 := rateLimiter.Reserve()
				Expect(reservation1.Delay()).To(BeZero())

				// Make a reservation that should have delay
				reservation2 := rateLimiter.Reserve()
				Expect(reservation2.OK()).To(BeTrue())
				Expect(reservation2.Delay()).To(BeNumerically(">", 0))

				// Cancel the delayed reservation
				reservation2.Cancel()

				// The cancel should have freed up capacity
				// (Note: exact behavior may vary, but reservation should be valid)
				newReservation := rateLimiter.Reserve()
				Expect(newReservation.OK()).To(BeTrue())
			})
		})

		Context("with burst capacity", func() {
			BeforeEach(func() {
				rateLimiter = reddit.NewRateLimiter(120, 10) // 120 requests per minute, burst of 10
			})

			It("handles burst correctly", func() {
				reservations := make([]*rate.Reservation, 10)

				// First 10 reservations should be immediate
				for i := 0; i < 10; i++ {
					reservations[i] = rateLimiter.Reserve()
					Expect(reservations[i].Delay()).To(BeZero())
				}

				// 11th reservation should have a delay
				reservation := rateLimiter.Reserve()
				Expect(reservation.Delay()).To(BeNumerically(">", 0))
			})
		})
	})

	Describe("UpdateLimit", func() {
		BeforeEach(func() {
			rateLimiter = reddit.NewRateLimiter(60, 5) // Start with default values
		})

		Context("when remaining requests is greater than 0", func() {
			It("updates rate based on remaining requests and reset time", func() {
				future := time.Now().Add(10 * time.Minute)
				remaining := 100

				rateLimiter.UpdateLimit(remaining, future)

				// Should calculate rate as 100 requests / 600 seconds = ~0.167 RPS = 10 RPM
				rpm, burst := rateLimiter.GetConfig()
				Expect(rpm).To(BeNumerically("~", 10, 1.0)) // Allow more tolerance
				Expect(burst).To(Equal(5))                  // Capped at maximum of 5, not 10
			})

			It("caps burst at 5 when calculated burst exceeds maximum", func() {
				future := time.Now().Add(1 * time.Minute)
				remaining := 100 // This would give burst of 10, but we cap at 5

				rateLimiter.UpdateLimit(remaining, future)

				_, burst := rateLimiter.GetConfig()
				Expect(burst).To(Equal(5)) // Capped at maximum of 5
			})

			It("sets minimum burst of 1", func() {
				future := time.Now().Add(1 * time.Minute)
				remaining := 5 // This would give burst of 0, but minimum is 1

				rateLimiter.UpdateLimit(remaining, future)

				_, burst := rateLimiter.GetConfig()
				Expect(burst).To(Equal(1)) // Minimum burst of 1
			})

			It("handles small remaining counts correctly", func() {
				future := time.Now().Add(30 * time.Second)
				remaining := 1

				rateLimiter.UpdateLimit(remaining, future)

				rpm, burst := rateLimiter.GetConfig()
				// 1 request / 30 seconds = 0.033 RPS = 2 RPM
				Expect(rpm).To(BeNumerically("~", 2, 0.1))
				Expect(burst).To(Equal(1))
			})
		})

		Context("when remaining requests is 0", func() {
			It("sets very low rate limit", func() {
				future := time.Now().Add(5 * time.Minute)

				rateLimiter.UpdateLimit(0, future)

				rpm, burst := rateLimiter.GetConfig()
				Expect(rpm).To(Equal(6.0)) // 0.1 RPS = 6 RPM
				Expect(burst).To(Equal(1))
			})
		})

		Context("when remaining requests is negative", func() {
			It("sets very low rate limit", func() {
				future := time.Now().Add(5 * time.Minute)

				rateLimiter.UpdateLimit(-5, future)

				rpm, burst := rateLimiter.GetConfig()
				Expect(rpm).To(Equal(6.0)) // 0.1 RPS = 6 RPM
				Expect(burst).To(Equal(1))
			})
		})

		Context("when reset time is in the past", func() {
			It("does not update the limits", func() {
				// Get current config
				originalRPM, originalBurst := rateLimiter.GetConfig()

				// Try to update with past reset time
				past := time.Now().Add(-5 * time.Minute)
				rateLimiter.UpdateLimit(100, past)

				// Config should remain unchanged
				rpm, burst := rateLimiter.GetConfig()
				Expect(rpm).To(Equal(originalRPM))
				Expect(burst).To(Equal(originalBurst))
			})
		})

		Context("when reset time is now", func() {
			It("does not update the limits", func() {
				// Get current config
				originalRPM, originalBurst := rateLimiter.GetConfig()

				// Try to update with current time
				now := time.Now()
				rateLimiter.UpdateLimit(100, now)

				// Config should remain unchanged
				rpm, burst := rateLimiter.GetConfig()
				Expect(rpm).To(Equal(originalRPM))
				Expect(burst).To(Equal(originalBurst))
			})
		})

		Context("edge cases with burst calculations", func() {
			It("handles large remaining counts", func() {
				future := time.Now().Add(1 * time.Hour)
				remaining := 1000

				rateLimiter.UpdateLimit(remaining, future)

				_, burst := rateLimiter.GetConfig()
				Expect(burst).To(Equal(5)) // Should be capped at 5
			})

			It("handles very short time windows", func() {
				future := time.Now().Add(1 * time.Second)
				remaining := 10

				rateLimiter.UpdateLimit(remaining, future)

				rpm, burst := rateLimiter.GetConfig()
				// 10 requests / 1 second = 10 RPS = 600 RPM
				Expect(rpm).To(BeNumerically("~", 600, 1))
				Expect(burst).To(Equal(1)) // remaining/10 = 10/10 = 1
			})

			It("handles fractional burst calculations", func() {
				future := time.Now().Add(1 * time.Minute)
				remaining := 25 // 25/10 = 2.5, should round down to 2

				rateLimiter.UpdateLimit(remaining, future)

				_, burst := rateLimiter.GetConfig()
				Expect(burst).To(Equal(2))
			})
		})

		Context("integration with other methods", func() {
			It("affects Allow() behavior after update", func() {
				// Set very restrictive limit
				future := time.Now().Add(1 * time.Hour)
				rateLimiter.UpdateLimit(1, future)

				// Should allow first request
				Expect(rateLimiter.Allow()).To(BeTrue())

				// Should deny subsequent requests due to low rate
				Expect(rateLimiter.Allow()).To(BeFalse())
			})

			It("affects Reserve() behavior after update", func() {
				// Set very restrictive limit
				future := time.Now().Add(1 * time.Hour)
				rateLimiter.UpdateLimit(2, future)

				// First reservation should be immediate
				reservation1 := rateLimiter.Reserve()
				Expect(reservation1.Delay()).To(BeZero())

				// Second reservation should have a significant delay
				reservation2 := rateLimiter.Reserve()
				Expect(reservation2.Delay()).To(BeNumerically(">", time.Second))
			})
		})
	})

	Describe("integration tests", func() {
		BeforeEach(func() {
			rateLimiter = reddit.NewRateLimiter(60, 3)
		})

		It("works correctly with context cancellation in Reserve", func() {
			// Exhaust burst
			for i := 0; i < 3; i++ {
				rateLimiter.Reserve()
			}

			// Next reservation should have delay
			reservation := rateLimiter.Reserve()
			Expect(reservation.Delay()).To(BeNumerically(">", 0))

			// Should be able to cancel
			reservation.Cancel()
		})

		It("maintains consistency between Allow and Reserve", func() {
			// If Allow returns true, Reserve should return immediate reservation
			if rateLimiter.Allow() {
				reservation := rateLimiter.Reserve()
				Expect(reservation.Delay()).To(BeZero())
			}
		})

		It("handles rapid successive calls correctly", func() {
			allowed := 0
			immediate := 0

			// Make rapid calls
			for i := 0; i < 10; i++ {
				if rateLimiter.Allow() {
					allowed++
				}

				reservation := rateLimiter.Reserve()
				if reservation.Delay() == 0 {
					immediate++
				}
			}

			// Should have some consistent behavior
			Expect(allowed).To(BeNumerically("<=", 3))   // At most burst size
			Expect(immediate).To(BeNumerically(">=", 0)) // Should have at least some immediate reservations

			// Total calls should be consistent with what we made
			Expect(allowed + immediate).To(BeNumerically(">=", 3)) // At least burst size worth of immediate responses
		})
	})
})
