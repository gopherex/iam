// Package iambot is the Telegram operator bot: it watches pending access
// requests through the IAM admin API and lets allowlisted Telegram users
// approve/deny them (or mint invites) without opening the admin panel.
package iambot

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// errConfig marks configuration problems (missing/invalid environment).
var errConfig = errors.New("iambot: config")

// defaultPollInterval is the pending-list poll cadence.
const defaultPollInterval = 30 * time.Second

// Config is the bot's environment-driven configuration.
type Config struct {
	// TelegramToken is the bot token from @BotFather.
	TelegramToken string
	// BaseURL is the IAM service base URL (e.g. https://iam.example.com).
	BaseURL string
	// ProjectID scopes every admin call to one project.
	ProjectID string
	// AdminToken is a project-admin bearer token for the admin API.
	AdminToken string
	// Environment is the optional X-Environment scope (live/test/…).
	Environment string
	// AllowedUsers is the allowlist of Telegram user ids the bot talks to.
	AllowedUsers []int64
	// StatePath is where the seen-request state is persisted.
	StatePath string
	// PollInterval is how often pending access requests are re-listed.
	PollInterval time.Duration
}

// FromEnv builds the Config from IAM_BOT_* environment variables.
func FromEnv() (Config, error) {
	cfg := Config{
		TelegramToken: os.Getenv("IAM_BOT_TELEGRAM_TOKEN"),
		BaseURL:       strings.TrimRight(os.Getenv("IAM_BOT_BASE_URL"), "/"),
		ProjectID:     os.Getenv("IAM_BOT_PROJECT_ID"),
		AdminToken:    os.Getenv("IAM_BOT_ADMIN_TOKEN"),
		Environment:   os.Getenv("IAM_BOT_ENVIRONMENT"),
		StatePath:     os.Getenv("IAM_BOT_STATE_PATH"),
		PollInterval:  defaultPollInterval,
	}

	if cfg.StatePath == "" {
		cfg.StatePath = "iam-bot-state.json"
	}

	if raw := os.Getenv("IAM_BOT_POLL_INTERVAL"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("%w: IAM_BOT_POLL_INTERVAL: %w", errConfig, err)
		}

		cfg.PollInterval = d
	}

	for _, raw := range strings.Split(os.Getenv("IAM_BOT_ALLOWED_USERS"), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("%w: IAM_BOT_ALLOWED_USERS: %q is not a telegram user id",
				errConfig, raw)
		}

		cfg.AllowedUsers = append(cfg.AllowedUsers, id)
	}

	switch {
	case cfg.TelegramToken == "":
		return Config{}, fmt.Errorf("%w: IAM_BOT_TELEGRAM_TOKEN is required", errConfig)
	case cfg.BaseURL == "":
		return Config{}, fmt.Errorf("%w: IAM_BOT_BASE_URL is required", errConfig)
	case cfg.ProjectID == "":
		return Config{}, fmt.Errorf("%w: IAM_BOT_PROJECT_ID is required", errConfig)
	case cfg.AdminToken == "":
		return Config{}, fmt.Errorf("%w: IAM_BOT_ADMIN_TOKEN is required", errConfig)
	case len(cfg.AllowedUsers) == 0:
		return Config{}, fmt.Errorf("%w: IAM_BOT_ALLOWED_USERS is required (comma-separated telegram user ids)", errConfig) //nolint:lll
	case cfg.PollInterval < time.Second:
		return Config{}, fmt.Errorf("%w: IAM_BOT_POLL_INTERVAL must be at least 1s", errConfig)
	}

	return cfg, nil
}

// Allows reports whether a Telegram user id may operate the bot.
func (c Config) Allows(userID int64) bool {
	for _, id := range c.AllowedUsers {
		if id == userID {
			return true
		}
	}

	return false
}
