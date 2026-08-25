// Package config holds the IAM service configuration, loaded from a config file
// (config.yaml, path via CONFIG_PATH) overlaid with environment variables and
// validated. It follows the komeet/go-server-toolkit pattern: a nested
// Config{Infra, Service} of structs carrying mapstructure / default / validate
// tags, populated by the generic LoadConfig[T].
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gopherex/iam/internal/domain"
)

// ErrPublicURLRequired is returned by Normalize when service.http.public_url is
// unset. It has no safe default: the value is the OIDC issuer every relying
// party pins, and it cannot be inferred from request headers without letting a
// client choose the issuer.
var ErrPublicURLRequired = errors.New(
	"service.http.public_url is required: set the absolute base URL this deployment is reachable at " +
		"(e.g. https://auth.example.com) via SERVICE_HTTP_PUBLIC_URL",
)

// Postgres is the connection config for the IAM store.
type Postgres struct {
	Host     string `default:"localhost" mapstructure:"host"      validate:"required,hostname|ip"`
	Port     int    `default:"5432"      mapstructure:"port"      validate:"required,min=1,max=65535"`
	Username string `default:"iam"       mapstructure:"username"  validate:"required"`
	Password string `default:"iam"       mapstructure:"password"  validate:"required"`
	Database string `default:"iam"       mapstructure:"database"  validate:"required"`
	SSLMode  string `default:"require"   mapstructure:"sslmode"   validate:"oneof=disable require verify-ca verify-full"`
	LogLevel string `default:"info"      mapstructure:"log_level" validate:"oneof=debug info warn error"`
}

// DSN renders the libpq/pgx connection string.
func (c *Postgres) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// HTTP is the inbound HTTP server config.
type HTTP struct {
	Addr            string `default:":8080" mapstructure:"addr"              validate:"required"`
	ReadTimeoutSec  int    `default:"15"    mapstructure:"read_timeout_sec"  validate:"min=1"`
	WriteTimeoutSec int    `default:"30"    mapstructure:"write_timeout_sec" validate:"min=1"`
	ShutdownSec     int    `default:"15"    mapstructure:"shutdown_sec"      validate:"min=1"`
	// ProbeAddr is the address for the liveness/readiness probe server. When set
	// to a different address than Addr, probes are served on their own listener
	// (a k8s sidecar port not exposed publicly); when empty or equal to Addr they
	// are mounted under /healthz/ on the main server.
	ProbeAddr string `default:":8081" mapstructure:"probe_addr"`
	// TrustedProxies is the list of CIDR ranges (or bare IPs) of reverse
	// proxies / load balancers in front of the service. The client IP is taken
	// from X-Forwarded-For / X-Real-IP ONLY when the connecting peer is in this
	// set; otherwise the real TCP peer (RemoteAddr) is used. Empty (default)
	// means trust no proxy headers — required so a client cannot spoof its IP to
	// bypass IP-keyed rate limiting. Set to the LB subnet in proxied deploys
	// (e.g. "10.0.0.0/8"). Env: SERVICE_HTTP_TRUSTED_PROXIES (comma-separated).
	TrustedProxies []string `mapstructure:"trusted_proxies"`
	// PublicURL is the absolute base URL this deployment is reachable at
	// (scheme://host[:port][/prefix], no trailing slash). It is the sole source
	// of the OIDC issuer and of every absolute URL the service advertises, so it
	// is REQUIRED: the service refuses to start without it. It is deliberately
	// NOT derived from Host / X-Forwarded-* — a request header cannot be allowed
	// to decide the issuer clients pin. Env: SERVICE_HTTP_PUBLIC_URL.
	PublicURL string `default:"" mapstructure:"public_url"`
}

// Logger is the structured-logging config.
type Logger struct {
	Level  string `default:"info" mapstructure:"level"  validate:"oneof=debug info warn error"`
	Format string `default:"json" mapstructure:"format" validate:"oneof=json text"`
}

// CORS is the browser cross-origin policy for runtime endpoints.
type CORS struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// Auth holds IAM token/issuer defaults applied when an environment does not
// override them.
//
// Token lifetimes and the acting environment are NOT configured here: access /
// refresh TTLs come from each project's session_policy config doc (env-scoped),
// and the runtime environment is selected per request by the X-Environment
// header. Server-level access_ttl_sec / refresh_ttl_sec / default_environment
// keys were removed because they were silently ignored (a footgun for operators
// who set them expecting an effect).
type Auth struct {
	// MasterKey is the platform operator (master-key) credential. When empty the
	// masterKey security scheme rejects every request — operator endpoints are
	// disabled until a key is configured (set via MASTER_KEY / service.auth.master_key).
	MasterKey string `default:"" mapstructure:"master_key"`
	// EncryptionKey is the base64-encoded 32-byte AES-256 key that encrypts
	// reversible secrets at rest (signing-key PEMs, TOTP secrets). Empty disables
	// at-rest encryption (passthrough). Set via SERVICE_AUTH_ENCRYPTION_KEY.
	EncryptionKey string `default:"" mapstructure:"encryption_key" validate:"omitempty,base64"`
	// SeedRoot, when true, ensures a root project exists on startup so the
	// operator (master key) has something to manage. Development convenience;
	// set via SERVICE_AUTH_SEED_ROOT.
	SeedRoot bool `default:"false" mapstructure:"seed_root"`
}

// Infrastructure is the external-dependency config (datastores, …).
type Infrastructure struct {
	Postgres Postgres `mapstructure:"postgres"`
}

// Service is the application-layer config (transport, logging, auth policy).
type Service struct {
	HTTP   HTTP   `mapstructure:"http"`
	Logger Logger `mapstructure:"logger"`
	CORS   CORS   `mapstructure:"cors"`
	Auth   Auth   `mapstructure:"auth"`
}

// Config is the full IAM service configuration.
type Config struct {
	Infra   Infrastructure `mapstructure:"infra"`
	Service Service        `mapstructure:"service"`
}

// Normalize canonicalizes and validates the values the loader cannot express as
// struct tags. It is called once, immediately after loading, and a returned
// error aborts startup.
//
// Today that is service.http.public_url: it is required, must be an absolute
// http(s) URL, and is stored without its trailing slash so callers can
// concatenate paths onto it directly.
func (c *Config) Normalize() error {
	pub := strings.TrimRight(strings.TrimSpace(c.Service.HTTP.PublicURL), "/")
	if pub == "" {
		return ErrPublicURLRequired
	}

	if err := domain.ValidateAbsoluteHTTPURL("service.http.public_url", pub); err != nil {
		return err
	}

	c.Service.HTTP.PublicURL = pub

	return nil
}
