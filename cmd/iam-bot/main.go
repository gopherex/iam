// iam-bot is the Telegram operator bot for IAM access requests: it pushes new
// pending requests to allowlisted operators and lets them approve/deny with
// inline buttons or mint invites, all through the IAM admin API.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gopherex/xlog"
	"github.com/gopherex/xprobe"

	"github.com/gopherex/iam/internal/build"
	"github.com/gopherex/iam/internal/iambot"
)

// probeReadHeaderTimeout bounds one probe request (mirrors the IAM server).
const probeReadHeaderTimeout = 5 * time.Second

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

	log := newLogger(cfg)
	log.Info("starting",
		xlog.String("service", build.ServiceName),
		xlog.String("version", build.Version),
		xlog.String("commit", build.Commit),
		xlog.String("iam", cfg.BaseURL),
		xlog.String("project", cfg.ProjectID),
		xlog.String("env", cfg.Environment),
		xlog.Int("operators", len(cfg.AllowedUsers)),
		xlog.String("poll_interval", cfg.PollInterval.String()),
	)

	iam, err := iambot.NewIAM(cfg.BaseURL, cfg.AdminToken, cfg.ProjectID, cfg.Environment)
	if err != nil {
		return fmt.Errorf("iam client: %w", err)
	}

	state, err := iambot.LoadState(cfg.StatePath)
	if err != nil {
		return fmt.Errorf("state: %w", err)
	}

	b, err := iambot.New(cfg, iam, state, log)
	if err != nil {
		return fmt.Errorf("bot: %w", err)
	}

	poller := iambot.NewPoller(b, iam, state)

	probeSrv := startProbes(cfg.HealthAddr, poller, log)

	defer func() {
		if probeSrv != nil {
			_ = probeSrv.Shutdown(context.Background())
		}
	}()

	go func() {
		if err := poller.Run(ctx, cfg.PollInterval); err != nil {
			log.Error("poller stopped", xlog.Error("err", err))
		}
	}()

	b.Start(ctx)
	log.Info("stopped")

	return nil
}

// newLogger builds the bot logger: JSON by default (container-friendly),
// console text for local runs; level via IAM_BOT_LOG_LEVEL.
func newLogger(cfg iambot.Config) *xlog.Logger {
	level, err := xlog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = xlog.InfoLevel
	}

	if cfg.LogFormat == "text" {
		return xlog.NewConsole(xlog.WithLevel(level), xlog.WithCaller(true))
	}

	return xlog.NewJSON(xlog.WithLevel(level))
}

// startProbes serves /healthz/liveness + /healthz/readiness on addr when
// configured; readiness follows the poller's HealthCheck. Returns nil (and
// serves nothing) when addr is empty.
func startProbes(addr string, poller *iambot.Poller, log *xlog.Logger) *http.Server {
	if addr == "" {
		return nil
	}

	live := xprobe.NewBool()
	live.Set(true)

	srv := &http.Server{
		Addr:              addr,
		Handler:           xprobe.Mux(xprobe.Liveness(live), xprobe.Readiness(xprobe.FromError(poller.HealthCheck))),
		ReadHeaderTimeout: probeReadHeaderTimeout,
	}

	go func() {
		log.Info("probes listening", xlog.String("addr", addr))

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("probe serve failed", xlog.Error("err", err))
		}
	}()

	return srv
}
