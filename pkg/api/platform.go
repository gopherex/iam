// Code scaffolded for IAM handler groups.
//
// PlatformService is pure orchestration: it holds aggregate-port interfaces (deps) and
// nothing else. It embeds oas.UnimplementedHandler so any operation it does not
// override returns not-implemented, and panics on every v1.0.0 operation until
// written. Each port method is atomic in its adapter — services never open a
// transaction.

package api

import (
	"context"

	"github.com/gopherex/iam/internal/domain"
	"github.com/gopherex/iam/internal/oas"
)

// PlatformConfig serves unauthenticated bootstrap config for a client.
type PlatformConfig interface {
	PublicConfig(ctx context.Context, projectID, clientID string) (*domain.PublicConfig, error)
}

// PlatformCsrf issues and verifies CSRF tokens for cookie-mode clients.
type PlatformCsrf interface {
	IssueCsrfToken(ctx context.Context, clientID string) (*domain.PlatformCsrfToken, error)
	// VerifyCsrfToken validates a CSRF token previously issued to clientID. It is
	// reusable within its TTL (synchronizer-token pattern); returns
	// domain.ErrInvalidCsrf on a missing/expired/mismatched token.
	VerifyCsrfToken(ctx context.Context, clientID, token string) error
}

// PlatformDeps are the ports the Platform service orchestrates.
type PlatformDeps struct {
	Config PlatformConfig
	Csrf   PlatformCsrf
}

// PlatformService implements the PlatformHandler slice of oas.Handler.
type PlatformService struct {
	oas.UnimplementedHandler
	deps PlatformDeps
}

// NewPlatformService builds the Platform service from its dependencies.
func NewPlatformService(deps PlatformDeps) *PlatformService { return &PlatformService{deps: deps} }

var _ oas.Handler = (*PlatformService)(nil)

func (s *PlatformService) GetV1ConfigPublic(ctx context.Context, params oas.GetV1ConfigPublicParams) (*oas.PublicConfig, error) {
	cfg, err := s.deps.Config.PublicConfig(ctx, params.XClientID, params.XClientID)
	if err != nil {
		return nil, err
	}
	return oasPublicConfig(cfg), nil
}

func (s *PlatformService) GetV1Csrf(ctx context.Context, params oas.GetV1CsrfParams) (*oas.GetV1CsrfOK, error) {
	tok, err := s.deps.Csrf.IssueCsrfToken(ctx, params.XClientID)
	if err != nil {
		return nil, err
	}
	return &oas.GetV1CsrfOK{CsrfToken: oas.NewOptString(tok.Token)}, nil
}

func (s *PlatformService) GetV1Health(ctx context.Context) (*oas.GetV1HealthOK, error) {
	return &oas.GetV1HealthOK{Status: oas.NewOptString("ok")}, nil
}

func (s *PlatformService) GetV1HealthLive(ctx context.Context) (*oas.GetV1HealthLiveOK, error) {
	return &oas.GetV1HealthLiveOK{Status: oas.NewOptString("ok")}, nil
}

func (s *PlatformService) GetV1HealthReady(ctx context.Context) (*oas.GetV1HealthReadyOK, error) {
	return &oas.GetV1HealthReadyOK{Status: oas.NewOptString("ok")}, nil
}

// oasPublicConfig maps the domain bootstrap config to the oas wire type.
func oasPublicConfig(c *domain.PublicConfig) *oas.PublicConfig {
	r := &oas.PublicConfig{
		Project: oas.NewOptPublicConfigProject(oas.PublicConfigProject{Name: oas.NewOptString(c.ProjectName)}),
		Methods: c.Methods,
		Locales: c.Locales,
	}
	// Only set the default locale when present: an empty string fails the oas
	// locale pattern on response validation, which would reject the response.
	if c.DefaultLocale != "" {
		r.DefaultLocale = oas.NewOptString(c.DefaultLocale)
	}
	for _, p := range c.Providers {
		r.Providers = append(r.Providers, oas.PublicConfigProvidersItem{
			ID:   oas.NewOptString(p.ID),
			Name: oas.NewOptString(p.Name),
		})
	}
	if len(c.ConsentDocuments) > 0 {
		docs := make([]oas.ConsentDocument, 0, len(c.ConsentDocuments))
		for i := range c.ConsentDocuments {
			docs = append(docs, oasConsentDocument(&c.ConsentDocuments[i]))
		}
		r.Consents = oas.NewOptConsentConfig(oas.ConsentConfig{Documents: docs})
	}
	return r
}

func oasConsentDocument(d *domain.ConsentDocument) oas.ConsentDocument {
	out := oas.ConsentDocument{
		Key:     d.Key,
		Version: d.Version,
	}
	if d.Title != "" {
		out.Title = oas.NewOptString(d.Title)
	}
	if d.Body != "" {
		out.Body = oas.NewOptString(d.Body)
	}
	if d.Locale != "" {
		out.Locale = oas.NewOptString(d.Locale)
	}
	out.Required = oas.NewOptBool(d.Required)
	if d.URL != "" {
		out.URL = oas.NewOptNilString(d.URL)
	}
	return out
}
