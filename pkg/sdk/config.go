package sdk

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

// ValidationMode selects how resource-server tokens are verified.
type ValidationMode string

const (
	// ValidationModeRemote calls IAM /v1/tokens/verify for every token. It is the
	// only mode that sees a REVOCATION: IAM checks its denylist there, so a token
	// revoked through /oauth2/revoke (or by the session ending) stops working
	// immediately.
	ValidationModeRemote ValidationMode = "remote"
	// ValidationModeLocal verifies JWTs in-process using IAM's public JWKS.
	//
	// It never asks IAM anything, which is the point — and the cost: a revoked
	// token still verifies until it expires, because nothing tells this process
	// that it was revoked. Choose it when that window (the project's access_ttl,
	// 10 minutes by default) is acceptable, and shorten access_ttl if it is not.
	ValidationModeLocal ValidationMode = "local"
	// ValidationModeHybrid verifies locally first and falls back to remote verify
	// when local verification cannot authenticate the token. Local verification
	// SUCCEEDS for a revoked token, so this mode inherits the local window rather
	// than the remote guarantee.
	ValidationModeHybrid ValidationMode = "hybrid"
)

// ErrUnsupportedValidationMode is returned by NewAuthenticator for an
// AuthenticatorConfig.Mode that names none of the ValidationMode constants.
var ErrUnsupportedValidationMode = errors.New("iam sdk: unsupported validation mode")

// AuthenticatorConfig is the high-level SDK wiring config for resource servers.
// It is intentionally tagged like internal/config service structs, so callers
// can load it with the same structconf/mapstructure pipeline.
type AuthenticatorConfig struct {
	Mode            ValidationMode `default:"remote" mapstructure:"mode"               validate:"oneof=remote local hybrid"` //nolint:lll
	BaseURL         string         `default:""       mapstructure:"base_url"           validate:"omitempty,url"`
	Credential      string         `default:""       mapstructure:"credential"`
	ProjectID       string         `default:""       mapstructure:"project_id"`
	Environment     string         `default:"live"   mapstructure:"environment"`
	Issuer          string         `default:""       mapstructure:"issuer"`
	Audience        string         `default:""       mapstructure:"audience"`
	JWKSURL         string         `default:""       mapstructure:"jwks_url"`
	JWKSCacheTTLSec int            `default:"300"    mapstructure:"jwks_cache_ttl_sec" validate:"min=1"`
	TokenType       string         `default:"access" mapstructure:"token_type"         validate:"omitempty,oneof=access id_token"` //nolint:lll
	HTTPClient      *http.Client   `mapstructure:"-" validate:"-"`
}

// NewAuthenticator builds a remote, local, or hybrid Authenticator from config.
func NewAuthenticator(config AuthenticatorConfig) (Authenticator, error) {
	mode := config.mode()
	switch mode {
	case ValidationModeRemote:
		return NewVerifier(config.remoteConfig(), WithAudience(config.Audience))
	case ValidationModeLocal:
		return NewLocalVerifier(config.localConfig())
	case ValidationModeHybrid:
		remote, err := NewVerifier(config.remoteConfig(), WithAudience(config.Audience))
		if err != nil {
			return nil, err
		}

		local, err := NewLocalVerifier(config.localConfig())
		if err != nil {
			return nil, err
		}

		return NewHybridVerifier(local, remote), nil
	default:
		return nil, ErrUnsupportedValidationMode
	}
}

func (c AuthenticatorConfig) mode() ValidationMode {
	mode := ValidationMode(strings.TrimSpace(string(c.Mode)))
	if mode == "" {
		return ValidationModeRemote
	}

	return mode
}

func (c AuthenticatorConfig) remoteConfig() Config {
	return Config{
		BaseURL:    c.BaseURL,
		Credential: c.Credential,
		HTTPClient: c.HTTPClient,
	}
}

func (c AuthenticatorConfig) localConfig() LocalConfig {
	return LocalConfig{
		BaseURL:     c.BaseURL,
		JWKSURL:     c.JWKSURL,
		ProjectID:   c.ProjectID,
		Environment: c.Environment,
		Issuer:      c.Issuer,
		Audience:    c.Audience,
		TokenType:   c.TokenType,
		CacheTTL:    time.Duration(c.JWKSCacheTTLSec) * time.Second,
		HTTPClient:  c.HTTPClient,
	}
}
