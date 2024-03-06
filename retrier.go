package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Retrier implements an exponentially backing off retry instance.
// Use New instead of creating this object directly.
type Retrier struct {
	// Attempts is the number of remaining attempts.
	// Unlimited when 0.
	Attempts int

	// Delay is the current delay between attempts.
	Delay time.Duration

	// Floor and Ceil are the minimum and maximum delays.
	Floor, Ceil time.Duration

	// Rate is the rate at which the delay grows.
	// E.g. 2 means the delay doubles each time.
	Rate float64

	// Jitter determines the level of indeterminism in the delay.
	//
	// It is the standard deviation of the normal distribution of a random variable
	// multiplied by the delay. E.g. 0.1 means the delay is normally distributed
	// with a standard deviation of 10% of the delay. Floor and Ceil are still
	// respected, making outlandish values impossible.
	//
	// Jitter can help avoid thundering herds.
	Jitter float64
}

// New creates a retrier that exponentially backs off from floor to ceil pauses.
func New(floor, ceil time.Duration) *Retrier {
	return &Retrier{
		Attempts: 0,
		Delay:    0,
		Floor:    floor,
		Ceil:     ceil,
		// Phi scales more calmly than 2, but still has nice pleasing
		// properties.
		Rate: math.Phi,
	}
}

func applyJitter(d time.Duration, jitter float64) time.Duration {
	if jitter == 0 {
		return d
	}

	d = time.Duration(rand.NormFloat64()*(jitter*float64(d)) + float64(d))

	return d
}

// Wait returns after min(Delay*Growth, Ceil) or ctx is cancelled.
// The first call to Wait will return immediately.
func (r *Retrier) Wait(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}

	if r.Delay < r.Ceil {
		r.Delay = time.Duration(float64(r.Delay) * r.Rate)
	}

	r.Delay = applyJitter(r.Delay, r.Jitter)

	if r.Delay > r.Ceil {
		r.Delay = r.Ceil
	}

	if r.Attempts > 0 {
		if r.Attempts == 1 {
			return false
		}
		r.Attempts--
	}

	select {
	case <-time.After(r.Delay):
		if r.Delay < r.Floor {
			r.Delay = r.Floor
		}
		return true
	case <-ctx.Done():
		return false
	}
}

// Reset resets the retrier to its initial state.
func (r *Retrier) Reset() {
	r.Delay = 0
}
