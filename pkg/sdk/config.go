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
	// ValidationModeRemote calls IAM /v1/tokens/verify for every token.
	ValidationModeRemote ValidationMode = "remote"
	// ValidationModeLocal verifies JWTs in-process using IAM's public JWKS.
	ValidationModeLocal ValidationMode = "local"
	// ValidationModeHybrid verifies locally first and falls back to remote verify
	// when local verification cannot authenticate the token.
	ValidationModeHybrid ValidationMode = "hybrid"
)

// AuthenticatorConfig is the high-level SDK wiring config for resource servers.
// It is intentionally tagged like internal/config service structs, so callers
// can load it with the same structconf/mapstructure pipeline.
type AuthenticatorConfig struct {
	Mode            ValidationMode `mapstructure:"mode" default:"remote" validate:"oneof=remote local hybrid"`
	BaseURL         string         `mapstructure:"base_url" default:"" validate:"omitempty,url"`
	Credential      string         `mapstructure:"credential" default:""`
	ProjectID       string         `mapstructure:"project_id" default:""`
	Environment     string         `mapstructure:"environment" default:"live"`
	Issuer          string         `mapstructure:"issuer" default:""`
	Audience        string         `mapstructure:"audience" default:""`
	JWKSURL         string         `mapstructure:"jwks_url" default:""`
	JWKSCacheTTLSec int            `mapstructure:"jwks_cache_ttl_sec" default:"300" validate:"min=1"`
	TokenType       string         `mapstructure:"token_type" default:"access" validate:"omitempty,oneof=access id_token"`
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
		return nil, errors.New("iam sdk: unsupported validation mode")
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
