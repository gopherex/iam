package postgres

// Store is the Postgres-backed persistence façade. It exposes typed repos that
// issue SQL through the sqld-generated query set bound to db.TxDB (the ctx-aware
// pgtx executor), so a repo call made inside a service's tx.Do* runs in that
// same transaction; outside one it runs on the pool (auto-commit).
//
// Repos are added per domain entity as the IAM domain is designed; each follows
// the envelope pattern: marshal the record to JSON, call the generated query,
// translate pgx errors via translatePgErr / isUniqueViolation.
type Store struct{ db *DB }

// New wraps a *DB. It does not touch the database.
func New(db *DB) *Store { return &Store{db: db} }

// Store returns the typed-repo façade over this connection bundle.
func (db *DB) Store() *Store { return &Store{db: db} }
