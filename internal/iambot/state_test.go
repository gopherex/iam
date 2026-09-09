package iambot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateDiffMarkPrune(t *testing.T) {
	t.Parallel()

	s := &State{Seen: map[string]struct{}{"a": {}}}

	fresh := s.Diff([]string{"a", "b", "c"})
	if len(fresh) != 2 || fresh[0] != "b" || fresh[1] != "c" {
		t.Fatalf("diff = %v, want [b c]", fresh)
	}

	s.MarkSeen("b", "c")

	if got := s.Diff([]string{"a", "b", "c"}); len(got) != 0 {
		t.Fatalf("diff after mark = %v, want empty", got)
	}

	// a got approved (left the pending list) and must leave the seen set.
	s.Prune(map[string]struct{}{"b": {}, "c": {}})

	if _, ok := s.Seen["a"]; ok {
		t.Fatal("prune must drop ids no longer pending")
	}
}

func TestStateSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state", "bot.json")

	s := &State{Seen: map[string]struct{}{}}
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}

	s.MarkSeen("x")

	s.FirstRunDone = true
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}

	got, err := LoadState(path)
	if err != nil {
		t.Fatal(err)
	}

	if !got.FirstRunDone {
		t.Fatal("first_run_done lost")
	}

	if _, ok := got.Seen["x"]; !ok {
		t.Fatal("seen[x] lost")
	}
}

func TestLoadStateMissingFile(t *testing.T) {
	t.Parallel()

	s, err := LoadState(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}

	if s.FirstRunDone {
		t.Fatal("fresh state must not be first-run-done (cold-start absorb)")
	}

	if len(s.Seen) != 0 {
		t.Fatalf("fresh state seen = %v", s.Seen)
	}
}

func TestSaveAtomicNoTempLeftBehind(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bot.json")
	if err := (&State{Seen: map[string]struct{}{}}).Save(path); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}
