package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	fileName        = "config"
	fileExtension   = "yaml"
	defaultFilePath = "./"
	filePathEnvName = "CONFIG_PATH"
)

type loadOptions struct {
	filename string
	fileExt  string
	filePath string
}

// LoadOption customises LoadConfig.
type LoadOption func(*loadOptions)

// WithFileName overrides the config file base name (default "config").
func WithFileName(name string) LoadOption { return func(o *loadOptions) { o.filename = name } }

// WithFileExt overrides the config file extension (default "yaml").
func WithFileExt(ext string) LoadOption { return func(o *loadOptions) { o.fileExt = ext } }

// WithFilePath overrides the config file directory (default "./" or $CONFIG_PATH).
func WithFilePath(path string) LoadOption { return func(o *loadOptions) { o.filePath = path } }

func newLoadOptions(opts ...LoadOption) *loadOptions {
	fp := os.Getenv(filePathEnvName)
	if fp == "" {
		fp = defaultFilePath
	}
	o := &loadOptions{filename: fileName, fileExt: fileExtension, filePath: fp}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Load loads, defaults, env-overlays and validates the IAM Config.
func Load(opts ...LoadOption) (*Config, error) { return LoadConfig[Config](opts...) }

// LoadConfig reads <name>.<ext> from the config path (ignoring a missing file),
// overlays a .env and the process environment (each mapstructure path bound to
// its UPPER_SNAKE_CASE env key), applies `default:` tags, unmarshals and then
// validates the struct. Mirrors the komeet configurator.
func LoadConfig[T any](opts ...LoadOption) (*T, error) {
	o := newLoadOptions(opts...)

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	v := viper.New()
	v.SetConfigName(o.filename)
	v.SetConfigType(o.fileExt)
	v.AddConfigPath(o.filePath)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := new(T)
	for envPath, envKey := range extractPaths(cfg) {
		if err := v.BindEnv(envPath, envKey); err != nil {
			return nil, err
		}
	}
	if err := defaults.Set(cfg); err != nil {
		return nil, fmt.Errorf("apply defaults: %w", err)
	}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

// extractPaths maps every leaf mapstructure path (e.g. infra.postgres.host) to
// its env key (INFRA_POSTGRES_HOST) for BindEnv.
func extractPaths[T any](i T) map[string]string {
	var paths []string
	extractMapstructurePaths(reflect.ValueOf(i), "", &paths)
	out := make(map[string]string, len(paths))
	for _, p := range paths {
		out[p] = strings.ToUpper(strings.ReplaceAll(p, ".", "_"))
	}
	return out
}

func extractMapstructurePaths(v reflect.Value, prefix string, paths *[]string) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			continue
		}
		if field.Type.Kind() == reflect.Struct {
			extractMapstructurePaths(v.Field(i), prefix+tag+".", paths)
		} else {
			*paths = append(*paths, prefix+tag)
		}
	}
}
