// iam-bot is the Telegram operator bot for IAM access requests: it pushes new
// pending requests to allowlisted operators and lets them approve/deny with
// inline buttons or mint invites, all through the IAM admin API.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gopherex/iam/internal/iambot"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := iambot.FromEnv()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	iam, err := iambot.NewIAM(cfg.BaseURL, cfg.AdminToken, cfg.ProjectID, cfg.Environment)
	if err != nil {
		return fmt.Errorf("iam client: %w", err)
	}

	state, err := iambot.LoadState(cfg.StatePath)
	if err != nil {
		return fmt.Errorf("state: %w", err)
	}

	b, err := iambot.New(cfg, iam, state)
	if err != nil {
		return fmt.Errorf("bot: %w", err)
	}

	go func() {
		if err := iambot.NewPoller(b, iam, state).Run(ctx, cfg.PollInterval); err != nil {
			fmt.Fprintln(os.Stderr, "poller:", err)
		}
	}()

	fmt.Fprintln(os.Stderr, "iam-bot: polling", cfg.BaseURL, "project", cfg.ProjectID, "every", cfg.PollInterval)

	b.Start(ctx)

	return nil
}
