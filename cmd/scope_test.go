package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/spf13/cobra"
)

func newScopeCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{}
	addScopeFlags(c)
	if err := c.ParseFlags(args); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestIsUnder(t *testing.T) {
	tests := []struct {
		path, dir string
		want      bool
	}{
		{"/a/b", "/a/b", true},
		{"/a/b/c", "/a/b", true},
		{"/a/bc", "/a/b", false},
		{"/a", "/a/b", false},
		{"/a/b", "/", true},
	}
	for _, tt := range tests {
		if got := isUnder(tt.path, tt.dir); got != tt.want {
			t.Errorf("isUnder(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
		}
	}
}

func TestRepoScopeDir(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "group", "one")
	nested := filepath.Join(root, "group", "two", "deep")
	outside := filepath.Join(root, "other")
	repos := []*model.Repository{
		{ID: "1", Name: "one", Path: inside},
		{ID: "2", Name: "deep", Path: nested},
		{ID: "3", Name: "other", Path: outside},
	}

	match, _ := repoScope(newScopeCmd(t, "--dir", filepath.Join(root, "group")), repos)
	for _, r := range repos {
		want := r.ID != "3"
		if match(r) != want {
			t.Errorf("--dir: repo %s matched = %v, want %v", r.Name, !want, want)
		}
	}
}

func TestRepoScopeAutoDetectsCurrentFolder(t *testing.T) {
	root := t.TempDir()
	group := filepath.Join(root, "group")
	repos := []*model.Repository{
		{ID: "1", Name: "one", Path: filepath.Join(group, "one")},
		{ID: "2", Name: "other", Path: filepath.Join(root, "other")},
	}

	for _, r := range repos {
		if err := os.MkdirAll(r.Path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(group)
	match, scope := repoScope(newScopeCmd(t), repos)
	if !match(repos[0]) || match(repos[1]) {
		t.Errorf("from %s: wrong repos in scope (%s)", group, scope)
	}

	empty := filepath.Join(root, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(empty)
	match, _ = repoScope(newScopeCmd(t), repos)
	if !match(repos[0]) || !match(repos[1]) {
		t.Errorf("from a folder without repos: want all repos in scope")
	}
}
