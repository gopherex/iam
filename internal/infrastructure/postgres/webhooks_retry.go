package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/gopherex/xlog"
)

const (
	// webhookRetryInterval is how often the worker scans for due retries.
	webhookRetryInterval = 30 * time.Second
	// webhookRetryBatch bounds how many due deliveries one scan drains.
	webhookRetryBatch = 100
	// webhookMaxDeliveryAttempts caps automatic retries; beyond it a delivery is
	// left 'failed' (recoverable via the manual RetryDelivery admin action) so a
	// permanently-broken endpoint stops consuming worker cycles forever.
	webhookMaxDeliveryAttempts = 10
)

// RunRetryWorker periodically re-delivers webhook deliveries whose backoff has
// elapsed. deliver() already writes next_attempt_at with exponential backoff on
// each failure and the partial index idx_iam_webhook_deliveries_retry serves the
// due-scan; previously nothing consumed that column, so a failed delivery was
// stranded until a manual retry. Blocks until ctx is cancelled.
func (a *PgWebhooks) RunRetryWorker(ctx context.Context, interval time.Duration, log *xlog.Logger) {
	if interval <= 0 {
		interval = webhookRetryInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.drainDueDeliveries(ctx, log)
		}
	}
}

func (a *PgWebhooks) drainDueDeliveries(ctx context.Context, log *xlog.Logger) {
	rows, err := a.db.Pool.Query(ctx, `SELECT id FROM iam_webhook_deliveries
		WHERE status IN ('pending', 'failed')
		  AND next_attempt_at IS NOT NULL AND next_attempt_at <= now()
		  AND attempt_count < $1
		ORDER BY next_attempt_at
		LIMIT $2`, webhookMaxDeliveryAttempts, webhookRetryBatch)
	if err != nil {
		if ctx.Err() == nil {
			log.Warn("webhook retry scan failed", xlog.Error("err", err))
		}

		return
	}

	var ids []string

	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr == nil {
			ids = append(ids, id)
		}
	}

	rows.Close()

	var delivered int

	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		// deliver records success or reschedules failure (with backoff) on the
		// row itself; a returned error is the transport/DB failure already
		// persisted, so it does not need to abort the batch.
		if _, err := a.deliver(ctx, id, false); err == nil {
			delivered++
		}
	}

	if len(ids) > 0 {
		log.Info("webhook retry drained",
			xlog.String("due", strconv.Itoa(len(ids))), xlog.String("delivered", strconv.Itoa(delivered)))
	}
}
