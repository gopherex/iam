package postgres

import (
	"context"

	"github.com/gopherex/pgtx/pkg/tx"
)

// withTx runs fn in a serializable transaction with the mandatory default retry
// policy. Every mutating port method wraps its work in this; bob/sqld calls
// inside pick up the ambient transaction from ctx via db.Bobx() / db.q().
//
// Retry is REQUIRED for serializable workloads (40001 serialization failures
// are retried by tx.DefaultRetryPolicy); never issue a multi-statement mutation
// outside withTx / withTxRet.
func (db *DB) withTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return tx.DoSerializable(ctx, db.TxManager, fn, tx.WithRetry(tx.DefaultRetryPolicy))
}

// withTxRet is withTx for operations that return a value.
func withTxRet[T any](ctx context.Context, db *DB, fn func(ctx context.Context) (T, error)) (T, error) {
	return tx.DoSerializableRet(ctx, db.TxManager, fn, tx.WithRetry(tx.DefaultRetryPolicy))
}
