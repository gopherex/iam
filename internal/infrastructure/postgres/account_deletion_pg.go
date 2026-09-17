package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

const (
	deletionConfigKey   = "account_deletion"
	deletionDefaultDays = 7
	deletionMaxDays     = 365
	deletionDay         = 24 * time.Hour
	deletionPending     = "pending"
	deletionCancelled   = "cancelled"
	deletionDeleted     = "deleted"
	deletionRequest     = "deletion_request"
	deletionCancel      = "deletion_cancel"
	deletionAuthorized  = "deletion_authorized"
	deletionProofTTL    = 5 * time.Minute
)

func (a *pgAccountStore) DeletionPolicy(
	ctx context.Context,
	scope domain.SecurityScope,
) (*domain.AccountDeletionPolicy, error) {
	scope, err := NewPgSecurity(a.db, a.emitter).scope(ctx, scope)
	if err != nil {
		return nil, err
	}

	policy := domain.AccountDeletionPolicy{GraceDays: deletionDefaultDays}

	var raw []byte

	err = a.db.TxDB.QueryRow(
		ctx,
		`SELECT data FROM iam_config WHERE project_id=$1 AND environment=$2 AND key=$3`,
		scope.ProjectID,
		scope.Environment,
		deletionConfigKey).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return &policy, nil
	}

	if err != nil {
		return nil, securityStoreError(err)
	}

	if err := json.Unmarshal(raw, &policy); err != nil {
		return nil, securityStoreError(err)
	}

	return &policy, nil
}

func (a *pgAccountStore) SetDeletionPolicy(
	ctx context.Context,
	scope domain.SecurityScope,
	policy domain.AccountDeletionPolicy,
) (*domain.AccountDeletionPolicy, error) {
	scope, err := NewPgSecurity(a.db, a.emitter).scope(ctx, scope)
	if err != nil {
		return nil, err
	}

	if policy.GraceDays < 1 || policy.GraceDays > deletionMaxDays {
		return nil, domain.ErrValidation.WithMessage("Deletion delay must be between 1 and 365 days")
	}

	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AccountDeletionPolicy, error) {
		raw, err := json.Marshal(policy)
		if err != nil {
			return nil, securityStoreError(err)
		}

		_, err = a.db.TxDB.Exec(
			ctx,
			`INSERT INTO iam_config(project_id,environment,key,data) VALUES($1,$2,$3,$4)
 ON CONFLICT(project_id,environment,key) DO UPDATE SET data=EXCLUDED.data,updated_at=now()`,
			scope.ProjectID,
			scope.Environment,
			deletionConfigKey,
			raw)
		if err != nil {
			return nil, securityStoreError(err)
		}

		err = a.emitter.Emit(
			ctx,
			domain.Event{
				Type:        "account.deletion_policy.updated",
				ProjectID:   scope.ProjectID,
				Environment: scope.Environment,
				AggregateID: scope.ProjectID,
				Payload:     policy,
			})

		return &policy, err
	})
}

func (a *pgAccountStore) DeletionStatus(
	ctx context.Context,
	scope domain.SecurityScope,
) (*domain.AccountDeletion, error) {
	scope, err := NewPgSecurity(a.db, a.emitter).scope(ctx, scope)
	if err != nil {
		return nil, err
	}

	var raw []byte

	err = a.db.TxDB.QueryRow(
		ctx,
		`SELECT data FROM iam_account_deletions WHERE project_id=$1 AND environment=$2 AND user_id=$3`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		policy, err := a.DeletionPolicy(ctx, scope)
		if err != nil {
			return nil, err
		}

		return &domain.AccountDeletion{OK: true, Status: "none", GraceDays: policy.GraceDays}, nil
	}

	if err != nil {
		return nil, securityStoreError(err)
	}

	var state domain.AccountDeletion
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, securityStoreError(err)
	}

	return &state, nil
}

// MutateDeletion never changes ordinary account access before the fixed deadline.
func (a *pgAccountStore) MutateDeletion(
	ctx context.Context,
	scope domain.SecurityScope,
	in domain.AccountDeletionInput,
	action string,
) (*domain.AccountDeletion, error) {
	if action != deletionRequest && action != deletionCancel {
		return nil, domain.ErrBadRequest
	}

	security := NewPgSecurity(a.db, a.emitter)

	scope, err := security.scope(ctx, scope)
	if err != nil {
		return nil, err
	}

	ctx = api.WithEnvironment(ctx, scope.Environment)
	if err := securityAccountGuard(ctx, a.db, scope.ProjectID, scope.AccountID); err != nil {
		return nil, err
	}

	hasPassword, err := a.deletionPassword(ctx, scope, in.Password)
	if err != nil {
		return nil, err
	}
	// Proof creation must commit even when this operation returns step_up_required.
	if err := a.requireDeletionProof(ctx, scope, in, action, hasPassword); err != nil {
		return nil, err
	}

	return withTxRet(ctx, a.db, func(ctx context.Context) (*domain.AccountDeletion, error) {
		return a.mutateDeletionLocked(ctx, scope, in, action, hasPassword)
	})
}

func (a *pgAccountStore) mutateDeletionLocked(
	ctx context.Context,
	scope domain.SecurityScope,
	in domain.AccountDeletionInput,
	action string,
	hasPassword bool,
) (*domain.AccountDeletion, error) {
	if err := a.lockDeletionAccount(ctx, scope); err != nil {
		return nil, err
	}

	if err := a.recheckDeletionPassword(ctx, scope, in.Password, hasPassword); err != nil {
		return nil, err
	}

	if err := securityAccountGuard(ctx, a.db, scope.ProjectID, scope.AccountID); err != nil {
		return nil, err
	}

	state, err := a.DeletionStatus(ctx, scope)
	if err != nil {
		return nil, err
	}

	if state.Status == deletionPending && !nowIn(ctx).Before(*state.DeleteAt) {
		return nil, domain.ErrConflict.WithMessage("The account deletion deadline has passed")
	}

	if action == deletionRequest && state.Status == deletionPending {
		return state, nil
	}

	if action == deletionCancel && state.Status != deletionPending {
		return state, nil
	}

	if err := a.consumeDeletionProof(ctx, scope, in.ProofToken, action, hasPassword); err != nil {
		return nil, err
	}

	return a.changeDeletion(ctx, scope, state, action, in.Reason)
}

func (a *pgAccountStore) lockDeletionAccount(ctx context.Context, scope domain.SecurityScope) error {
	var id string

	err := a.db.TxDB.QueryRow(
		ctx,
		`SELECT id FROM iam_users WHERE id=$1 AND project_id=$2 AND environment=$3 FOR UPDATE`,
		scope.AccountID,
		scope.ProjectID,
		scope.Environment).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	}

	return securityStoreError(err)
}

func (a *pgAccountStore) changeDeletion(
	ctx context.Context,
	scope domain.SecurityScope,
	state *domain.AccountDeletion,
	action, reason string,
) (*domain.AccountDeletion, error) {
	now := nowIn(ctx)
	event := domain.WebhookEventDeletionCancelled

	if action == deletionRequest {
		policy, err := a.DeletionPolicy(ctx, scope)
		if err != nil {
			return nil, err
		}

		deadline := now.Add(time.Duration(policy.GraceDays) * deletionDay)
		state = &domain.AccountDeletion{
			OK:          true,
			RequestID:   newUUID(),
			Status:      deletionPending,
			RequestedAt: &now,
			DeleteAt:    &deadline,
			GraceDays:   policy.GraceDays,
		}
		event = domain.WebhookEventDeletionScheduled
	} else {
		state.Status = deletionCancelled
		state.CancelledAt = &now
	}

	if err := a.saveDeletion(ctx, scope, state); err != nil {
		return nil, err
	}

	if err := a.recordDeletion(ctx, scope, state, event, reason); err != nil {
		return nil, err
	}

	if err := a.notifyDeletion(ctx, scope, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (a *pgAccountStore) saveDeletion(
	ctx context.Context,
	scope domain.SecurityScope,
	state *domain.AccountDeletion,
) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return securityStoreError(err)
	}

	_, err = a.db.TxDB.Exec(
		ctx,
		`INSERT INTO iam_account_deletions(project_id,environment,user_id,request_id,status,delete_at,data)
 VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(project_id,environment,user_id) DO UPDATE
 SET request_id=EXCLUDED.request_id,status=EXCLUDED.status,delete_at=EXCLUDED.delete_at,data=EXCLUDED.data`,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		state.RequestID,
		state.Status,
		state.DeleteAt,
		raw)
	if err != nil {
		return securityStoreError(err)
	}

	_, err = a.db.TxDB.Exec(
		ctx,
		`UPDATE iam_users SET data=jsonb_set(data,'{deletion}',$1::jsonb),updated_at=$2
 WHERE id=$3 AND project_id=$4 AND environment=$5`,
		raw,
		nowIn(ctx),
		scope.AccountID,
		scope.ProjectID,
		scope.Environment)

	return securityStoreError(err)
}

func (a *pgAccountStore) recordDeletion(
	ctx context.Context,
	scope domain.SecurityScope,
	state *domain.AccountDeletion,
	typ, reason string,
) error {
	payload := map[string]any{
		"user_id":     scope.AccountID,
		"environment": scope.Environment,
		"request_id":  state.RequestID,
		"delete_at":   state.DeleteAt,
		"reason":      reason,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return securityStoreError(err)
	}

	actor := scope.AccountID
	if scope.ActorID != "" {
		actor = scope.ActorID
	}

	if typ == domain.WebhookEventUserDeleted {
		actor = "account_deletion_worker"
	}

	if err := NewPgAudit(a.db, a.emitter).record(ctx, scope.ProjectID, typ, actor, scope.AccountID, raw); err != nil {
		return err
	}

	return a.emitter.Emit(
		ctx,
		domain.Event{
			Type:        typ,
			ProjectID:   scope.ProjectID,
			Environment: scope.Environment,
			AggregateID: scope.AccountID,
			Payload:     payload,
		})
}

func (a *pgAccountStore) DeletionHistory(
	ctx context.Context,
	scope domain.SecurityScope,
) (*domain.AccountDeletionHistory, error) {
	scope, err := NewPgSecurity(a.db, a.emitter).scope(ctx, scope)
	if err != nil {
		return nil, err
	}

	state, err := a.DeletionStatus(ctx, scope)
	if err != nil {
		return nil, err
	}

	rows, err := a.db.TxDB.Query(
		ctx,
		`SELECT type,at,data->>'request_id',COALESCE(actor_id,''),COALESCE(data->>'reason','') FROM iam_audit_logs
 WHERE project_id=$1 AND target_id=$2 AND data->>'environment'=$3 AND data ? 'request_id'
 ORDER BY at DESC,id DESC LIMIT 100`,
		scope.ProjectID,
		scope.AccountID,
		scope.Environment)
	if err != nil {
		return nil, securityStoreError(err)
	}
	defer rows.Close()

	history := &domain.AccountDeletionHistory{Deletion: *state, Events: []domain.AccountDeletionEvent{}}

	for rows.Next() {
		var event domain.AccountDeletionEvent
		if err := rows.Scan(&event.Type, &event.At, &event.RequestID, &event.ActorID, &event.Reason); err != nil {
			return nil, securityStoreError(err)
		}

		history.Events = append(history.Events, event)
	}

	return history, securityStoreError(rows.Err())
}

func (a *pgAccountStore) notifyDeletion(
	ctx context.Context,
	scope domain.SecurityScope,
	state *domain.AccountDeletion,
) error {
	security := NewPgSecurity(a.db, a.emitter)

	account, err := security.account(ctx, scope)
	if err != nil {
		return err
	}

	contact := ""
	if account.EmailVerified {
		contact = account.PrimaryEmail
	} else if account.PhoneVerified {
		contact = account.PrimaryPhone
	}

	if contact == "" {
		return nil
	}

	policy, err := security.Policy(ctx, scope)
	if err != nil {
		return err
	}

	summary := "Account deletion cancelled. Your account remains available."
	if state.Status == deletionPending {
		summary = "Account deletion scheduled for " + state.DeleteAt.UTC().Format(time.RFC3339) +
			". Sign in and choose Cancel deletion before that date. " +
			"Your account remains available until then. Signing in does not cancel deletion."
	}

	if strings.HasPrefix(account.Locale, "ru") {
		summary = "Удаление аккаунта отменено. Вы можете пр" +
			"одолжать пользоваться аккаунтом."
		if state.Status == deletionPending {
			summary = "Аккаунт будет удалён " + state.DeleteAt.UTC().Format(time.RFC3339) +
				". До этой даты доступ сохранится. Чтобы " +
				"отменить удаление, войдите в приложение " +
				"и выберите отмену удаления. Сам вход не " +
				"отменяет заявку."
		}
	}

	return security.queueDelivery(
		ctx,
		scope,
		securityDeliveryPayload{
			To:       contact,
			Template: "account_deletion",
			Summary:  summary,
			Link:     policy.ContinueURL,
			Locale:   account.Locale,
		},
		"",
		"",
		"deletion:"+state.RequestID+":"+state.Status)
}

func accountDeletionAccess(ctx context.Context, db *DB, project, env, account string) error {
	var due bool

	err := db.TxDB.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM iam_account_deletions
 WHERE project_id=$1 AND environment=$2 AND user_id=$3 AND status IN ('pending','deleted') AND delete_at<=$4)`,
		project,
		env,
		account,
		nowIn(ctx)).Scan(&due)
	if err != nil {
		return securityStoreError(err)
	}

	if due {
		return domain.ErrForbidden.WithMessage("The account deletion deadline has passed")
	}

	return nil
}
