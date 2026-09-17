package postgres

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/pkg/api"
)

type securityDeliveryPayload struct {
	To           string `json:"to"`
	Template     string `json:"template"`
	Code         string `json:"code,omitempty"`
	Continuation string `json:"continuation,omitempty"`
	Summary      string `json:"summary,omitempty"`
	Link         string `json:"link,omitempty"`
	Locale       string `json:"locale,omitempty"`
}

// SecurityDeliveryJob is only available to the internal delivery worker.
type SecurityDeliveryJob struct {
	ExpiresAt   *time.Time
	Channel     string
	CreatedAt   time.Time
	ID          string
	ProjectID   string
	Environment string
	Status      string
	To          string
	Template    string
	Code        string
	Link        string
	Summary     string
	Locale      string
}

func (s *pgSecurity) queueDelivery(
	ctx context.Context,
	scope domain.SecurityScope,
	payload securityDeliveryPayload,
	incidentID, caseID, dedup string,
) error {
	if err := s.deliveryLink(ctx, scope, &payload); err != nil {
		return err
	}

	now := nowIn(ctx)

	v := domain.SecurityDelivery{
		ID:              newUUID(),
		IncidentID:      incidentID,
		CaseID:          caseID,
		Channel:         securityEmail,
		RecipientMasked: securityMask(payload.To),
		Status:          "queued",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if !strings.Contains(payload.To, "@") {
		v.Channel = securitySMS
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return securityStoreError(err)
	}

	private, err := s.encrypt(payload)
	if err != nil {
		return securityStoreError(err)
	}

	tag, err := s.db.TxDB.Exec(
		ctx,
		`INSERT INTO
iam_security_deliveries(id,project_id,environment,user_id,dedup_key,status,created_at,data,private_data)
VALUES($1,$2,$3,$4,$5,'queued',$6,$7,$8) ON CONFLICT(dedup_key) DO NOTHING`,
		v.ID,
		scope.ProjectID,
		scope.Environment,
		scope.AccountID,
		dedup,
		now,
		raw,
		private)
	if err != nil {
		return securityStoreError(err)
	}

	if tag.RowsAffected() == 0 {
		return nil
	}

	return s.emit(
		ctx,
		scope,
		"security.delivery.requested",
		v.ID,
		map[string]any{"delivery_id": v.ID})
}

func (s *pgSecurity) DeliveryJob(
	ctx context.Context,
	scope domain.SecurityScope,
	id string,
) (*SecurityDeliveryJob, error) {
	var (
		delivery domain.SecurityDelivery
		p        securityDeliveryPayload
	)

	if err := s.load(ctx, scope, securityDeliveries, id, &delivery, &p); err != nil {
		return nil, securityStoreError(err)
	}

	return &SecurityDeliveryJob{
		ExpiresAt:   securityDeliveryExpiry(delivery.CreatedAt, p),
		Channel:     delivery.Channel,
		CreatedAt:   delivery.CreatedAt,
		ID:          id,
		ProjectID:   scope.ProjectID,
		Environment: scope.Environment,
		Status:      delivery.Status,
		To:          p.To,
		Template:    p.Template,
		Code:        p.Code,
		Link:        p.Link,
		Summary:     p.Summary,
		Locale:      p.Locale,
	}, nil
}

func (s *pgSecurity) DeliveryResult(
	ctx context.Context,
	scope domain.SecurityScope,
	id, status, code string,
) error {
	return s.db.withTx(ctx, func(ctx context.Context) error {
		var delivery domain.SecurityDelivery
		if err := s.load(ctx, scope, securityDeliveries, id, &delivery, nil); err != nil {
			return securityStoreError(err)
		}

		delivery.Status = status
		delivery.LastError = code
		delivery.Attempts++
		delivery.UpdatedAt = nowIn(ctx)

		return s.save(ctx, scope, securityDeliveries, id, status, delivery)
	})
}

func (s *pgSecurity) RetryDelivery(
	ctx context.Context,
	scope domain.SecurityScope,
	id string,
) (*domain.SecurityDelivery, error) {
	scope, err := s.scope(ctx, scope)
	if err != nil {
		return nil, securityStoreError(err)
	}

	return withTxRet(ctx, s.db, func(ctx context.Context) (*domain.SecurityDelivery, error) {
		var delivery domain.SecurityDelivery

		var payload securityDeliveryPayload

		if err := s.load(ctx, scope, securityDeliveries, id, &delivery, &payload); err != nil {
			return nil, securityStoreError(err)
		}

		if delivery.Status == "accepted" {
			return &delivery, nil
		}

		if expiry := securityDeliveryExpiry(delivery.CreatedAt, payload); expiry != nil && !nowIn(ctx).Before(*expiry) {
			return nil, domain.ErrTokenExpired.WithMessage(
				"Start a fresh continuation; this delivery contains expired proof")
		}

		delivery.Status = "queued"
		delivery.LastError = ""

		delivery.UpdatedAt = nowIn(ctx)
		if err := s.save(ctx, scope, securityDeliveries, id, delivery.Status, delivery); err != nil {
			return nil, securityStoreError(err)
		}

		if err := s.emit(
			ctx,
			scope,
			"security.delivery.requested",
			id,
			map[string]any{"delivery_id": id}); err != nil {
			return nil, securityStoreError(err)
		}

		return &delivery, nil
	})
}

func (s *pgSecurity) notifyStatus(ctx context.Context, f *securityFlow, status string) error {
	scope := securityFlowScope(f)

	policy, err := s.Policy(ctx, scope)
	if err != nil {
		return securityStoreError(err)
	}

	if !policy.Notify || policy.Mode != securityEnforce {
		return nil
	}

	recipients := []string{f.Recipient}

	account, err := s.account(ctx, scope)
	if err != nil {
		return securityStoreError(err)
	}

	if account.EmailVerified && account.PrimaryEmail != f.Recipient {
		recipients = append(recipients, account.PrimaryEmail)
	}

	for _, recipient := range recipients {
		if recipient != "" {
			if err := s.queueDelivery(
				ctx,
				scope,
				securityDeliveryPayload{To: recipient, Template: "security_status", Summary: status},
				f.IncidentID,
				f.CaseID,
				"status:"+f.ID+":"+status+":"+accountHashToken(recipient)); err != nil {
				return securityStoreError(err)
			}
		}
	}

	return nil
}

func (s *pgSecurity) deliveryLink(
	ctx context.Context,
	scope domain.SecurityScope,
	payload *securityDeliveryPayload,
) error {
	policy, err := s.Policy(ctx, scope)
	if err != nil {
		return securityStoreError(err)
	}

	if payload.Continuation == "" {
		return nil
	}
	{
		if principal, ok := api.PrincipalFrom(ctx); ok && principal.ClientID != "" {
			if configured := policy.ClientURLs[principal.ClientID]; configured != "" {
				policy.ContinueURL = configured
			}
		}

		if !securityURLValid(policy.ContinueURL) {
			return domain.ErrValidation.WithMessage(
				"Configure the security continuation URL before starting recovery")
		}

		continuationURL, err := url.Parse(policy.ContinueURL)
		if err != nil {
			return securityStoreError(err)
		}

		continuationURL.Fragment = "iam_security=" + url.QueryEscape(payload.Continuation)
		payload.Link = continuationURL.String()
	}

	return nil
}

// Status notifications have no expiring credential. Only proof-bearing payloads expire.
func securityDeliveryExpiry(created time.Time, payload securityDeliveryPayload) *time.Time {
	var ttl time.Duration

	switch {
	case payload.Code != "":
		ttl = securityCodeTTL
	case payload.Continuation != "":
		ttl = securityContinuationTTL
	default:
		return nil
	}

	expiry := created.Add(ttl)

	return &expiry
}
