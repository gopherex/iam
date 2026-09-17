package postgres

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/gopherex/iam/internal/domain"
)

func (s *pgSecurity) restoreActions(ctx context.Context, f *securityFlow) ([]string, error) {
	config, err := NewConfigReader(s.db, 0).AuthConfig(ctx, f.ProjectID)
	if err != nil {
		return nil, securityStoreError(err)
	}

	if len(config.Methods) == 0 || slices.Contains(config.Methods, securityEmail) {
		return []string{securitySetPassword, securityRequestSupport}, nil
	}

	actions := []string{securityRequestSupport}
	if slices.Contains(config.Methods, securityPasskey) {
		actions = append(actions, "begin_passkey")
		if len(f.WebAuthnSession) > 0 {
			actions = append(actions, "finish_passkey")
		}
	}

	if slices.Contains(config.Methods, "phone") {
		actions = append(actions, "set_phone")
		if f.RecoveryPhone != "" {
			actions = append(actions, "verify_phone")
		}
	}

	if slices.Contains(config.Methods, "magic_link") || slices.Contains(config.Methods, "oauth") {
		actions = append(actions, "complete_recovery")
	}

	return actions, nil
}

func (s *pgSecurity) restoreWithoutPassword(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	switch in.Action {
	case "begin_passkey":
		return s.beginRecoveryPasskey(ctx, f)
	case "finish_passkey":
		return s.finishRecoveryPasskey(ctx, f, in)
	case "set_phone":
		return s.sendRecoveryPhone(ctx, f, in.Contact)
	case "verify_phone":
		if f.State.AttemptsLeft <= 0 || !nowIn(ctx).Before(f.CodeExpires) {
			f.State.ErrorCode = "challenge_expired"
			return nil
		}

		if subtle.ConstantTimeCompare([]byte(accountHashToken(in.Code)), []byte(f.CodeHash)) != 1 {
			f.State.AttemptsLeft--
			f.State.ErrorCode = "invalid_code"

			return nil
		}

		if err := s.prepareRestoredCredentials(ctx, f); err != nil {
			return securityStoreError(err)
		}

		account, err := s.account(ctx, securityFlowScope(f))
		if err != nil {
			return securityStoreError(err)
		}

		account.PrimaryPhone = f.RecoveryPhone
		account.PhoneVerified = true

		//nolint:musttag // Preserve the existing persisted domain envelope.
		raw, err := json.Marshal(account)
		if err != nil {
			return securityStoreError(err)
		}

		if _, err := s.db.TxDB.Exec(
			ctx,
			`UPDATE iam_users SET primary_phone=$1,data=data || $2::jsonb WHERE id=$3 AND project_id=$4 AND
environment=$5`,
			f.RecoveryPhone,
			raw,
			f.AccountID,
			f.ProjectID,
			f.Environment); err != nil {
			return securityStoreError(err)
		}

		return s.complete(ctx, f, securityAccountSecured)
	case "complete_recovery":
		if err := s.prepareRestoredCredentials(ctx, f); err != nil {
			return securityStoreError(err)
		}

		return s.complete(ctx, f, securityAccountSecured)
	default:
		return domain.ErrBadRequest
	}
}

func (s *pgSecurity) prepareRestoredCredentials(ctx context.Context, f *securityFlow) error {
	if err := s.invalidateRecoveryCredentials(ctx, f); err != nil {
		return securityStoreError(err)
	}

	if _, err := s.db.TxDB.Exec(
		ctx,
		`DELETE FROM iam_credentials WHERE project_id=$1 AND environment=$2 AND user_id=$3`,
		f.ProjectID,
		f.Environment,
		f.AccountID); err != nil {
		return securityStoreError(err)
	}

	return s.restoreAccountContacts(ctx, f)
}

func (s *pgSecurity) sendRecoveryPhone(ctx context.Context, f *securityFlow, phone string) error {
	if err := domain.ValidatePhone(phone); err != nil {
		return securityStoreError(err)
	}

	if f.State.ResendAt != nil && nowIn(ctx).Before(*f.State.ResendAt) {
		return domain.ErrFlowResendTooSoon
	}

	allowed, err := s.throttle(
		ctx,
		securityFlowScope(f),
		f.AccountID,
		"recovery_phone",
		securityDeliveryBurst,
		time.Hour)
	if err != nil {
		return securityStoreError(err)
	}

	if !allowed {
		return domain.ErrRateLimited
	}

	code, err := coreAuthRandomCode()
	if err != nil {
		return securityStoreError(err)
	}

	f.RecoveryPhone = phone
	f.CodeHash = accountHashToken(code)
	f.CodeExpires = nowIn(ctx).Add(securityCodeTTL)
	f.State.AttemptsLeft = securityCodeAttempts
	resend := nowIn(ctx).Add(time.Minute)
	f.State.ResendAt = &resend
	f.State.ContactMasked = securityMask(phone)

	return s.queueDelivery(
		ctx,
		securityFlowScope(f),
		securityDeliveryPayload{To: phone, Template: securityCodeTemplate, Code: code},
		f.IncidentID,
		f.CaseID,
		"recovery-phone:"+f.ID+":"+newUUID())
}

func (s *pgSecurity) beginRecoveryPasskey(ctx context.Context, f *securityFlow) error {
	adapter := NewPgWebAuthnAccounts(s.db, s.emitter, nil)

	relyingParty, err := adapter.rpConfigFor(ctx, f.ProjectID)
	if err != nil {
		return securityStoreError(err)
	}

	user, err := adapter.loadWebauthnUser(ctx, f.AccountID)
	if err != nil {
		return securityStoreError(err)
	}

	creation, session, err := relyingParty.BeginRegistration(
		user,
		gowebauthn.WithAuthenticatorSelection(
			protocol.AuthenticatorSelection{UserVerification: protocol.VerificationRequired}))
	if err != nil {
		return domain.ErrMFAInvalid
	}

	f.WebAuthnSession, err = json.Marshal(session)
	if err != nil {
		return securityStoreError(err)
	}

	f.State.Challenge, err = webauthnOptionsMap(creation.Response)

	return securityStoreError(err)
}

func (s *pgSecurity) finishRecoveryPasskey(
	ctx context.Context,
	f *securityFlow,
	in domain.SecurityFlowInput,
) error {
	var session gowebauthn.SessionData
	if err := json.Unmarshal(f.WebAuthnSession, &session); err != nil {
		return domain.ErrMFAInvalid
	}

	adapter := NewPgWebAuthnAccounts(s.db, s.emitter, nil)

	relyingParty, err := adapter.rpConfigFor(ctx, f.ProjectID)
	if err != nil {
		return securityStoreError(err)
	}

	user, err := adapter.loadWebauthnUser(ctx, f.AccountID)
	if err != nil {
		return securityStoreError(err)
	}

	reader, err := webauthnCredentialReader(in.Credential)
	if err != nil {
		return securityStoreError(err)
	}

	parsed, err := protocol.ParseCredentialCreationResponseBody(reader)
	if err != nil {
		return domain.ErrMFAInvalid
	}

	credential, err := relyingParty.CreateCredential(user, session, parsed)
	if err != nil {
		return domain.ErrMFAInvalid
	}

	if err := s.prepareRestoredCredentials(ctx, f); err != nil {
		return securityStoreError(err)
	}

	if _, err := adapter.webauthnInsertCredentialRow(
		ctx,
		f.ProjectID,
		f.Environment,
		f.AccountID,
		credential,
		"Recovered passkey"); err != nil {
		return securityStoreError(err)
	}

	f.WebAuthnSession = nil
	f.State.Challenge = nil

	return s.complete(ctx, f, securityAccountSecured)
}

// restoreAccountContacts restores the verified snapshot captured before the incident.
// A support decision instead authorizes its separately verified replacement email.
func (s *pgSecurity) restoreAccountContacts(ctx context.Context, f *securityFlow) error {
	if f.SupportGrant {
		return s.applySupportContact(ctx, f)
	}

	account, err := s.account(ctx, securityFlowScope(f))
	if err != nil {
		return err
	}

	account.PrimaryEmail = f.Contacts.Email
	account.EmailVerified = f.Contacts.Email != ""
	account.PrimaryPhone = f.Contacts.Phone
	account.PhoneVerified = f.Contacts.Phone != ""
	account.UpdatedAt = nowIn(ctx)
	//nolint:musttag // Preserve the existing persisted account envelope.
	raw, err := json.Marshal(account)
	if err != nil {
		return securityStoreError(err)
	}

	_, err = s.db.TxDB.Exec(
		ctx,
		`UPDATE iam_users
 SET primary_email=NULLIF($1,''),primary_phone=NULLIF($2,''),data=data || $3::jsonb,updated_at=$4
 WHERE id=$5 AND project_id=$6 AND environment=$7`,
		account.PrimaryEmail,
		account.PrimaryPhone,
		raw,
		account.UpdatedAt,
		f.AccountID,
		f.ProjectID,
		f.Environment)

	return translatePgErr("account", err)
}

func securityVerifiedContacts(account *domain.Account) securityIncidentPrivate {
	contacts := securityIncidentPrivate{}
	if account.EmailVerified {
		contacts.Email = account.PrimaryEmail
	}

	if account.PhoneVerified {
		contacts.Phone = account.PrimaryPhone
	}

	return contacts
}
