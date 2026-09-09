package iambot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// State is the bot's durable memory of which access requests were already
// announced. The admin list endpoint exposes no created_at and its ordering is
// id-lexicographic (not temporal), so "new" is defined exactly as "pending id
// not in the seen set" — immune to clocks, ordering, pagination and restarts.
type State struct {
	// Seen holds the ids of pending requests already pushed to the operators.
	// Ids leave the set when they leave the pending list (approved/denied), so
	// it stays the size of the current pending backlog.
	Seen map[string]struct{} `json:"seen"`

	// FirstRunDone marks that the cold-start backlog was silently absorbed.
	FirstRunDone bool `json:"first_run_done"`
}

// LoadState reads the state file; a missing file yields a fresh empty state
// (and firstRunDone=false so the caller can absorb the backlog quietly).
func LoadState(path string) (*State, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Seen: map[string]struct{}{}}, nil
		}

		return nil, fmt.Errorf("read state: %w", err)
	}

	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("parse state %s: %w", path, err)
	}

	if s.Seen == nil {
		s.Seen = map[string]struct{}{}
	}

	return &s, nil
}

// Diff returns the ids from ids that were not seen yet, preserving order.
func (s *State) Diff(ids []string) []string {
	var fresh []string

	for _, id := range ids {
		if _, ok := s.Seen[id]; !ok {
			fresh = append(fresh, id)
		}
	}

	return fresh
}

// MarkSeen records ids as announced.
func (s *State) MarkSeen(ids ...string) {
	for _, id := range ids {
		s.Seen[id] = struct{}{}
	}
}

// Prune drops seen ids that are no longer pending, keeping the set bounded by
// the live backlog.
func (s *State) Prune(live map[string]struct{}) {
	for id := range s.Seen {
		if _, ok := live[id]; !ok {
			delete(s.Seen, id)
		}
	}
}

// State file permissions: dir 0700 (private), file 0600.
const (
	stateDirPerm  os.FileMode = 0o700
	stateFilePerm os.FileMode = 0o600
)

// Save persists the state atomically (write to a temp file, then rename).
func (s *State) Save(path string) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), stateDirPerm); err != nil && filepath.Dir(path) != "." {
		return fmt.Errorf("mkdir state dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, stateFilePerm); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}

	return nil
}
