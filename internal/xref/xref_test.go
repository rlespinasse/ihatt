package xref

import (
	"testing"
	"time"

	"github.com/rlespinasse/ihatt/internal/model"
)

func TestMRReferencesOnlyForGitLab(t *testing.T) {
	commit := &model.Commit{Hash: "abc", RepoID: "r1", Message: "fix the thing (!42)", Date: time.Now()}

	if got := DetectFromCommit(commit, nil, false); len(got) != 0 {
		t.Errorf("github repo: got %d refs for !42, want 0", len(got))
	}

	got := DetectFromCommit(commit, nil, true)
	if len(got) != 1 || got[0].TargetType != "pr" || got[0].TargetID != "42" {
		t.Fatalf("gitlab repo: got %+v, want one pr ref to 42", got)
	}
}
