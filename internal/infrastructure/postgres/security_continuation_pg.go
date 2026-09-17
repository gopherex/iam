package postgres

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gopherex/iam/internal/domain"
)

const securityExchangeMode = "exchange"

type securityExchange struct {
	KeyHash string    `json:"key_hash"`
	Token   string    `json:"token"`
	Until   time.Time `json:"until"`
}

// The continuation alone cannot retrieve a consumed flow. Only the same
// client-held secret may retry, before any flow mutation.
func (s *pgSecurity) replayContinuation(ctx context.Context, scope domain.SecurityScope,
	in domain.SecurityFlowInput,
) (*domain.SecurityFlowState, error) {
	var (
		consumed  bool
		encrypted string
	)

	err := s.db.TxDB.QueryRow(ctx, `SELECT consumed,exchange_data FROM iam_security_continuations
WHERE token_hash=$1 AND project_id=$2 AND environment=$3 AND expires_at>$4`,
		accountHashToken(in.ContinuationToken), scope.ProjectID, scope.Environment, nowIn(ctx)).Scan(&consumed, &encrypted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInvalidToken
	}

	if err != nil {
		return nil, securityStoreError(err)
	}

	if !consumed {
		return nil, ErrNotFound
	}

	if in.ExchangeKey == "" || encrypted == "" {
		return nil, domain.ErrInvalidToken
	}

	var replay securityExchange

	if err := s.decrypt(encrypted, &replay); err != nil {
		return nil, err
	}

	if !nowIn(ctx).Before(replay.Until) ||
		subtle.ConstantTimeCompare([]byte(replay.KeyHash), []byte(accountHashToken(in.ExchangeKey))) != 1 {
		return nil, domain.ErrInvalidToken
	}

	f, err := s.loadFlow(ctx, scope, replay.Token)
	if err != nil {
		return nil, err
	}

	if f.CurrentToken != replay.Token || f.State.Status != securityPending {
		return nil, domain.ErrInvalidToken
	}

	if err := s.validateFlowAuthority(ctx, f); err != nil {
		return nil, err
	}

	return s.viewFlow(ctx, f, replay.Token)
}

func (s *pgSecurity) rememberContinuation(ctx context.Context, scope domain.SecurityScope,
	in domain.SecurityFlowInput, token string,
) error {
	encrypted, err := s.encrypt(securityExchange{
		KeyHash: accountHashToken(in.ExchangeKey),
		Token:   token,
		Until:   nowIn(ctx).Add(securityRetryGrace),
	})
	if err != nil {
		return err
	}

	_, err = s.db.TxDB.Exec(ctx, `UPDATE iam_security_continuations SET exchange_data=$1
WHERE token_hash=$2 AND project_id=$3 AND environment=$4`,
		encrypted, accountHashToken(in.ContinuationToken), scope.ProjectID, scope.Environment)

	return securityStoreError(err)
}
