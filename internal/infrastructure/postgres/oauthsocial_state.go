package postgres

// OAuth social-login CSRF state store + redirect-target validation.
//
// StartLogin/StartLink persist the caller-supplied `state` (sha256-hashed) bound
// to the provider and a VALIDATED redirect target in a short-lived iam_challenges
// row. CompleteLoginRedirect/CompleteLink look the state up (constant-time by
// hash), reject unknown/expired/replayed values, and use the STORED redirect —
// closing both the OAuth state-CSRF and the open-redirect holes.

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

const oauthStateTTL = 10 * time.Minute

// oauthSafeRedirect returns candidate only if it is a same-origin relative path
// (starts with "/" but not "//"); anything else (absolute/scheme-relative URL)
// falls back to the provider's configured default. Prevents open redirects.
func oauthSafeRedirect(candidate, fallback string) string {
	if strings.HasPrefix(candidate, "/") && !strings.HasPrefix(candidate, "//") {
		return candidate
	}

	return fallback
}

func oauthHashState(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

type oauthStateData struct {
	Redirect  string `json:"redirect"`
	AccountID string `json:"account_id,omitempty"`
}

// storeState persists a one-time CSRF state bound to provider + a validated
// redirect (and account, for link flows).
func (a *pgOAuthSocial) storeState(ctx context.Context, projectID, provider, state, redirect, accountID string) error {
	if state == "" {
		return domain.ErrBadRequest.WithMessage("state is required")
	}

	data, err := json.Marshal(oauthStateData{Redirect: redirect, AccountID: accountID})
	if err != nil {
		return fmt.Errorf("marshal oauth state: %w", err)
	}

	rawData := json.RawMessage(data)
	id := newUUID()
	typ := "oauth_state"
	ch := null.From(oauthHashState(state))
	sub := null.From(provider)
	exp := nowUTC().Add(oauthStateTTL)

	return a.db.withTx(ctx, func(ctx context.Context) error {
		setter := &models.IamChallengeSetter{
			ID: &id, ProjectID: &projectID, Type: &typ,
			Subject: &sub, CodeHash: &ch, ExpiresAt: &exp, Data: &rawData,
		}

		_, err := models.IamChallenges.Insert(setter).One(ctx, a.db.Bobx())
		if err != nil {
			return fmt.Errorf("insert oauth state: %w", err)
		}

		return nil
	})
}

// consumeState verifies and single-use-consumes a CSRF state, returning the
// stored redirect + account it was bound to. Unknown/expired/replayed/mismatched
// state is rejected (no session is minted by the caller).
func (a *pgOAuthSocial) consumeState(ctx context.Context, projectID, provider, state string) (redirect, accountID string, err error) {
	if state == "" {
		return "", "", domain.ErrBadRequest.WithMessage("state is required")
	}

	err = a.db.withTx(ctx, func(ctx context.Context) error {
		row, qerr := models.IamChallenges.Query(
			sm.Where(models.IamChallenges.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamChallenges.Columns.Type.EQ(psql.Arg("oauth_state"))),
			sm.Where(models.IamChallenges.Columns.CodeHash.EQ(psql.Arg(oauthHashState(state)))),
		).One(ctx, a.db.Bobx())
		if qerr != nil {
			if errors.Is(translatePgErr("state", qerr), ErrNotFound) {
				return domain.ErrBadRequest.WithMessage("invalid state")
			}

			return fmt.Errorf("query oauth state: %w", qerr)
		}

		if row.Consumed {
			return domain.ErrBadRequest.WithMessage("state already used")
		}

		if nowIn(ctx).After(row.ExpiresAt) {
			return domain.ErrBadRequest.WithMessage("state expired")
		}

		if sub, _ := row.Subject.Get(); subtle.ConstantTimeCompare([]byte(sub), []byte(provider)) != 1 {
			return domain.ErrBadRequest.WithMessage("state provider mismatch")
		}

		var stateData oauthStateData
		if len(row.Data) > 0 {
			if err := json.Unmarshal(row.Data, &stateData); err != nil {
				return domain.ErrBadRequest.WithMessage("corrupted OAuth state data")
			}
		}

		redirect, accountID = stateData.Redirect, stateData.AccountID
		consumed := true

		return row.Update(ctx, a.db.Bobx(), &models.IamChallengeSetter{Consumed: &consumed})
	})

	return redirect, accountID, err
}
