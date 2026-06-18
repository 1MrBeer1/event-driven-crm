package httpx

import (
	"context"
	"time"
)

func Retry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}
