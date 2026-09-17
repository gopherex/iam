package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/infrastructure/postgres"
)

func (p *Publisher) publishSecurityDelivery(ctx context.Context, event eventEnvelope) error {
	scope := domain.SecurityScope{ProjectID: event.ProjectID, Environment: event.Environment}
	store := postgres.NewPgSecurity(p.db, nil)
	id := stringValue(event.Payload, "delivery_id")

	job, err := store.DeliveryJob(ctx, scope, id)
	if err != nil {
		return err
	}

	if job.Status == "accepted" || job.Status == "expired" {
		return nil
	}

	fail := func(code string, cause error) error {
		if err := store.DeliveryResult(ctx, scope, id, "failed", code); err != nil {
			return errors.Join(cause, err)
		}

		return cause
	}

	if job.ExpiresAt != nil && !time.Now().Before(*job.ExpiresAt) {
		return store.DeliveryResult(ctx, scope, id, "expired", "proof_expired")
	}

	if job.Channel == "sms" {
		return p.publishSecuritySMS(ctx, event, store, scope, id, job, fail)
	}

	provider, skip, err := p.resolveSMTPProviderOrSkip(ctx, event)
	if err != nil {
		return fail("provider_error", err)
	}

	if skip {
		return store.DeliveryResult(ctx, scope, id, "blocked", "provider_not_configured")
	}

	locale := p.resolveLocale(ctx, event, job.Locale)

	rendered, err := p.renderTemplate(
		ctx,
		event.ProjectID,
		emailJob{
			TemplateID: job.Template,
			To:         job.To,
			Locale:     locale,
			Data:       map[string]any{"code": job.Code, "link": job.Link, "summary": job.Summary},
		})
	if err != nil {
		return fail("template_error", err)
	}

	if err := provider.send(ctx, job.To, rendered); err != nil {
		return fail("smtp_error", fmt.Errorf("security SMTP delivery: %w", err))
	}

	return store.DeliveryResult(ctx, scope, id, "accepted", "")
}

func (p *Publisher) publishSecuritySMS(ctx context.Context, event eventEnvelope,
	store interface {
		DeliveryResult(ctx context.Context, scope domain.SecurityScope, id, status, code string) error
	},
	scope domain.SecurityScope, id string, job *postgres.SecurityDeliveryJob, fail func(string, error) error,
) error {
	provider, err := p.smsProvider(ctx, event.ProjectID)
	if errors.Is(err, errNoSMSProvider) {
		return store.DeliveryResult(ctx, scope, id, "blocked", "provider_not_configured")
	}

	if err != nil {
		return fail("provider_error", err)
	}

	body := "Account security: " + job.Summary + " " + job.Link
	if job.Code != "" {
		body = "Account security verification code: " + job.Code
	}

	if err := provider.send(ctx, job.To, body); err != nil {
		return fail("sms_error", err)
	}

	return store.DeliveryResult(ctx, scope, id, "accepted", "")
}
