// Command iam runs the IAM server: it loads configuration, connects to Postgres,
// applies migrations, assembles the ogen-generated HTTP API (pkg/api) over the
// Postgres adapters, exposes liveness/readiness probes, runs the transactional
// outbox relay, and shuts down gracefully on SIGINT/SIGTERM.
//
// Wiring stack (all first-party gopherex libs):
//   - config:   github.com/gopherex/xconf/pkg/structconf  (typed Config load)
//   - logger:   github.com/gopherex/xlog
//   - probes:   github.com/gopherex/xprobe
//   - shutdown: github.com/gopherex/xshutdown
//   - outbox:   github.com/gopherex/pg-outbox             (mock publisher for now)
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	outbox "github.com/gopherex/pg-outbox"
	"github.com/gopherex/xconf/pkg/structconf"
	"github.com/gopherex/xlog"
	"github.com/gopherex/xprobe"
	"github.com/gopherex/xshutdown"

	"github.com/gopherex/iam/internal/config"
	"github.com/gopherex/iam/internal/infrastructure/postgres"
	"github.com/gopherex/iam/internal/oas"
	"github.com/gopherex/iam/pkg/api"
)

func main() {
	if err := run(); err != nil {
		// run already logged the cause; exit non-zero for the supervisor.
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "", "path to config file (yaml/json/toml); overrides CONFIG_PATH discovery")
	flag.Parse()

	// ----- config -----
	loadOpts := []structconf.Option{
		structconf.WithConfigPath("config"), // $CONFIG_PATH/config.{yaml,...}
		structconf.WithDotEnv(),             // optional .env in cwd
		structconf.WithEnvPrefix("IAM"),     // IAM_INFRA_POSTGRES_HOST, ...
	}
	if *configPath != "" {
		loadOpts = append(loadOpts, structconf.WithFile(*configPath))
	}
	cfg, err := structconf.Load[config.Config](loadOpts...)
	if err != nil {
		slog.Error("config load failed", "err", err)
		return err
	}

	// ----- logger -----
	log := newLogger(cfg.Service.Logger).AppendName("iam")
	log.Info("starting", xlog.String("addr", cfg.Service.HTTP.Addr))

	ctx := context.Background()

	// ----- postgres -----
	db, err := postgres.Connect(ctx, cfg.Infra.Postgres.DSN())
	if err != nil {
		log.Error("postgres connect failed", xlog.Error("err", err))
		return err
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		log.Error("migrate failed", xlog.Error("err", err))
		return err
	}
	// Outbox owns its own table (outbox_messages); apply its migrations too.
	for _, stmt := range outbox.Migrations() {
		if _, err := db.Pool.Exec(ctx, stmt); err != nil {
			log.Error("outbox migrate failed", xlog.Error("err", err))
			return err
		}
	}
	log.Info("migrations applied")

	// ----- API handler (12 feature groups over Postgres adapters) -----
	handler := buildHandler(db)
	auth := postgres.NewAuthenticator(db, cfg.Service.Auth.MasterKey)
	srv, err := oas.NewServer(handler, api.NewSecurityHandler(auth), oas.WithErrorHandler(api.ErrorHandler))
	if err != nil {
		log.Error("server build failed", xlog.Error("err", err))
		return err
	}

	// ----- probes (mounted under /healthz/* alongside the API) -----
	live := xprobe.NewBool()
	live.Set(true)
	probeMux := xprobe.Mux(
		xprobe.Liveness(live),
		xprobe.Readiness(xprobe.FromError(db.Ping)),
	)

	root := http.NewServeMux()
	root.Handle("/healthz/", probeMux)
	root.Handle("/", srv)

	httpSrv := &http.Server{
		Addr:         cfg.Service.HTTP.Addr,
		Handler:      root,
		ReadTimeout:  time.Duration(cfg.Service.HTTP.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Service.HTTP.WriteTimeoutSec) * time.Second,
	}

	// ----- outbox relay (mock publisher: log-only) -----
	// enqueue + relay both run on the pool; once outbox emission points are wired
	// in the adapters, swap the enqueue executor to db.TxDB so inserts join the
	// caller's transaction.
	ob, err := outbox.New(db.Pool, db.Pool, &mockPublisher{log: log.AppendName("outbox")},
		outbox.WithLogger(slog.Default()),
		outbox.WithPollInterval(time.Second),
	)
	if err != nil {
		log.Error("outbox init failed", xlog.Error("err", err))
		return err
	}

	// ----- shutdown orchestration -----
	sd := xshutdown.New(ctx,
		xshutdown.WithTimeout(time.Duration(cfg.Service.HTTP.ShutdownSec)*time.Second),
		xshutdown.WithErrorHandler(func(err error) { log.Error("shutdown error", xlog.Error("err", err)) }),
	)
	// Cleanup runs in registration order: stop serving, flip liveness, close DB.
	sd.RegisterFnErr(
		func(ctx context.Context) error { return httpSrv.Shutdown(ctx) },
		func(ctx context.Context) error { live.Set(false); return nil },
	)
	// Background workers cancel with the shutdown context.
	sd.Go(func(ctx context.Context) {
		if err := ob.Run(ctx); err != nil && ctx.Err() == nil {
			log.Error("outbox relay stopped", xlog.Error("err", err))
		}
	})
	sd.Go(func(context.Context) {
		log.Info("listening", xlog.String("addr", cfg.Service.HTTP.Addr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http serve failed", xlog.Error("err", err))
		}
	})

	// Block until SIGINT/SIGTERM, then run the registered cleanups.
	if err := sd.Run(); err != nil {
		log.Error("shutdown completed with errors", xlog.Error("err", err))
		return err
	}
	log.Info("stopped")
	return nil
}

// newLogger builds the application logger from config (json or console).
func newLogger(c config.Logger) *xlog.Logger {
	level, err := xlog.ParseLevel(c.Level)
	if err != nil {
		level = xlog.InfoLevel
	}
	opts := []xlog.Option{xlog.WithLevel(level), xlog.WithCaller(true)}
	if c.Format == "text" {
		return xlog.NewConsole(opts...)
	}
	return xlog.NewJSON(opts...)
}

// buildHandler assembles the full IAM handler from the Postgres adapters, one
// option per feature group.
func buildHandler(db *postgres.DB) *api.Service {
	platform := postgres.NewPgPlatform(db) // implements PlatformConfig + PlatformCsrf
	coreAuth := postgres.NewPgCoreAuth(db) // implements CoreAuthAccounts + CoreAuthTokens

	return api.New(
		api.WithPlatform(api.NewPlatformService(api.PlatformDeps{
			Config: platform,
			Csrf:   platform,
		})),
		api.WithCoreAuth(api.NewCoreAuthService(api.CoreAuthDeps{
			Accounts: coreAuth,
			Tokens:   coreAuth,
		})),
		api.WithPasswordless(api.NewPasswordlessService(api.PasswordlessDeps{
			Accounts: postgres.NewPgPasswordlessAccounts(db),
		})),
		api.WithOAuthSocial(api.NewOAuthSocialService(api.OAuthSocialDeps{
			Accounts: postgres.NewPgOAuthSocial(db),
		})),
		api.WithWebAuthn(api.NewWebAuthnService(api.WebAuthnDeps{
			Accounts: postgres.NewPgWebAuthnAccounts(db),
		})),
		api.WithMFA(api.NewMFAService(api.MFADeps{
			Accounts: postgres.NewPgMFAAccounts(db),
		})),
		api.WithAccount(api.NewAccountService(api.AccountDeps{
			Accounts: postgres.NewPgAccountStore(db),
		})),
		api.WithMachineIdentity(api.NewMachineIdentityService(api.MachineIdentityDeps{
			Keys: postgres.NewPgMachineIdentities(db),
		})),
		api.WithFederation(api.NewFederationService(api.FederationDeps{
			Connections: postgres.NewPgFederationConnections(db),
			Runtime:     postgres.NewPgFederationRuntime(db),
			Scim:        postgres.NewPgFederationScim(db),
		})),
		api.WithOIDCProvider(api.NewOIDCProviderService(api.OIDCProviderDeps{
			Grants: postgres.NewPgOIDCGrants(db),
		})),
		api.WithAdmin(api.NewAdminService(api.AdminDeps{
			Users:           postgres.NewPgAdminUsers(db),
			Apps:            postgres.NewPgAdminApps(db),
			ServiceAccounts: postgres.NewPgAdminServiceAccounts(db),
			APIKeys:         postgres.NewPgAdminAPIKeys(db),
			Connections:     postgres.NewPgAdminConnections(db),
			Config:          postgres.NewPgAdminConfig(db),
			Keys:            postgres.NewPgAdminKeys(db),
			AccessRequests:  postgres.NewPgAdminAccessRequests(db),
		})),
		api.WithOperator(api.NewOperatorService(api.OperatorDeps{
			Projects: postgres.NewPgOperator(db),
		})),
	)
}

// mockPublisher is a placeholder outbox transport: it logs each batch and acks
// it. Replace with a real broker publisher (NATS/Kafka/…) when outbox emission
// points are wired in the adapters.
type mockPublisher struct{ log *xlog.Logger }

func (p *mockPublisher) Publish(_ context.Context, msgs []outbox.Message) error {
	for _, m := range msgs {
		p.log.Info("would publish",
			xlog.String("id", m.ID),
			xlog.String("topic", m.Topic),
			xlog.String("type", m.MessageType),
		)
	}
	return nil
}
