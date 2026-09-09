package iambot

import (
	"context"
	"time"
)

// Poller watches the pending list and announces requests the state has not
// seen yet. See State for how "new" is defined (seen-set diff, not time).
type Poller struct {
	bot   *Bot
	iam   *IAM
	state *State
}

// NewPoller builds the backlog watcher.
func NewPoller(bot *Bot, iam *IAM, state *State) *Poller {
	return &Poller{bot: bot, iam: iam, state: state}
}

// Run polls until ctx is cancelled. The first successful pass on a fresh state
// silently absorbs the existing backlog (a brand-new bot must not spam the
// whole pending history); everything pending stays reachable via /pending.
func (p *Poller) Run(ctx context.Context, interval time.Duration) error {
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
		return // transient: next tick retries; /pending surfaces errors on demand
	}

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
		// Nothing to do beyond continuing: the worst case after a crash is a
		// duplicate announcement for requests seen since the last save.
		_ = err
	}
}

// statePath is the persisted state location from the bot config.
func (p *Poller) statePath() string {
	return p.bot.cfg.StatePath
}
