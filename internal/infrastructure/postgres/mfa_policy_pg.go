package postgres

// Runtime reader for the mfa_policy doc (iam_config key="mfa_policy"). The doc is
// validated on the admin WRITE path (domain.MFAPolicySpec.Validate, wired in
// admin_pg.go); here we read it at runtime to enforce the part that is
// well-defined inside IAM: the allowed_factors enrollment gate.
//
// Mirrors coreAuthLoadPasswordPolicy / the captcha config reader: read the row
// via models.FindIamConfig(project, effectiveEnv, key), unmarshal row.Data into
// the typed domain spec, and treat a missing row as "no policy" — which keeps a
// project with no mfa_policy doc behaving exactly as before (every implemented
// factor enrollable).
//
// Field semantics:
//   - allowed_factors    — gate enrollment of new factors (ENFORCED here, §2).
//   - required_for_admins — loaded but a no-op at the moment: IAM has no
//                           admin-role subject on end-user accounts, and the
//                           existing AuthenticatePassword path already forces the
//                           second factor for any account that HAS one enrolled.
//                           We deliberately do NOT hard-block 0-factor accounts
//                           (that would lock out every existing user). The flag
//                           becomes enforceable once an admin-role subject exists
//                           (AuthZ integration).
//   - remember_device     — loaded but not acted on: a trusted-device skip
//                           requires a verified device-fingerprint signal that is
//                           not captured at login yet. Skipping MFA without that
//                           signal would be a security regression, so it is left
//                           as a no-op rather than silently weakening MFA.

import (
	"context"
	"errors"

	"github.com/gopherex/iam/internal/domain"
	models "github.com/gopherex/iam/internal/infrastructure/postgres/gen/bob/models"
)

// mfaLoadPolicy reads iam_config(project, effectiveEnv, key="mfa_policy") and
// decodes it into a domain.MFAPolicySpec. A missing row returns the zero value
// (AllowedFactors == nil => FactorAllowed is true for every factor), i.e. the
// project is treated as having no MFA policy. The stored doc is plain JSON (the
// admin write path stores real JSON, not base64), so unmarshalling row.Data into
// the spec works directly.
func (a *pgMFAAccounts) mfaLoadPolicy(ctx context.Context, projectID string) (domain.MFAPolicySpec, error) {
	var pol domain.MFAPolicySpec
	env, err := effectiveEnv(ctx, a.db, projectID, mfaDefaultEnv)
	if err != nil {
		return pol, err
	}
	row, err := models.FindIamConfig(ctx, a.db.Bobx(), projectID, env, "mfa_policy")
	if err != nil {
		if errors.Is(translatePgErr("config", err), ErrNotFound) {
			return pol, nil // no policy => allow all (backward compatible)
		}
		return pol, err
	}
	if len(row.Data) > 0 {
		if err := unmarshal(row.Data, &pol); err != nil {
			return pol, err
		}
	}
	return pol, nil
}

// mfaGateEnroll loads the project's mfa_policy and denies the enrollment of a
// factor type the policy does not list in allowed_factors. dbFactorType is the
// internal/DB factor name ("totp"/"sms"/"email"/"webauthn"/"recovery"); the
// policy-name mapping (email->email_otp, recovery->backup_codes) is handled by
// domain.MFAPolicySpec.FactorAllowed. A missing policy or an unset/empty
// allowed_factors list allows every factor (no change from prior behaviour).
//
// Only ENROLLMENT is gated. Challenge/verify of a factor enrolled before a
// policy tightening must keep working so a policy change never locks users out.
func (a *pgMFAAccounts) mfaGateEnroll(ctx context.Context, projectID, dbFactorType string) error {
	pol, err := a.mfaLoadPolicy(ctx, projectID)
	if err != nil {
		return err
	}
	if !pol.FactorAllowed(dbFactorType) {
		return domain.ErrMFAFactorNotAllowed.WithMessage(
			"MFA factor " + domain.MFAPolicyFactorName(dbFactorType) + " is not permitted by policy")
	}
	return nil
}
