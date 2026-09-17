package postgres

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
)

func (s *pgSecurity) reviewSessions(ctx context.Context, f *securityFlow) ([]domain.SecuritySession, error) {
	rows, err := s.db.TxDB.Query(
		ctx,
		`SELECT id,COALESCE(data->>'device_name','') FROM iam_sessions WHERE project_id=$1
AND environment=$2 AND user_id=$3 ORDER BY created_at DESC LIMIT 1000`,
		f.ProjectID,
		f.Environment,
		f.AccountID)
	if err != nil {
		return nil, securityStoreError(err)
	}
	defer rows.Close()

	sessions := []domain.SecuritySession{}

	for rows.Next() {
		var session domain.SecuritySession
		if err := rows.Scan(&session.ID, &session.Name); err != nil {
			return nil, securityStoreError(err)
		}

		sessions = append(sessions, session)
	}

	return sessions, securityStoreError(rows.Err())
}

//nolint:funlen // Keep the ordered security transition together for audit.
func (s *pgSecurity) selectFactor(ctx context.Context, f *securityFlow, id string) error {
	if f.State.ResendAt != nil && nowIn(ctx).Before(*f.State.ResendAt) {
		return domain.ErrFlowResendTooSoon
	}

	allowed, err := s.throttle(
		ctx,
		securityFlowScope(f),
		f.AccountID,
		"factor_delivery",
		securityDeliveryBurst,
		securityProofRateWindow)
	if err != nil {
		return securityStoreError(err)
	}

	if !allowed {
		f.State.ErrorCode = securityRateLimited
		return nil
	}

	f.FactorID = id
	f.CodeHash = ""
	f.WebAuthnSession = nil
	f.State.Challenge = nil

	if id == securityPasskey {
		return s.beginProofPasskey(ctx, f)
	}

	var raw []byte

	err = s.db.TxDB.QueryRow(
		ctx,
		`SELECT data FROM iam_factors WHERE id=$1 AND project_id=$2 AND environment=$3 AND
user_id=$4 AND status='active' AND created_at<=$5`,
		id,
		f.ProjectID,
		f.Environment,
		f.AccountID,
		f.Cutoff).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrMFAInvalid
	}

	if err != nil {
		return securityStoreError(err)
	}

	var factor domain.Factor
	if err := json.Unmarshal(raw, &factor); err != nil { //nolint:musttag // Preserve stored field names.
		return securityStoreError(err)
	}

	if factor.Type == "totp" {
		return nil
	}

	if factor.Type != securityEmail && factor.Type != securitySMS {
		return domain.ErrMFAInvalid
	}

	code, err := coreAuthRandomCode()
	if err != nil {
		return securityStoreError(err)
	}

	f.CodeHash = accountHashToken(code)
	f.CodeExpires = nowIn(ctx).Add(securityCodeTTL)
	f.FactorRecipient = factor.Hint
	resend := nowIn(ctx).Add(time.Minute)
	f.State.ResendAt = &resend
	f.State.ContactMasked = securityMask(factor.Hint)

	return s.queueDelivery(
		ctx,
		securityFlowScope(f),
		securityDeliveryPayload{To: factor.Hint, Template: securityCodeTemplate, Code: code},
		f.IncidentID,
		f.CaseID,
		"factor:"+f.ID+":"+newUUID())
}

func (s *pgSecurity) proofWebAuthnUser(ctx context.Context, f *securityFlow) (*webauthnUser, error) {
	user, err := NewPgWebAuthnAccounts(s.db, s.emitter, nil).loadWebauthnUser(ctx, f.AccountID)
	if err != nil {
		return nil, securityStoreError(err)
	}

	eligible := user.creds[:0]
	for i := range user.creds {
		cred := &user.creds[i]

		var exists bool
		if err := s.db.TxDB.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM iam_webauthn_credentials WHERE credential_id=$1 AND project_id=$2
AND environment=$3 AND user_id=$4 AND created_at<=$5)`,
			base64.RawURLEncoding.EncodeToString(cred.ID),
			f.ProjectID,
			f.Environment,
			f.AccountID,
			f.Cutoff).Scan(&exists); err != nil {
			return nil, securityStoreError(err)
		}

		if exists {
			eligible = append(eligible, *cred)
		}
	}

	user.creds = eligible

	return user, nil
}

func (s *pgSecurity) verifySelectedFactor(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) (bool, error) {
	if in.FactorID != f.FactorID {
		return false, nil
	}

	if f.FactorID == securityPasskey {
		return s.verifyProofPasskey(ctx, f, in)
	}

	if f.CodeHash == "" || !nowIn(ctx).Before(f.CodeExpires) {
		return false, nil
	}

	valid := subtle.ConstantTimeCompare([]byte(accountHashToken(in.Code)), []byte(f.CodeHash)) == 1
	if valid {
		f.CodeHash = ""
	}

	return valid, nil
}

// Keep established factors on self-service recovery. A support grant replaces
// inaccessible factors; credentials created during the incident are removed.
func (s *pgSecurity) invalidateRecoveryCredentials(ctx context.Context, f *securityFlow) error {
	for _, table := range []string{
		"iam_factors",
		"iam_webauthn_credentials",
		"iam_identities",
		"iam_recovery_codes",
	} {
		if _, err := s.db.TxDB.Exec(
			ctx,
			`DELETE FROM `+table+` WHERE project_id=$1 AND environment=$2 AND user_id=$3 AND ($4 OR created_at>$5)`,
			f.ProjectID,
			f.Environment,
			f.AccountID,
			f.SupportGrant,
			f.Cutoff); err != nil {
			return securityStoreError(err)
		}
	}

	if _, err := s.db.TxDB.Exec(
		ctx,
		`UPDATE iam_flows SET status='aborted' WHERE project_id=$1 AND environment=$2 AND user_id=$3
AND kind NOT IN ('security_review','security_recovery') AND status='pending'`,
		f.ProjectID,
		f.Environment,
		f.AccountID); err != nil {
		return securityStoreError(err)
	}

	_, err := s.db.TxDB.Exec(
		ctx,
		`UPDATE iam_challenges SET consumed=true WHERE project_id=$1 AND environment=$2 AND
NOT consumed AND (subject=$3 OR subject IN (SELECT primary_email FROM iam_users WHERE
id=$3 UNION SELECT primary_phone FROM iam_users WHERE id=$3) OR data->>'AccountID'=$3
OR data->>'account_id'=$3)`,
		f.ProjectID,
		f.Environment,
		f.AccountID)

	return securityStoreError(err)
}

func (s *pgSecurity) beginProofPasskey(ctx context.Context, f *securityFlow) error {
	adapter := NewPgWebAuthnAccounts(s.db, s.emitter, nil)

	user, err := s.proofWebAuthnUser(ctx, f)
	if err != nil {
		return securityStoreError(err)
	}

	if len(user.creds) == 0 {
		return domain.ErrMFAInvalid
	}

	relyingParty, err := adapter.rpConfigFor(ctx, f.ProjectID)
	if err != nil {
		return securityStoreError(err)
	}

	assertion, session, err := relyingParty.BeginLogin(
		user,
		gowebauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return domain.ErrMFAInvalid
	}

	f.WebAuthnSession, err = json.Marshal(session)
	if err != nil {
		return securityStoreError(err)
	}

	f.State.Challenge, err = webauthnOptionsMap(assertion.Response)

	return securityStoreError(err)
}

func (s *pgSecurity) verifyProofPasskey(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) (bool, error) {
	if len(f.WebAuthnSession) == 0 {
		return false, nil
	}

	var session gowebauthn.SessionData
	if err := json.Unmarshal(f.WebAuthnSession, &session); err != nil {
		return false, securityStoreError(err)
	}

	reader, err := webauthnCredentialReader(in.Credential)
	if err != nil {
		return false, securityStoreError(err)
	}

	parsed, err := protocol.ParseCredentialRequestResponseBody(reader)
	if err != nil {
		return false, nil //nolint:nilerr // Bad proof consumes an attempt.
	}

	user, err := s.proofWebAuthnUser(ctx, f)
	if err != nil {
		return false, securityStoreError(err)
	}

	adapter := NewPgWebAuthnAccounts(s.db, s.emitter, nil)

	relyingParty, err := adapter.rpConfigFor(ctx, f.ProjectID)
	if err != nil {
		return false, securityStoreError(err)
	}

	validated, err := relyingParty.ValidateLogin(user, session, parsed)
	if err != nil {
		return false, nil //nolint:nilerr // Bad proof consumes an attempt.
	}

	if err := adapter.webauthnBumpSignCount(ctx, f.ProjectID, f.AccountID, validated); err != nil {
		return false, securityStoreError(err)
	}

	f.WebAuthnSession = nil
	f.State.Challenge = nil

	return true, nil
}
