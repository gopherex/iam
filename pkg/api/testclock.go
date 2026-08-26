package api

// Test-clock middleware.
//
// A non-live environment can run its clock ahead, so a test can make something
// expire instead of waiting for it. The offset is resolved once per request and
// carried on the context; everything that decides "has this expired?" reads it
// from there.
//
// Live is never asked. The environment header is checked first, and a live (or
// absent) environment short-circuits before any lookup — the offset only exists
// where test mode is allowed to exist at all.

import (
	"context"
	"net/http"
	"time"

	"github.com/gopherex/iam/internal/domain"
)

// TestClock resolves an environment's clock offset. Implemented by the store.
type TestClock interface {
	ClockOffset(ctx context.Context, projectID, environment string) (time.Duration, error)
}

// liveEnvironment is the one environment test mode may never touch.
const liveEnvironment = "live"

// TestClockMiddleware puts a non-live environment's clock offset on the request
// context. It is a no-op without a resolver, so a deployment that does not want
// test mode simply does not wire one.
func TestClockMiddleware(clock TestClock, next http.Handler) http.Handler {
	if clock == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env := r.Header.Get("X-Environment")
		project := r.Header.Get("X-Client-Id")

		if env == "" || env == liveEnvironment || project == "" {
			next.ServeHTTP(w, r)

			return
		}

		offset, err := clock.ClockOffset(r.Context(), project, env)
		if err != nil || offset == 0 {
			// A clock that cannot be read is a clock that is not set. Failing the
			// request would make an unrelated misconfiguration look like an auth
			// outage in a test environment.
			next.ServeHTTP(w, r)

			return
		}

		next.ServeHTTP(w, r.WithContext(domain.WithClockOffset(r.Context(), offset)))
	})
}
