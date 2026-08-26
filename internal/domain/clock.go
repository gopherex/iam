package domain

// The test clock.
//
// Time-dependent behavior — a code expiring, a session idling out, a token
// running past its lifetime — is half of what an auth server does, and none of
// it is testable if the only way to reach it is to wait. So a non-live
// environment can carry a clock offset: the server keeps writing real
// timestamps, and reads the current time as `now + offset`.
//
// The offset is carried on the request context rather than looked up wherever it
// is needed, so a request either has one or does not, and the hot path in a live
// environment never asks.

import (
	"context"
	"time"
)

type clockOffsetKey struct{}

// WithClockOffset returns a context whose current time runs `offset` ahead.
func WithClockOffset(ctx context.Context, offset time.Duration) context.Context {
	if offset == 0 {
		return ctx
	}

	return context.WithValue(ctx, clockOffsetKey{}, offset)
}

// ClockOffset is how far ahead this request's clock runs. Zero for every request
// that did not come from a test-mode environment, which is all of them in live.
func ClockOffset(ctx context.Context) time.Duration {
	offset, _ := ctx.Value(clockOffsetKey{}).(time.Duration)

	return offset
}

// Now is the current time as this request sees it.
func Now(ctx context.Context) time.Time {
	return time.Now().UTC().Add(ClockOffset(ctx))
}
