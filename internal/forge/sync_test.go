package forge

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

func openStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.NewBoltStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestFirstSyncIsFull(t *testing.T) {
	s := openStore(t)
	plan, err := Start(s, "repo", "github", "issue", false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Incremental() {
		t.Fatal("no watermark yet: sync must be full")
	}
}

func TestWatermarkMakesNextSyncIncremental(t *testing.T) {
	s := openStore(t)
	plan, _ := Start(s, "repo", "github", "issue", false)
	if err := plan.Done(s); err != nil {
		t.Fatal(err)
	}

	next, err := Start(s, "repo", "github", "issue", false)
	if err != nil {
		t.Fatal(err)
	}
	if !next.Incremental() {
		t.Fatal("after a completed sync the next one must be incremental")
	}
	if next.Since.After(plan.started) {
		t.Errorf("Since %v must reach back past the watermark %v (overlap)", next.Since, plan.started)
	}
}

func TestFullFlagIgnoresWatermark(t *testing.T) {
	s := openStore(t)
	plan, _ := Start(s, "repo", "github", "issue", false)
	plan.Done(s)

	full, _ := Start(s, "repo", "github", "issue", true)
	if full.Incremental() {
		t.Fatal("--full must ignore the watermark")
	}
}

func TestStaleSchemaForcesFullSync(t *testing.T) {
	s := openStore(t)
	key := "repo:github:issue"
	if err := s.PutSyncState(key, &model.SyncState{Watermark: time.Time{}, Schema: model.ForgeSchemaVersion - 1}); err != nil {
		t.Fatal(err)
	}
	plan, _ := Start(s, "repo", "github", "issue", false)
	if plan.Incremental() {
		t.Fatal("a schema-version mismatch must force a full sync")
	}
}

func TestKindsAreTrackedSeparately(t *testing.T) {
	s := openStore(t)
	issues, _ := Start(s, "repo", "gitlab", "issue", false)
	issues.Done(s)

	prs, _ := Start(s, "repo", "gitlab", "pr", false)
	if prs.Incremental() {
		t.Fatal("completing issues must not advance the pr watermark")
	}
}
