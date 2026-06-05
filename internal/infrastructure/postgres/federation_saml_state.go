package postgres

// SAML SP-initiation correlation + assertion replay protection.
//
// SamlLogin persists the unguessable RelayState it issues; SamlAcs consumes it,
// so an assertion that arrives without a matching outstanding request (an
// IdP-initiated / cross-site-forged POST) is rejected. Each consumed assertion
// ID is recorded so a captured assertion cannot be replayed within its validity
// window.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/aarondl/opt/null"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

const fedSamlRequestTTL = 10 * time.Minute

func fedHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// fedStoreSamlRequest records the issued RelayState for an SP-initiated flow.
func (a *pgFederationRuntime) fedStoreSamlRequest(ctx context.Context, projectID, connectionID, relayState, redirect string) error {
	if relayState == "" {
		return nil // nothing to correlate (no SP-initiated state)
	}
	data, _ := json.Marshal(map[string]string{"redirect": redirect, "connection_id": connectionID})
	rm := json.RawMessage(data)
	id := newUUID()
	typ := "saml_request"
	sub := null.From(connectionID)
	ch := null.From(fedHash(relayState))
	exp := nowUTC().Add(fedSamlRequestTTL)
	return a.db.withTx(ctx, func(ctx context.Context) error {
		setter := &models.IamChallengeSetter{
			ID: &id, ProjectID: &projectID, Type: &typ,
			Subject: &sub, CodeHash: &ch, ExpiresAt: &exp, Data: &rm,
		}
		_, err := models.IamChallenges.Insert(setter).One(ctx, a.db.Bobx())
		return err
	})
}

// fedConsumeSamlRequest verifies + single-use-consumes a RelayState for the
// connection. ok=false means no outstanding SP-initiated request matched (the
// caller decides whether to allow IdP-initiated for the connection).
func (a *pgFederationRuntime) fedConsumeSamlRequest(ctx context.Context, projectID, connectionID, relayState string) (ok bool, err error) {
	if relayState == "" {
		return false, nil
	}
	err = a.db.withTx(ctx, func(ctx context.Context) error {
		row, qerr := models.IamChallenges.Query(
			sm.Where(models.IamChallenges.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamChallenges.Columns.Type.EQ(psql.Arg("saml_request"))),
			sm.Where(models.IamChallenges.Columns.CodeHash.EQ(psql.Arg(fedHash(relayState)))),
		).One(ctx, a.db.Bobx())
		if qerr != nil {
			if errors.Is(translatePgErr("saml_request", qerr), ErrNotFound) {
				return nil // ok stays false
			}
			return qerr
		}
		if sub, _ := row.Subject.Get(); sub != connectionID {
			return nil
		}
		if row.Consumed || nowUTC().After(row.ExpiresAt) {
			return nil
		}
		consumed := true
		if uerr := row.Update(ctx, a.db.Bobx(), &models.IamChallengeSetter{Consumed: &consumed}); uerr != nil {
			return uerr
		}
		ok = true
		return nil
	})
	return ok, err
}

// fedAssertNotReplayed records a consumed assertion ID and rejects a duplicate
// (replay) within the assertion validity window.
func (a *pgFederationRuntime) fedAssertNotReplayed(ctx context.Context, projectID, connectionID, assertionID string, notOnOrAfter time.Time) error {
	if assertionID == "" {
		return domain.ErrSSOError
	}
	return a.db.withTx(ctx, func(ctx context.Context) error {
		_, qerr := models.IamChallenges.Query(
			sm.Where(models.IamChallenges.Columns.ProjectID.EQ(psql.Arg(projectID))),
			sm.Where(models.IamChallenges.Columns.Type.EQ(psql.Arg("saml_assertion"))),
			sm.Where(models.IamChallenges.Columns.CodeHash.EQ(psql.Arg(fedHash(assertionID)))),
		).One(ctx, a.db.Bobx())
		if qerr == nil {
			return domain.ErrSSOError.WithMessage("assertion replay")
		}
		if !errors.Is(translatePgErr("assertion", qerr), ErrNotFound) {
			return qerr
		}
		exp := notOnOrAfter
		if exp.IsZero() || exp.Before(nowUTC()) {
			exp = nowUTC().Add(fedSamlRequestTTL)
		}
		id := newUUID()
		typ := "saml_assertion"
		sub := null.From(connectionID)
		ch := null.From(fedHash(assertionID))
		raw := json.RawMessage(`{}`)
		setter := &models.IamChallengeSetter{
			ID: &id, ProjectID: &projectID, Type: &typ,
			Subject: &sub, CodeHash: &ch, ExpiresAt: &exp, Data: &raw,
		}
		_, ierr := models.IamChallenges.Insert(setter).One(ctx, a.db.Bobx())
		return ierr
	})
}
