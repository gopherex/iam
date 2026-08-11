// Package config holds the IAM service configuration, loaded from a config file
// (config.yaml, path via CONFIG_PATH) overlaid with environment variables and
// validated. It follows the komeet/go-server-toolkit pattern: a nested
// Config{Infra, Service} of structs carrying mapstructure / default / validate
// tags, populated by the generic LoadConfig[T].
package config

import "fmt"

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
type Auth struct {
	DefaultEnvironment string `default:"live"    mapstructure:"default_environment" validate:"required"`
	AccessTTLSec       int    `default:"600"     mapstructure:"access_ttl_sec"      validate:"min=60"`
	RefreshTTLSec      int    `default:"2592000" mapstructure:"refresh_ttl_sec"     validate:"min=60"`
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
