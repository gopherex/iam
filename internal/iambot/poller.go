package iambot

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Poller watches the pending list and announces requests the state has not
// seen yet. See State for how "new" is defined (seen-set diff, not time).
type Poller struct {
	bot   *Bot
	iam   *IAM
	state *State

	interval time.Duration

	mu          sync.RWMutex
	started     bool
	lastSuccess time.Time
	lastErr     error
}

// NewPoller builds the backlog watcher.
func NewPoller(bot *Bot, iam *IAM, state *State) *Poller {
	return &Poller{bot: bot, iam: iam, state: state}
}

// Run polls until ctx is cancelled. The first successful pass on a fresh state
// silently absorbs the existing backlog (a brand-new bot must not spam the
// whole pending history); everything pending stays reachable via /pending.
func (p *Poller) Run(ctx context.Context, interval time.Duration) error {
	p.interval = interval

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		p.pollOnce(ctx)

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// pollOnce lists the pending backlog, announces the unseen part, and persists
// the pruned seen-set.
func (p *Poller) pollOnce(ctx context.Context) {
	pending, err := p.iam.ListPending(ctx)
	if err != nil {
		p.noteResult(err)

		return // transient: next tick retries; /pending surfaces errors on demand
	}

	p.noteResult(nil)

	live := make(map[string]struct{}, len(pending))
	ids := make([]string, 0, len(pending))

	byID := make(map[string]AccessRequest, len(pending))
	for i := range pending {
		live[pending[i].ID] = struct{}{}
		ids = append(ids, pending[i].ID)
		byID[pending[i].ID] = pending[i]
	}

	fresh := p.state.Diff(ids)

	// Cold start: absorb the backlog quietly so the chat is not flooded with
	// pre-existing requests the operators can already see via /pending.
	if !p.state.FirstRunDone {
		fresh = nil
		p.state.FirstRunDone = true
	}

	for _, id := range fresh {
		req := byID[id]
		if err := p.bot.Announce(ctx, &req); err != nil {
			// Announce best-effort per request: a user who blocked the bot
			// must not wedge the rest of the announcements.
			continue
		}
	}

	p.state.MarkSeen(ids...)
	p.state.Prune(live)

	if err := p.state.Save(p.statePath()); err != nil {
		p.bot.log.Warn("state save failed", errField(err), idField(p.statePath()))
	}
}

// noteResult records the outcome of a polling pass for HealthCheck.
func (p *Poller) noteResult(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.started = true

	p.lastErr = err
	if err == nil {
		p.lastSuccess = time.Now()
	}
}

// errHealthNotStarted / errHealthStale are the readiness failure modes.
var (
	errHealthNotStarted = errors.New("iambot: poller has not completed a pass yet")
	errHealthStale      = errors.New("iambot: last successful poll is stale")
)

// staleFactor is how many intervals the last successful poll may age before
// readiness goes red.
const staleFactor = 3

// HealthCheck reports readiness: at least one pass ran, the last pass did not
// error, and the last success is not older than a few intervals.
func (p *Poller) HealthCheck(context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return errHealthNotStarted
	}

	if p.lastErr != nil {
		return p.lastErr
	}

	stale := time.Duration(staleFactor) * p.interval
	if stale > 0 && time.Since(p.lastSuccess) > stale {
		return errHealthStale
	}

	return nil
}

// statePath is the persisted state location from the bot config.
func (p *Poller) statePath() string {
	return p.bot.cfg.StatePath
}
