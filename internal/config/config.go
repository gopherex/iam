// Package config holds the IAM service configuration, loaded from a config file
// (config.yaml, path via CONFIG_PATH) overlaid with environment variables and
// validated. It follows the komeet/go-server-toolkit pattern: a nested
// Config{Infra, Service} of structs carrying mapstructure / default / validate
// tags, populated by the generic LoadConfig[T].
package config

import "fmt"

// Postgres is the connection config for the IAM store.
type Postgres struct {
	Host     string `mapstructure:"host" default:"localhost" validate:"required,hostname|ip"`
	Port     int    `mapstructure:"port" default:"5432" validate:"required,min=1,max=65535"`
	Username string `mapstructure:"username" default:"iam" validate:"required"`
	Password string `mapstructure:"password" default:"iam" validate:"required"`
	Database string `mapstructure:"database" default:"iam" validate:"required"`
	SSLMode  string `mapstructure:"sslmode" default:"disable" validate:"oneof=disable require verify-ca verify-full"`
	LogLevel string `mapstructure:"log_level" default:"info" validate:"oneof=debug info warn error"`
}

// DSN renders the libpq/pgx connection string.
func (c *Postgres) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// HTTP is the inbound HTTP server config.
type HTTP struct {
	Addr            string `mapstructure:"addr" default:":8080" validate:"required"`
	ReadTimeoutSec  int    `mapstructure:"read_timeout_sec" default:"15" validate:"min=1"`
	WriteTimeoutSec int    `mapstructure:"write_timeout_sec" default:"30" validate:"min=1"`
	ShutdownSec     int    `mapstructure:"shutdown_sec" default:"15" validate:"min=1"`
}

// Logger is the structured-logging config.
type Logger struct {
	Level  string `mapstructure:"level" default:"info" validate:"oneof=debug info warn error"`
	Format string `mapstructure:"format" default:"json" validate:"oneof=json text"`
}

// CORS is the browser cross-origin policy for runtime endpoints.
type CORS struct {
	AllowedOrigins []string `mapstructure:"allowed_origins" default:"[\"*\"]"`
}

// Auth holds IAM token/issuer defaults applied when an environment does not
// override them.
type Auth struct {
	DefaultEnvironment string `mapstructure:"default_environment" default:"live" validate:"required"`
	AccessTTLSec       int    `mapstructure:"access_ttl_sec" default:"1800" validate:"min=60"`
	RefreshTTLSec      int    `mapstructure:"refresh_ttl_sec" default:"2592000" validate:"min=60"`
	// MasterKey is the platform operator (master-key) credential. When empty the
	// masterKey security scheme rejects every request — operator endpoints are
	// disabled until a key is configured (set via MASTER_KEY / service.auth.master_key).
	MasterKey string `mapstructure:"master_key" default:""`
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
