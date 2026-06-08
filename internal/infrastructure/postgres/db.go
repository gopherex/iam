// Package postgres is the Postgres-backed persistence for IAM: a hand-written
// pgx + pgtx + bob layer following the komeet pattern — a single *pgxpool.Pool,
// a pgtx transaction manager (tx.Trm) services use to run repo calls inside an
// ambient transaction, a ctx-aware TxDB executor, and a bob pool for typed
// query building.
//
// Storage model: each record type maps to a table mirroring the queryable
// envelope columns (id, project_id, created_at, updated_at, plus any secondary
// keys) with the full domain object stored as a `data jsonb` column. Reads map
// pgx.ErrNoRows onto a domain not-found; unique violations onto a conflict.
package postgres

import (
	"context"

	"github.com/gopherex/pgtx"
	pgtxotel "github.com/gopherex/pgtx/contrib/otel"
	pgtxlib "github.com/gopherex/pgtx/pkg/tx"
	"github.com/gopherex/xlog"
	pgxlog "github.com/gopherex/xlog/contrib/libs/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	bobpgx "github.com/stephenafamo/bob/drivers/pgx"
)

type connectOptions struct {
	logger        *xlog.Logger
	queryLogLevel string
	metrics       bool
}

// ConnectOption customizes Postgres connection instrumentation.
type ConnectOption func(*connectOptions)

// WithLogger routes pgx query logs through xlog.
func WithLogger(logger *xlog.Logger) ConnectOption {
	return func(o *connectOptions) {
		o.logger = logger
	}
}

// WithQueryLogLevel overrides the pgx query log level.
func WithQueryLogLevel(level string) ConnectOption {
	return func(o *connectOptions) {
		o.queryLogLevel = level
	}
}

// WithMetrics registers OpenTelemetry metrics for the pgx pool.
func WithMetrics(enabled bool) ConnectOption {
	return func(o *connectOptions) {
		o.metrics = enabled
	}
}

// DB is the Postgres connection bundle: the raw pool, the pgtx transaction
// manager services run repo calls through, the ctx-aware TxDB executor (picks up
// the ambient transaction from ctx) and a bob pool over the same pool.
type DB struct {
	Pool      *pgxpool.Pool
	TxManager pgtxlib.Trm
	TxDB      pgtxlib.DB
	Bob       bobpgx.Pool
	// Cipher encrypts reversible secrets at rest (signing-key PEMs, TOTP secrets).
	// Defaults to a passthrough cipher; cmd installs a real one via UseCipher.
	Cipher Cipher
}

// Connect opens a pgx pool against dsn and builds the tx manager / executors.
func Connect(ctx context.Context, dsn string, opts ...ConnectOption) (*DB, error) {
	options := connectOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	if options.logger != nil {
		tracerOpts := []pgxlog.Option(nil)
		if options.queryLogLevel != "" {
			level, err := tracelog.LogLevelFromString(options.queryLogLevel)
			if err != nil {
				return nil, err
			}
			tracerOpts = append(tracerOpts, pgxlog.WithLogLevel(level))
		}
		tracer, err := pgxlog.NewTracer(options.logger, tracerOpts...)
		if err != nil {
			return nil, err
		}
		cfg.ConnConfig.Tracer = tracer
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if options.metrics {
		if err := pgtxotel.RegisterMetrics(pool); err != nil {
			pool.Close()
			return nil, err
		}
	}
	txManager, err := pgtx.NewTxManager(pool, pgtxlib.ReadCommitted())
	if err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{
		Pool:      pool,
		TxManager: txManager,
		TxDB:      pgtx.NewTxDB(pool),
		Bob:       bobpgx.NewPool(pool),
		Cipher:    NewIdentityCipher(),
	}, nil
}

// UseCipher installs the at-rest secret cipher (call once after Connect, before
// serving). A nil cipher is ignored, keeping the passthrough default.
func (db *DB) UseCipher(c Cipher) {
	if c != nil {
		db.Cipher = c
	}
}

// Trm returns the transaction manager services wire into their tx.Trm field so
// repo calls inside tx.Do* share one transaction.
func (db *DB) Trm() pgtxlib.Trm { return db.TxManager }

// Close releases the underlying pool.
func (db *DB) Close() {
	if db != nil && db.Pool != nil {
		db.Pool.Close()
	}
}

// Ping verifies connectivity.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
