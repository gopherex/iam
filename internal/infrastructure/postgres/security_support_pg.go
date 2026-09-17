package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
)

type securityCasePrivate struct {
	Protected         bool     `json:"protected"`
	AllSessions       bool     `json:"all_sessions"`
	TargetSession     string   `json:"target_session"`
	RevokedSessionIDs []string `json:"revoked_session_ids"`
	Contact           string   `json:"contact"`
	Identifier        string   `json:"identifier"`
}

func (s *pgSecurity) Case(ctx context.Context, scope domain.SecurityScope, id string) (*domain.SecurityCase, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	var c domain.SecurityCase

	err = s.load(ctx, scope, securityCases, id, &c, nil)

	return &c, securityStoreError(err)
}

//nolint:funlen,cyclop // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) openCase(ctx context.Context, f *securityFlow, message string) error {
	scope := securityFlowScope(f)
	if f.Identifier == "" && f.AccountID != "" {
		a, err := s.account(ctx, scope)
		if err != nil {
			return securityStoreError(err)
		}

		f.Identifier = a.PrimaryEmail
	}

	if f.Identifier == "" {
		return domain.ErrValidation.WithMessage("Account identifier is required")
	}

	key := accountHashToken(f.Identifier + "\x00" + f.Recipient)

	var raw []byte

	err := s.db.TxDB.QueryRow(
		ctx,
		`SELECT data FROM iam_security_cases WHERE project_id=$1 AND environment=$2 AND data->>'request_key'=$3
AND status IN ('pending','needs_information','approved') ORDER BY created_at DESC
LIMIT 1`,
		f.ProjectID,
		f.Environment,
		key).Scan(&raw)
	if err == nil {
		return s.resumeSupportCase(ctx, f, raw)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return securityStoreError(err)
	}

	accountID := f.AccountID
	if accountID == "" {
		err = s.db.TxDB.QueryRow(
			ctx,
			`SELECT id FROM iam_users WHERE project_id=$1 AND environment=$2 AND (primary_email=$3
OR primary_phone=$3)`,
			f.ProjectID,
			f.Environment,
			f.Identifier).Scan(&accountID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return securityStoreError(err)
		}
	}

	now := nowIn(ctx)

	c := domain.SecurityCase{
		ID:            newUUID(),
		AccountID:     accountID,
		Status:        securityPending,
		ContactMasked: securityMask(f.Recipient),
		CreatedAt:     now,
		UpdatedAt:     now,
		Messages:      []domain.SecurityCaseMessage{},
	}
	if strings.TrimSpace(message) != "" {
		c.Messages = append(
			c.Messages,
			domain.SecurityCaseMessage{At: now, Author: "user", Message: message})
	}

	private, err := s.encrypt(
		securityCasePrivate{
			Contact:           f.Recipient,
			Identifier:        f.Identifier,
			Protected:         f.Protected,
			AllSessions:       f.State.AllSessions,
			TargetSession:     f.TargetSession,
			RevokedSessionIDs: f.State.RevokedSessionIDs,
		})
	if err != nil {
		return securityStoreError(err)
	}

	raw, err = json.Marshal(c)
	if err != nil {
		return securityStoreError(err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return securityStoreError(err)
	}

	payload["request_key"] = key

	raw, err = json.Marshal(payload)
	if err != nil {
		return securityStoreError(err)
	}

	_, err = s.db.TxDB.Exec(
		ctx,
		`INSERT INTO iam_security_cases(id,project_id,environment,user_id,status,created_at,data,private_data)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		c.ID,
		f.ProjectID,
		f.Environment,
		accountID,
		c.Status,
		now,
		raw,
		private)
	if err != nil {
		return securityStoreError(err)
	}

	f.CaseID = c.ID
	f.State.Case = &c
	f.State.Step = securityAwaitingSupport

	if err := s.emit(
		ctx,
		scope,
		"security.recovery.requested",
		c.ID,
		map[string]any{"case_id": c.ID}); err != nil {
		return securityStoreError(err)
	}

	return s.sendCaseContinuation(ctx, scope, &c, f.Recipient, "support_status")
}

func (s *pgSecurity) caseMessage(ctx context.Context, f *securityFlow, message string) error {
	message = strings.TrimSpace(message)
	if message == "" || len(message) > 4096 {
		return domain.ErrValidation
	}

	scope := domain.SecurityScope{ProjectID: f.ProjectID, Environment: f.Environment}

	c, err := s.Case(ctx, scope, f.CaseID)
	if err != nil {
		return securityStoreError(err)
	}

	if c.Status != securityPending && c.Status != securityNeedsInformation {
		return domain.ErrConflict
	}

	if len(c.Messages) >= securityMessageHistoryLimit {
		return domain.ErrRateLimited
	}

	c.Messages = append(
		c.Messages,
		domain.SecurityCaseMessage{At: nowIn(ctx), Author: "user", Message: message})
	c.Status = securityPending
	c.UpdatedAt = nowIn(ctx)

	return s.saveCase(ctx, scope, c)
}

func (s *pgSecurity) saveCase(
	ctx context.Context,
	scope domain.SecurityScope,
	c *domain.SecurityCase,
) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return securityStoreError(err)
	}

	_, err = s.db.TxDB.Exec(
		ctx,
		`UPDATE iam_security_cases SET data=data || $1::jsonb,status=$2 WHERE id=$3 AND project_id=$4
AND environment=$5`,
		raw,
		c.Status,
		c.ID,
		scope.ProjectID,
		scope.Environment)

	return securityStoreError(err)
}

//nolint:funlen,cyclop // Keep ordered checks and writes together in this security transaction.
func (s *pgSecurity) DecideCase(
	ctx context.Context,
	scope domain.SecurityScope,
	id string,
	in domain.SecurityCaseDecision,
) (*domain.SecurityCase, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	if scope.ActorID == "" || strings.TrimSpace(in.Message) == "" || strings.TrimSpace(in.Evidence) == "" {
		return nil, domain.ErrValidation.WithMessage("Decision, user message and verification evidence are required")
	}

	return withTxRet(ctx, s.db, func(ctx context.Context) (*domain.SecurityCase, error) {
		var (
			c       domain.SecurityCase
			private securityCasePrivate
		)

		if err := s.load(ctx, scope, securityCases, id, &c, &private); err != nil {
			return nil, securityStoreError(err)
		}

		if c.Status != securityPending && c.Status != securityNeedsInformation {
			return nil, domain.ErrConflict
		}

		switch in.Action {
		case "approve":
			if err := s.validateSupportAccount(ctx, scope, &c); err != nil {
				return nil, securityStoreError(err)
			}

			c.Status = securityApproved
		case "reject":
			c.Status = "rejected"
		case "request_information":
			c.Status = securityNeedsInformation
		default:
			return nil, domain.ErrValidation
		}

		c.UpdatedAt = nowIn(ctx)

		c.Messages = append(
			c.Messages,
			domain.SecurityCaseMessage{At: c.UpdatedAt, Author: "support", Message: in.Message})
		if err := s.saveCase(ctx, scope, &c); err != nil {
			return nil, securityStoreError(err)
		}

		evidence, err := s.db.Cipher.Encrypt(in.Evidence)
		if err != nil {
			return nil, securityStoreError(err)
		}

		_, err = s.db.TxDB.Exec(
			ctx,
			`INSERT INTO
iam_security_case_decisions(id,project_id,environment,case_id,actor_id,action,evidence,created_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
			newUUID(),
			scope.ProjectID,
			scope.Environment,
			c.ID,
			scope.ActorID,
			in.Action,
			evidence,
			c.UpdatedAt)
		if err != nil {
			return nil, securityStoreError(err)
		}

		purpose := "support_status"
		if c.Status == securityApproved {
			purpose = securitySupportGrant
		}

		if err := s.sendCaseContinuation(ctx, scope, &c, private.Contact, purpose); err != nil {
			return nil, securityStoreError(err)
		}

		if err := s.emit(
			ctx,
			scope,
			"security.recovery.decided",
			c.ID,
			map[string]any{"case_id": c.ID, "status": c.Status, "actor_id": scope.ActorID}); err != nil {
			return nil, securityStoreError(err)
		}

		return &c, nil
	})
}

func (s *pgSecurity) sendCaseContinuation(
	ctx context.Context,
	scope domain.SecurityScope,
	c *domain.SecurityCase,
	contact, purpose string,
) error {
	scope.AccountID = c.AccountID

	token, err := s.continuation(ctx, scope, "", c.ID, purpose)
	if err != nil {
		return securityStoreError(err)
	}

	return s.queueDelivery(
		ctx,
		scope,
		securityDeliveryPayload{
			To:           contact,
			Template:     securityRecoveryTemplate,
			Continuation: token,
			Summary:      c.Status,
		},
		"",
		c.ID,
		"case:"+c.ID+":"+newUUID())
}

func (s *pgSecurity) continuation(
	ctx context.Context,
	scope domain.SecurityScope,
	incidentID, caseID, purpose string,
) (string, error) {
	token, err := accountRandomToken(securityTokenBytes)
	if err != nil {
		return "", securityStoreError(err)
	}

	if purpose == securitySupportGrant {
		_, err = s.db.TxDB.Exec(
			ctx,
			`UPDATE iam_security_continuations SET consumed=true,exchange_data='' WHERE project_id=$1 AND environment=$2
AND case_id=$3 AND purpose='support_grant'`,
			scope.ProjectID,
			scope.Environment,
			caseID)
		if err != nil {
			return "", securityStoreError(err)
		}
	}

	_, err = s.db.TxDB.Exec(
		ctx,
		`INSERT INTO
iam_security_continuations(token_hash,project_id,environment,user_id,incident_id,case_id,purpose,expires_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		accountHashToken(token),
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		incidentID,
		caseID,
		purpose,
		nowIn(ctx).Add(securityContinuationTTL))

	return token, securityStoreError(err)
}

func (s *pgSecurity) applySupportContact(ctx context.Context, f *securityFlow) error {
	// The support grant authorizes the new contact that support reviewed. The
	// user cannot replace it in a submit request.
	a, err := s.account(ctx, securityFlowScope(f))
	if err != nil {
		return securityStoreError(err)
	}

	a.PrimaryEmail = f.Recipient
	a.EmailVerified = true
	a.PrimaryPhone = ""
	a.PhoneVerified = false
	a.UpdatedAt = nowIn(ctx)

	raw, err := json.Marshal(a) //nolint:musttag // Preserve stored field names.
	if err != nil {
		return securityStoreError(err)
	}

	_, err = s.db.TxDB.Exec(
		ctx,
		`UPDATE iam_users SET primary_email=$1,primary_phone=NULL,data=data || $2::jsonb,updated_at=$3
WHERE id=$4 AND project_id=$5 AND environment=$6`,
		f.Recipient,
		raw,
		nowIn(ctx),
		f.AccountID,
		f.ProjectID,
		f.Environment)

	return translatePgErr("account", err)
}

func (s *pgSecurity) resumeSupportCase(ctx context.Context, f *securityFlow, raw []byte) error {
	scope := securityFlowScope(f)

	var c domain.SecurityCase
	if err := json.Unmarshal(raw, &c); err != nil {
		return securityStoreError(err)
	}

	if f.Protected {
		if err := s.saveSupportProtection(ctx, f, c.ID); err != nil {
			return err
		}
	}

	f.CaseID = c.ID
	f.State.Case = &c

	f.State.Step = securityAwaitingSupport
	if c.Status == securityApproved {
		return s.sendCaseContinuation(ctx, scope, &c, f.Recipient, securitySupportGrant)
	}

	return nil
}

func (s *pgSecurity) validateSupportAccount(
	ctx context.Context,
	scope domain.SecurityScope,
	c *domain.SecurityCase,
) error {
	if c.AccountID == "" {
		return domain.ErrValidation.WithMessage("The requested account could not be identified")
	}

	a, err := s.account(
		ctx,
		domain.SecurityScope{
			ProjectID:   scope.ProjectID,
			Environment: scope.Environment,
			AccountID:   c.AccountID,
		})
	if err != nil {
		return securityStoreError(err)
	}

	if a.Status != "active" {
		return domain.ErrAccountSuspended
	}

	return nil
}

func (s *pgSecurity) saveSupportProtection(ctx context.Context, f *securityFlow, caseID string) error {
	var (
		current domain.SecurityCase
		private securityCasePrivate
	)

	if err := s.load(ctx, securityFlowScope(f), securityCases, caseID, &current, &private); err != nil {
		return err
	}

	private.Protected = true
	private.AllSessions = f.State.AllSessions
	private.TargetSession = f.TargetSession
	private.RevokedSessionIDs = f.State.RevokedSessionIDs

	encrypted, err := s.encrypt(private)
	if err != nil {
		return err
	}

	_, err = s.db.TxDB.Exec(ctx, `UPDATE iam_security_cases SET private_data=$1
 WHERE id=$2 AND project_id=$3 AND environment=$4`, encrypted, caseID, f.ProjectID, f.Environment)

	return securityStoreError(err)
}
