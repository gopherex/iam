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
	"errors"
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

// errEncryptionKeyRequired guards startup: secrets-at-rest encryption must be
// configured before the service accepts traffic.
var errEncryptionKeyRequired = errors.New("service.auth.encryption_key is required")

// apiPathPrefixes returns the URL namespaces served by the generated API.
// Anything outside them falls through to the SPA, which is how the hosted OIDC
// provider screens under /oauth/ are reached: /oauth2/ is the protocol surface
// and does NOT capture /oauth/, so those screens need no extra routing.
func apiPathPrefixes() []string {
	return []string{"/v1/", "/mgmt/", "/oauth2/", "/p/"}
}

func main() {
	if err := run(); err != nil {
		// run already logged the cause; exit non-zero for the supervisor.
		os.Exit(1)
	}
}

// setupTelemetry starts the tracing SDK and, when metrics scraping is
// enabled, the standalone metrics endpoint. With scraping on the SDK is asked
// not to build its own metrics provider — two providers would mean two sets
// of the same instruments. The returned shutdown func combines both (a nil
// component shuts down as a no-op), so the caller has one thing to defer.
func setupTelemetry(ctx context.Context, cfg *config.Config) (http.Handler, func(context.Context) error, error) {
	telemetryOpts := []xtracesdk.Option{
		xtracesdk.WithService(build.ServiceName),
		xtracesdk.WithVersion(build.Version),
		xtracesdk.WithInstanceID(build.InstanceID),
	}
	if cfg.Service.HTTP.MetricsEnabled {
		telemetryOpts = append(telemetryOpts, xtracesdk.WithoutMetrics())
	}

	telemetryShutdown, err := xtracesdk.Setup(ctx, telemetryOpts...)
	if err != nil {
		return nil, nil, err
	}

	if !cfg.Service.HTTP.MetricsEnabled {
		return nil, telemetryShutdown, nil
	}

	metricsHandler, metricsShutdown, err := setupMetrics(ctx, build.ServiceName, build.Version, build.InstanceID)
	if err != nil {
		return nil, telemetryShutdown, err
	}

	shutdown := func(ctx context.Context) error {
		return errors.Join(telemetryShutdown(ctx), metricsShutdown(ctx))
	}

	return metricsHandler, shutdown, nil
}

// buildRootHandler assembles the top-level request router: API namespaces go
// through the full middleware pipeline (outermost first: request meta,
// X-Environment, CSRF for cookie-mode requests evaluated before cookie auth
// while there is no Authorization header, cookie auth promoting the session
// cookie to a Bearer header, CORS, security headers, then per-project rate
// limiting) to the generated server; everything else falls through to the
// embedded admin SPA. Probes/metrics are mounted here too, unless
// ProbeAddr wants them on a separate listener (a k8s sidecar port not
// exposed publicly) — the returned bool tells the caller which.
func buildRootHandler(cfg *config.Config, db *postgres.DB, emitter postgres.Emitter, auth api.Authenticator, srv http.Handler, probeMux *http.ServeMux, metricsHandler http.Handler) (http.Handler, bool) {
	// Trust forwarding headers (X-Forwarded-For/X-Real-IP) only from configured
	// proxy networks; otherwise the real TCP peer is used so clients cannot spoof
	// their IP to bypass IP-keyed rate limits.
	api.SetTrustedProxies(cfg.Service.HTTP.TrustedProxies)

	apiPipeline := api.RequestMetaMiddleware(
		api.EnvironmentMiddleware(
			api.TestClockMiddleware(postgres.NewPgTestMode(db, emitter),
				api.CSRFMiddleware(postgres.NewPgPlatform(db))(
					api.CookieAuthMiddleware(
						api.SoftAuthMiddleware(auth)(srv))))))
	// CORS allows the statically configured origins plus the dynamic per-client
	// union (app clients' allowed_origins), cached for 60s.
	apiPipeline = api.CORSMiddleware(cfg.Service.CORS.AllowedOrigins, postgres.NewPgAdminApps(db, emitter), 60*time.Second)(apiPipeline)
	apiPipeline = api.SecurityHeaders(apiPipeline)
	// Rate limiting honors per-project overrides (iam_config key=rate_limits),
	// falling back to the hardcoded defaults when a project has no doc.
	apiPipeline = api.NewRateLimitMiddleware(postgres.NewPgRateLimits(db))(apiPipeline)

	root := http.NewServeMux()
	// API namespaces go to the generated server; everything else is the embedded
	// admin SPA (a stub until the binary is built with `make build` / -tags embed).
	for _, prefix := range apiPathPrefixes() {
		root.Handle(prefix, apiPipeline)
	}

	root.Handle("/", api.SecurityHeaders(web.Handler()))

	// Metrics ride with the probes: the same listener a cluster already scrapes,
	// and the same one it does not expose.
	if metricsHandler != nil {
		probeMux.Handle(metricsPath, metricsHandler)
	}

	separateProbes := cfg.Service.HTTP.ProbeAddr != "" && cfg.Service.HTTP.ProbeAddr != cfg.Service.HTTP.Addr
	if !separateProbes {
		root.Handle("/healthz/", probeMux)

		if metricsHandler != nil {
			root.Handle(metricsPath, metricsHandler)
		}
	}

	return root, separateProbes
}

// buildServers assembles the liveness/readiness probe mux, the root HTTP
// handler, and the two net/http servers: the API server always, and a
// separate probe server only when ProbeAddr wants its own listener.
func buildServers(cfg *config.Config, db *postgres.DB, emitter postgres.Emitter, auth api.Authenticator, srv, metricsHandler http.Handler) (*http.Server, *http.Server, *xprobe.Bool) {
	live := xprobe.NewBool()
	live.Set(true)
	probeMux := xprobe.Mux(
		xprobe.Liveness(live),
		xprobe.Readiness(xprobe.FromError(db.Ping)),
	)

	root, separateProbes := buildRootHandler(cfg, db, emitter, auth, srv, probeMux, metricsHandler)

	httpSrv := &http.Server{
		Addr:           cfg.Service.HTTP.Addr,
		Handler:        http.MaxBytesHandler(root, 1<<20),
		ReadTimeout:    time.Duration(cfg.Service.HTTP.ReadTimeoutSec) * time.Second,
		WriteTimeout:   time.Duration(cfg.Service.HTTP.WriteTimeoutSec) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	var probeSrv *http.Server
	if separateProbes {
		// A probe request is a tiny GET with no body, so a short bound is safe
		// and is what closes off Slowloris against this listener too.
		probeSrv = &http.Server{Addr: cfg.Service.HTTP.ProbeAddr, Handler: probeMux, ReadHeaderTimeout: 5 * time.Second}
	}

	return httpSrv, probeSrv, live
}

// registerShutdownHooks registers the cleanup steps sd runs, in order: stop
// serving (API, then probes), flip liveness, flush telemetry. telemetryShutdown
// is a pointer since it is nilled out after running (RegisterFnErr's cleanups
// can run more than once — e.g. a signal during an already-running shutdown).
func registerShutdownHooks(sd *xshutdown.Manager, httpSrv, probeSrv *http.Server, live *xprobe.Bool, telemetryShutdown *func(context.Context) error) {
	sd.RegisterFnErr(
		func(ctx context.Context) error { return httpSrv.Shutdown(ctx) },
		func(ctx context.Context) error {
			if probeSrv != nil {
				return probeSrv.Shutdown(ctx)
			}

			return nil
		},
		func(_ context.Context) error { live.Set(false); return nil },
		func(ctx context.Context) error {
			if *telemetryShutdown == nil {
				return nil
			}

			err := (*telemetryShutdown)(ctx)
			*telemetryShutdown = nil

			return err
		},
	)
}

// startBackgroundWorkers launches the service's long-running loops; they
// cancel with the shutdown context sd provides.
func startBackgroundWorkers(sd *xshutdown.Manager, ob *outbox.Outbox, db *postgres.DB, webhooks *postgres.PgWebhooks, log *xlog.Logger) {
	sd.Go(func(ctx context.Context) {
		if err := ob.Run(ctx); err != nil && ctx.Err() == nil {
			log.Error("outbox relay stopped", xlog.Error("err", err))
		}
	})
	// Garbage collector: prune expired runtime rows (challenges, flows, auth /
	// device / PAR codes, timed-out sessions and refresh tokens) that are only
	// filtered read-side and would otherwise grow without bound.
	sd.Go(func(ctx context.Context) {
		db.RunGarbageCollector(ctx, 0, log.AppendName("gc"))
	})
	// Webhook retry: drain deliveries whose exponential backoff has elapsed
	// (deliver() writes next_attempt_at; nothing consumed it before).
	sd.Go(func(ctx context.Context) {
		webhooks.RunRetryWorker(ctx, 0, log.AppendName("webhook-retry"))
	})
	// Jobs: drain pending async jobs (bulk user import, audit/data exports).
	sd.Go(func(ctx context.Context) {
		db.RunJobsWorker(ctx, 0, log.AppendName("jobs"))
	})
}

// startServing launches the listener goroutines: the probe server (when it
// has its own listener) and the API server.
func startServing(sd *xshutdown.Manager, httpSrv, probeSrv *http.Server, probeAddr, apiAddr string, log *xlog.Logger) {
	if probeSrv != nil {
		sd.Go(func(context.Context) {
			log.Info("probes listening", xlog.String("addr", probeAddr))

			if err := probeSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("probe serve failed", xlog.Error("err", err))
			}
		})
	}

	sd.Go(func(context.Context) {
		log.Info("listening", xlog.String("addr", apiAddr))

		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http serve failed", xlog.Error("err", err))
		}
	})
}

// setupDatabase connects to Postgres, wires the at-rest cipher and public URL
// the rest of the service depends on, and applies migrations. It refuses to
// start with secrets-at-rest encryption disabled (an empty EncryptionKey
// would otherwise silently yield the identity/passthrough cipher).
func setupDatabase(ctx context.Context, cfg *config.Config, log *xlog.Logger) (*postgres.DB, error) {
	if cfg.Service.Auth.EncryptionKey == "" {
		log.Error("secrets-at-rest encryption is DISABLED — set service.auth.encryption_key (base64 32-byte AES-256 key) before running in production")
		return nil, errEncryptionKeyRequired
	}

	db, err := postgres.Connect(ctx, cfg.Infra.Postgres.DSN(),
		postgres.WithLogger(log.AppendName("postgres")),
		postgres.WithQueryLogLevel(cfg.Infra.Postgres.LogLevel),
		postgres.WithMetrics(true),
	)
	if err != nil {
		log.Error("postgres connect failed", xlog.Error("err", err))
		return nil, err
	}

	cph, err := postgres.NewCipher(cfg.Service.Auth.EncryptionKey)
	if err != nil {
		log.Error("encryption key invalid", xlog.Error("err", err))
		return nil, err
	}

	db.UseCipher(cph)
	// The public base URL is the OIDC issuer prefix and the origin of every
	// absolute URL the service advertises. Normalize() already proved it is an
	// absolute http(s) URL without a trailing slash.
	db.UsePublicURL(cfg.Service.HTTP.PublicURL)

	if err := runMigrations(ctx, db, log); err != nil {
		return nil, err
	}

	return db, nil
}

// runMigrations applies the service's own migrations plus the outbox
// library's (it owns its own table, outbox_messages, so it migrates itself).
func runMigrations(ctx context.Context, db *postgres.DB, log *xlog.Logger) error {
	if err := db.Migrate(ctx); err != nil {
		log.Error("migrate failed", xlog.Error("err", err))
		return err
	}

	for _, stmt := range outbox.Migrations() {
		if _, err := db.Pool.Exec(ctx, stmt); err != nil {
			log.Error("outbox migrate failed", xlog.Error("err", err))
			return err
		}
	}

	log.Info("migrations applied")

	return nil
}

// buildAPIServer wires the ogen-generated server (12 feature groups over
// Postgres adapters) over the master-key/session authenticator.
func buildAPIServer(cfg *config.Config, db *postgres.DB, emitter postgres.Emitter, webhooks *postgres.PgWebhooks) (api.Authenticator, *oas.Server, error) {
	handler := buildHandler(db, emitter, webhooks)
	auth := postgres.NewAuthenticator(db, cfg.Service.Auth.MasterKey)

	srv, err := oas.NewServer(handler, api.NewSecurityHandler(auth), oas.WithErrorHandler(api.ErrorHandler))
	if err != nil {
		return nil, nil, err
	}

	return auth, srv, nil
}

// loadConfig loads and normalizes the typed config from $CONFIG_PATH/config.*,
// an optional .env, IAM_-prefixed env vars, and (if set) an explicit
// -config file, in that precedence order.
func loadConfig(configPath string) (*config.Config, error) {
	loadOpts := []structconf.Option{
		structconf.WithConfigPath("config"), // $CONFIG_PATH/config.{yaml,...}
		structconf.WithDotEnv(),             // optional .env in cwd
		structconf.WithEnvPrefix("IAM"),     // IAM_INFRA_POSTGRES_HOST, ...
	}
	if configPath != "" {
		loadOpts = append(loadOpts, structconf.WithFile(configPath))
	}

	cfg, err := structconf.Load[config.Config](loadOpts...)
	if err != nil {
		slog.Error("config load failed", "err", err)
		return nil, err
	}

	if err := cfg.Normalize(); err != nil {
		slog.Error("config invalid", "err", err)
		return nil, err
	}

	return cfg, nil
}

// setupOutboxAndEmitter builds the outbox relay (email publisher; enqueue
// joins the caller tx via db.TxDB — BatchSize=1 because the relay marks the
// WHOLE claimed batch for retry when Publish returns an error, so batching
// delivery would re-send messages already delivered earlier in the batch on
// retry) and the auditing emitter wrapping it, so privileged (admin/
// operator/service) mutations record an audit-log row in the same
// transaction as the mutation. When configured, it also seeds the root
// operator/project so there is something to manage on first boot.
func setupOutboxAndEmitter(ctx context.Context, cfg *config.Config, db *postgres.DB, webhooks *postgres.PgWebhooks, log *xlog.Logger) (*outbox.Outbox, postgres.Emitter, error) {
	ob, err := outbox.New(db.Pool, db.TxDB, notifications.NewPublisher(db, webhooks, log.AppendName("outbox")),
		outbox.WithInstanceID(build.InstanceID),
		outbox.WithLogger(buildSlogLogger()),
		outbox.WithPollInterval(time.Second),
		outbox.WithBatchSize(1),
		outbox.WithMaxAttempts(10),
		outbox.WithRetryBackoff(outbox.ExpBackoff(time.Second, 5*time.Minute, true)),
	)
	if err != nil {
		log.Error("outbox init failed", xlog.Error("err", err))
		return nil, nil, err
	}

	emitter := postgres.NewAuditingEmitter(db, postgres.NewOutboxEmitter(ob))

	if cfg.Service.Auth.SeedRoot {
		if err := seedRoot(ctx, db, emitter, log); err != nil {
			log.Error("seed root failed", xlog.Error("err", err))
			return nil, nil, err
		}
	}

	return ob, emitter, nil
}

// newServiceLogger builds the application logger and starts the telemetry
// SDK's host-runtime instrumentation (proc/runtime metrics).
func newServiceLogger(cfg *config.Config) (*xlog.Logger, error) {
	log := newLogger(cfg.Service.Logger).AppendName(build.ServiceName).With(buildFields()...)

	if err := xtracesdk.StartHostRuntime(); err != nil {
		log.Error("telemetry runtime instrumentation failed", xlog.Error("err", err))
		return nil, err
	}

	return log, nil
}

func run() error {
	configPath := flag.String("config", "", "path to config file (yaml/json/toml); overrides CONFIG_PATH discovery")

	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}

	ctx := context.Background()

	metricsHandler, telemetryShutdown, err := setupTelemetry(ctx, cfg)
	if err != nil {
		slog.Error("telemetry setup failed", "err", err)
		return err
	}

	defer func() {
		if telemetryShutdown != nil {
			_ = telemetryShutdown(context.Background())
		}
	}()

	log, err := newServiceLogger(cfg)
	if err != nil {
		return err
	}

	log.Info("starting", xlog.String("addr", cfg.Service.HTTP.Addr))

	// ----- postgres -----
	db, err := setupDatabase(ctx, cfg, log)
	if err != nil {
		return err
	}

	defer db.Close()

	webhooks := postgres.NewPgWebhooks(db, nil)

	ob, emitter, err := setupOutboxAndEmitter(ctx, cfg, db, webhooks, log)
	if err != nil {
		return err
	}

	auth, srv, err := buildAPIServer(cfg, db, emitter, webhooks)
	if err != nil {
		log.Error("server build failed", xlog.Error("err", err))
		return err
	}

	probeAddr := cfg.Service.HTTP.ProbeAddr
	httpSrv, probeSrv, live := buildServers(cfg, db, emitter, auth, srv, metricsHandler)

	// ----- shutdown orchestration -----
	sd := xshutdown.New(ctx,
		xshutdown.WithTimeout(time.Duration(cfg.Service.HTTP.ShutdownSec)*time.Second),
		xshutdown.WithErrorHandler(func(err error) { log.Error("shutdown error", xlog.Error("err", err)) }),
	)
	registerShutdownHooks(sd, httpSrv, probeSrv, live, &telemetryShutdown)
	startBackgroundWorkers(sd, ob, db, webhooks, log)
	startServing(sd, httpSrv, probeSrv, probeAddr, cfg.Service.HTTP.Addr, log)

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

	// Level, caller, the tee core, and the trace options appended below.
	const loggerOptionCount = 4

	opts := make([]xlog.Option, 0, loggerOptionCount)
	opts = append(opts, xlog.WithLevel(level), xlog.WithCaller(true))
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
func buildHandler(db *postgres.DB, emitter postgres.Emitter, webhooks *postgres.PgWebhooks) *api.Service {
	platform := postgres.NewPgPlatform(db) // implements PlatformConfig + PlatformCsrf
	// cfgReader is the shared runtime reader for project-config docs
	// (password_policy / session_policy / auth / ...). One instance is injected
	// into every enforcement adapter so its TTL cache is shared.
	cfgReader := postgres.NewConfigReader(db, 30*time.Second)
	coreAuth := postgres.NewPgCoreAuth(db, emitter, cfgReader) // implements CoreAuthAccounts + CoreAuthTokens

	return api.New(
		api.WithPlatform(api.NewPlatformService(api.PlatformDeps{
			Config: platform,
			Csrf:   platform,
		})),
		api.WithCoreAuth(api.NewCoreAuthService(api.CoreAuthDeps{
			Accounts: coreAuth,
			Tokens:   coreAuth,
			MFA:      postgres.NewPgMFAAccounts(db, emitter, cfgReader),
		})),
		api.WithCoreAuthFlows(api.CoreAuthFlowDeps{
			Flows: postgres.NewPgCoreAuthFlows(db, emitter, coreAuth, cfgReader),
		}),
		api.WithPasswordless(api.NewPasswordlessService(api.PasswordlessDeps{
			Accounts: postgres.NewPgPasswordlessAccounts(db, emitter, cfgReader, coreAuth),
		})),
		api.WithOAuthSocial(api.NewOAuthSocialService(api.OAuthSocialDeps{
			Accounts: postgres.NewPgOAuthSocial(db, emitter, cfgReader),
		})),
		api.WithWebAuthn(api.NewWebAuthnService(api.WebAuthnDeps{
			Accounts: postgres.NewPgWebAuthnAccounts(db, emitter, cfgReader),
		})),
		api.WithMFA(api.NewMFAService(api.MFADeps{
			Accounts: postgres.NewPgMFAAccounts(db, emitter, cfgReader),
		})),
		api.WithAccount(api.NewAccountService(api.AccountDeps{
			Accounts: postgres.NewPgAccountStore(db, emitter),
		})),
		api.WithMachineIdentity(api.NewMachineIdentityService(api.MachineIdentityDeps{
			Keys: postgres.NewPgMachineIdentities(db, emitter),
		})),
		api.WithFederation(api.NewFederationService(api.FederationDeps{
			Connections: postgres.NewPgFederationConnections(db, emitter),
			Runtime:     postgres.NewPgFederationRuntime(db, emitter, cfgReader),
			Scim:        postgres.NewPgFederationScim(db, emitter),
		})),
		api.WithOIDCProvider(api.NewOIDCProviderService(api.OIDCProviderDeps{
			Grants: postgres.NewPgOIDCGrants(db, emitter, cfgReader),
		})),
		api.WithAdmin(api.NewAdminService(api.AdminDeps{
			Users:           postgres.NewPgAdminUsers(db, emitter),
			Apps:            postgres.NewPgAdminApps(db, emitter),
			ServiceAccounts: postgres.NewPgAdminServiceAccounts(db, emitter),
			APIKeys:         postgres.NewPgAdminAPIKeys(db, emitter),
			Connections:     postgres.NewPgAdminConnections(db, emitter),
			Config:          postgres.NewPgAdminConfig(db, emitter),
			Roles:           postgres.NewPgRoles(db, emitter),
			Keys:            postgres.NewPgAdminKeys(db, emitter),
			AccessRequests:  postgres.NewPgAdminAccessRequests(db, emitter),
			Invites:         postgres.NewPgInvites(db, emitter),
			Webhooks:        webhooks,
			Grants:          postgres.NewPgOIDCGrants(db, emitter, cfgReader),
			Audit:           postgres.NewPgAudit(db, emitter),
			Jobs:            postgres.NewPgJobs(db, emitter),
			Hooks:           postgres.NewPgHooks(db, emitter),
			Risk:            postgres.NewPgRisk(db, emitter),
			TestMode:        postgres.NewPgTestMode(db, emitter),
		})),
		api.WithOperator(api.NewOperatorService(api.OperatorDeps{
			Projects: postgres.NewPgOperator(db, emitter),
		})),
	)
}
