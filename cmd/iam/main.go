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
//   - tracing:  github.com/gopherex/xtrace
//   - outbox:   github.com/gopherex/pg-outbox             (email delivery + event log)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	outbox "github.com/gopherex/pg-outbox"
	"github.com/gopherex/xconf/pkg/structconf"
	"github.com/gopherex/xlog"
	"github.com/gopherex/xprobe"
	"github.com/gopherex/xshutdown"
	xlogtrace "github.com/gopherex/xtrace/contrib/libs/xlog"
	xtracesdk "github.com/gopherex/xtrace/contrib/sdk"
	logglobal "go.opentelemetry.io/otel/log/global"

	"github.com/gopherex/iam/internal/build"
	"github.com/gopherex/iam/internal/config"
	"github.com/gopherex/iam/internal/infrastructure/notifications"
	"github.com/gopherex/iam/internal/infrastructure/postgres"
	"github.com/gopherex/iam/internal/oas"
	"github.com/gopherex/iam/pkg/api"
	"github.com/gopherex/iam/web"
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

	ctx := context.Background()

	// ----- telemetry -----
	telemetryShutdown, err := xtracesdk.Setup(ctx,
		xtracesdk.WithService(build.ServiceName),
		xtracesdk.WithVersion(build.Version),
		xtracesdk.WithInstanceID(build.InstanceID),
	)
	if err != nil {
		slog.Error("telemetry setup failed", "err", err)
		return err
	}
	defer func() {
		if telemetryShutdown != nil {
			_ = telemetryShutdown(context.Background())
		}
	}()

	// ----- logger -----
	log := newLogger(cfg.Service.Logger).AppendName(build.ServiceName).With(buildFields()...)
	if err := xtracesdk.StartHostRuntime(); err != nil {
		log.Error("telemetry runtime instrumentation failed", xlog.Error("err", err))
		return err
	}
	log.Info("starting", xlog.String("addr", cfg.Service.HTTP.Addr))

	// ----- postgres -----
	db, err := postgres.Connect(ctx, cfg.Infra.Postgres.DSN(),
		postgres.WithLogger(log.AppendName("postgres")),
		postgres.WithQueryLogLevel(cfg.Infra.Postgres.LogLevel),
		postgres.WithMetrics(true),
	)
	if err != nil {
		log.Error("postgres connect failed", xlog.Error("err", err))
		return err
	}
	defer db.Close()

	// At-rest secret encryption (signing-key PEMs, TOTP secrets).
	cph, err := postgres.NewCipher(cfg.Service.Auth.EncryptionKey)
	if err != nil {
		log.Error("encryption key invalid", xlog.Error("err", err))
		return err
	}
	db.UseCipher(cph)
	if cfg.Service.Auth.EncryptionKey == "" {
		log.Error("secrets-at-rest encryption is DISABLED — set service.auth.encryption_key (base64 32-byte AES-256 key) before running in production")
		return fmt.Errorf("service.auth.encryption_key is required")
	}

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

	// ----- outbox (email publisher; enqueue joins the caller tx via db.TxDB) -----
	ob, err := outbox.New(db.Pool, db.TxDB, notifications.NewPublisher(db, log.AppendName("outbox")),
		outbox.WithInstanceID(build.InstanceID),
		outbox.WithLogger(buildSlogLogger()),
		outbox.WithPollInterval(time.Second),
	)
	if err != nil {
		log.Error("outbox init failed", xlog.Error("err", err))
		return err
	}
	emitter := postgres.NewOutboxEmitter(ob)

	// ----- optional root seed (operator gets a project to manage) -----
	if cfg.Service.Auth.SeedRoot {
		if err := seedRoot(ctx, db, emitter, log); err != nil {
			log.Error("seed root failed", xlog.Error("err", err))
			return err
		}
	}

	// ----- API handler (12 feature groups over Postgres adapters) -----
	handler := buildHandler(db, emitter)
	auth := postgres.NewAuthenticator(db, cfg.Service.Auth.MasterKey)
	srv, err := oas.NewServer(handler, api.NewSecurityHandler(auth), oas.WithErrorHandler(api.ErrorHandler))
	if err != nil {
		log.Error("server build failed", xlog.Error("err", err))
		return err
	}

	// ----- probes -----
	live := xprobe.NewBool()
	live.Set(true)
	probeMux := xprobe.Mux(
		xprobe.Liveness(live),
		xprobe.Readiness(xprobe.FromError(db.Ping)),
	)

	// API request pipeline (outermost first): X-Environment -> ctx; CSRF for
	// cookie-mode requests (evaluated before cookie auth, while there is no
	// Authorization header); cookie auth promotes the session cookie to a Bearer
	// header; then the generated API server.
	apiPipeline := api.EnvironmentMiddleware(
		api.CSRFMiddleware(postgres.NewPgPlatform(db))(
			api.CookieAuthMiddleware(srv)))
	apiPipeline = api.CORSMiddleware(cfg.Service.CORS.AllowedOrigins)(apiPipeline)
	apiPipeline = api.SecurityHeaders(apiPipeline)
	apiPipeline = api.RateLimitMiddleware(apiPipeline)

	root := http.NewServeMux()
	// API namespaces go to the generated server; everything else is the embedded
	// admin SPA (a stub until the binary is built with `make build` / -tags embed).
	for _, prefix := range []string{"/v1/", "/mgmt/", "/oauth2/", "/p/"} {
		root.Handle(prefix, apiPipeline)
	}
	root.Handle("/", api.SecurityHeaders(web.Handler()))

	// Probes get their own listener when ProbeAddr differs from the API port (a
	// k8s sidecar port not exposed publicly); otherwise they mount on the API
	// server under /healthz/.
	probeAddr := cfg.Service.HTTP.ProbeAddr
	separateProbes := probeAddr != "" && probeAddr != cfg.Service.HTTP.Addr
	if !separateProbes {
		root.Handle("/healthz/", probeMux)
	}

	httpSrv := &http.Server{
		Addr:           cfg.Service.HTTP.Addr,
		Handler:        http.MaxBytesHandler(root, 1<<20),
		ReadTimeout:    time.Duration(cfg.Service.HTTP.ReadTimeoutSec) * time.Second,
		WriteTimeout:   time.Duration(cfg.Service.HTTP.WriteTimeoutSec) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	var probeSrv *http.Server
	if separateProbes {
		probeSrv = &http.Server{Addr: probeAddr, Handler: probeMux}
	}

	// ----- shutdown orchestration -----
	sd := xshutdown.New(ctx,
		xshutdown.WithTimeout(time.Duration(cfg.Service.HTTP.ShutdownSec)*time.Second),
		xshutdown.WithErrorHandler(func(err error) { log.Error("shutdown error", xlog.Error("err", err)) }),
	)
	// Cleanup runs in registration order: stop serving, flip liveness, flush telemetry.
	sd.RegisterFnErr(
		func(ctx context.Context) error { return httpSrv.Shutdown(ctx) },
		func(ctx context.Context) error {
			if probeSrv != nil {
				return probeSrv.Shutdown(ctx)
			}
			return nil
		},
		func(ctx context.Context) error { live.Set(false); return nil },
		func(ctx context.Context) error {
			if telemetryShutdown == nil {
				return nil
			}
			err := telemetryShutdown(ctx)
			telemetryShutdown = nil
			return err
		},
	)
	// Background workers cancel with the shutdown context.
	sd.Go(func(ctx context.Context) {
		if err := ob.Run(ctx); err != nil && ctx.Err() == nil {
			log.Error("outbox relay stopped", xlog.Error("err", err))
		}
	})
	if probeSrv != nil {
		sd.Go(func(context.Context) {
			log.Info("probes listening", xlog.String("addr", probeAddr))
			if err := probeSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("probe serve failed", xlog.Error("err", err))
			}
		})
	}
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
	otelCore := xlog.NewFilterCore(
		xlogtrace.Core(logglobal.GetLoggerProvider().Logger(build.ServiceName)),
		xlog.NewAtomicLevel(level),
	)
	var base *xlog.Logger
	if c.Format == "text" {
		base = xlog.NewConsole(opts...)
	} else {
		base = xlog.NewJSON(opts...)
	}
	opts = append(opts,
		xlog.WithCore(xlog.NewTeeCore(base.Core(), otelCore)),
	)
	opts = append(opts, xlogtrace.Options(xlog.ErrorLevel)...)
	if c.Format == "text" {
		return xlog.NewConsole(opts...)
	}
	return xlog.NewJSON(opts...)
}

func buildFields() []xlog.Field {
	return []xlog.Field{
		xlog.String("service", build.ServiceName),
		xlog.String("version", build.Version),
		xlog.String("commit", build.Commit),
		xlog.String("build_time", build.BuildTime),
		xlog.String("instance_id", build.InstanceID),
	}
}

func buildSlogLogger() *slog.Logger {
	return slog.Default().With(
		slog.String("service", build.ServiceName),
		slog.String("version", build.Version),
		slog.String("commit", build.Commit),
		slog.String("build_time", build.BuildTime),
		slog.String("instance_id", build.InstanceID),
	)
}

// buildHandler assembles the full IAM handler from the Postgres adapters, one
// option per feature group.
func buildHandler(db *postgres.DB, emitter postgres.Emitter) *api.Service {
	platform := postgres.NewPgPlatform(db)          // implements PlatformConfig + PlatformCsrf
	coreAuth := postgres.NewPgCoreAuth(db, emitter) // implements CoreAuthAccounts + CoreAuthTokens

	return api.New(
		api.WithPlatform(api.NewPlatformService(api.PlatformDeps{
			Config: platform,
			Csrf:   platform,
		})),
		api.WithCoreAuth(api.NewCoreAuthService(api.CoreAuthDeps{
			Accounts: coreAuth,
			Tokens:   coreAuth,
			MFA:      postgres.NewPgMFAAccounts(db, emitter),
		})),
		api.WithCoreAuthFlows(api.CoreAuthFlowDeps{
			Flows: postgres.NewPgCoreAuthFlows(db, emitter, coreAuth),
		}),
		api.WithPasswordless(api.NewPasswordlessService(api.PasswordlessDeps{
			Accounts: postgres.NewPgPasswordlessAccounts(db, emitter),
		})),
		api.WithOAuthSocial(api.NewOAuthSocialService(api.OAuthSocialDeps{
			Accounts: postgres.NewPgOAuthSocial(db, emitter),
		})),
		api.WithWebAuthn(api.NewWebAuthnService(api.WebAuthnDeps{
			Accounts: postgres.NewPgWebAuthnAccounts(db, emitter),
		})),
		api.WithMFA(api.NewMFAService(api.MFADeps{
			Accounts: postgres.NewPgMFAAccounts(db, emitter),
		})),
		api.WithAccount(api.NewAccountService(api.AccountDeps{
			Accounts: postgres.NewPgAccountStore(db, emitter),
		})),
		api.WithMachineIdentity(api.NewMachineIdentityService(api.MachineIdentityDeps{
			Keys: postgres.NewPgMachineIdentities(db, emitter),
		})),
		api.WithFederation(api.NewFederationService(api.FederationDeps{
			Connections: postgres.NewPgFederationConnections(db, emitter),
			Runtime:     postgres.NewPgFederationRuntime(db, emitter),
			Scim:        postgres.NewPgFederationScim(db, emitter),
		})),
		api.WithOIDCProvider(api.NewOIDCProviderService(api.OIDCProviderDeps{
			Grants: postgres.NewPgOIDCGrants(db, emitter),
		})),
		api.WithAdmin(api.NewAdminService(api.AdminDeps{
			Users:           postgres.NewPgAdminUsers(db, emitter),
			Apps:            postgres.NewPgAdminApps(db, emitter),
			ServiceAccounts: postgres.NewPgAdminServiceAccounts(db, emitter),
			APIKeys:         postgres.NewPgAdminAPIKeys(db, emitter),
			Connections:     postgres.NewPgAdminConnections(db, emitter),
			Config:          postgres.NewPgAdminConfig(db, emitter),
			Keys:            postgres.NewPgAdminKeys(db, emitter),
			AccessRequests:  postgres.NewPgAdminAccessRequests(db, emitter),
		})),
		api.WithOperator(api.NewOperatorService(api.OperatorDeps{
			Projects: postgres.NewPgOperator(db, emitter),
		})),
	)
}
