package retry

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	t.Run("Attempts", func(t *testing.T) {
		t.Run("Respects count and sleeps between attempts", func(t *testing.T) {
			count := 0
			start := time.Now()

			const sleep = time.Millisecond * 10

			New(func() error {
				count++
				return errors.Errorf("asdfasdf")
			}, sleep).Attempts(5).Run()

			assert.Equal(t, 5, count)
			assert.WithinDuration(t, start.Add(sleep*5), time.Now(), sleep)
		})

		t.Run("returns as soon as error is nil", func(t *testing.T) {
			start := time.Now()

			var count int
			New(func() error {
				count++
				return nil
			}, time.Minute).Attempts(100).Run()

			assert.Equal(t, 1, count)
			assert.WithinDuration(t, time.Now(), start, time.Millisecond)
		})
	})
	t.Run("Backoff", func(t *testing.T) {
		t.Run("return when nil", func(t *testing.T) {
			var count int
			err := New(
				func() error {
					count++
					if count == 10 {
						return nil
					}
					return io.EOF
				},
				time.Millisecond,
			).Timeout(time.Minute).Backoff(time.Second).Run()

			assert.Equal(t, 10, count)
			assert.NoError(t, err)
		})
	})

	t.Run("Timeout", func(t *testing.T) {
		t.Run("Respects timeout and sleeps between attempts", func(t *testing.T) {
			count := 0
			start := time.Now()

			const sleep = time.Millisecond * 10

			New(func() error {
				count++
				return errors.Errorf("asdfasdf")
			}, sleep).Timeout(sleep * 5).Run()

			assert.Equal(t, 5, count)
			assert.WithinDuration(t, start.Add(sleep*5), time.Now(), sleep)
		})

		t.Run("returns as soon as error is nil", func(t *testing.T) {
			start := time.Now()

			New(func() error {
				return nil
			}, time.Minute).Timeout(time.Hour).Run()

			assert.WithinDuration(t, time.Now(), start, time.Millisecond*10)
		})
	})

	t.Run("Cond", func(t *testing.T) {
		count := 0

		err := New(func() error {
			if count == 5 {
				return io.ErrNoProgress
			}
			count++
			return io.EOF
		}, time.Millisecond).Cond(func(err error) bool {
			return err == io.EOF
		}).Run()

		assert.Equal(t, count, 5)
		assert.Equal(t, io.ErrNoProgress, err)
	})

	t.Run("Context", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())

		var count int

		err := New(func() error {
			if count == 3 {
				cancel()
			} else {
				count++
			}
			return io.EOF
		}, time.Millisecond).Context(ctx).Run()

		assert.Equal(t, 3, count)
		assert.Equal(t, io.EOF, err)
	})
}
