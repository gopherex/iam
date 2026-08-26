package postgres

// nowIn is the current time as the request sees it: real time, plus a test
// environment's clock offset when one is set. Use it wherever "has this
// expired?" is being decided, so a test can move time instead of waiting.
//
// Writes keep using nowUTC(): a timestamp records when something actually
// happened. Moving the clock forward makes what was issued earlier expire, which
// is what a test is asking for.

import (
	"context"
	"time"

	"github.com/gopherex/iam/internal/domain"
)

func nowIn(ctx context.Context) time.Time {
	return nowUTC().Add(domain.ClockOffset(ctx))
}
