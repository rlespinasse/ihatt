package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/spf13/cobra"
)

// repoMatch reports whether a tracked repo is in the scope of a command.
type repoMatch func(*model.Repository) bool

// addScopeFlags registers the flags that pick which tracked repos a command acts on.
func addScopeFlags(c *cobra.Command) {
	c.Flags().String("repo", "", "Act only on this repository (by name or ID)")
	c.Flags().String("dir", "", "Act only on the repositories under this directory")
	c.Flags().Bool("all", false, "Act on every tracked repository, even from inside one")
}

// repoScope returns the repos a command should act on, and a label describing that scope.
// The order is: --repo, then --dir, then --all. Without a flag, from inside a tracked repo
// the scope is that repo; otherwise, from a directory holding tracked repos it is those
// repos; otherwise it is every repo.
func repoScope(cmd *cobra.Command, repos []*model.Repository) (repoMatch, string) {
	if name, _ := cmd.Flags().GetString("repo"); name != "" {
		return func(r *model.Repository) bool { return r.Name == name || r.ID == name }, "repository " + name
	}
	if dir, _ := cmd.Flags().GetString("dir"); dir != "" {
		return underDir(dir), "repositories under " + resolvePath(dir)
	}
	if all, _ := cmd.Flags().GetBool("all"); all {
		return allRepos, "all repositories"
	}
	if current := currentRepo(repos); current != nil {
		return func(r *model.Repository) bool { return r.ID == current.ID }, "repository " + current.Name + " (current directory; use --all for every repo)"
	}
	if wd, err := os.Getwd(); err == nil {
		match := underDir(wd)
		if n := countRepos(repos, match); n > 0 {
			return match, fmt.Sprintf("%d repositories under current directory (use --all for every repo)", n)
		}
	}
	return allRepos, "all repositories"
}

func allRepos(*model.Repository) bool {
	return true
}

// underDir matches the repos whose path is dir or inside it.
func underDir(dir string) repoMatch {
	dir = resolvePath(dir)
	return func(r *model.Repository) bool {
		return isUnder(resolvePath(r.Path), dir)
	}
}

// currentRepo returns the tracked repo that contains the working directory, or nil.
// With nested repos, the innermost one wins.
func currentRepo(repos []*model.Repository) *model.Repository {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	wd = resolvePath(wd)

	var best *model.Repository
	for _, repo := range repos {
		if !isUnder(wd, resolvePath(repo.Path)) {
			continue
		}
		if best == nil || len(repo.Path) > len(best.Path) {
			best = repo
		}
	}
	return best
}

func countRepos(repos []*model.Repository, match repoMatch) int {
	n := 0
	for _, r := range repos {
		if match(r) {
			n++
		}
	}
	return n
}

// isUnder reports whether path is dir or a path inside dir. Both must be resolved.
func isUnder(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, strings.TrimSuffix(dir, string(filepath.Separator))+string(filepath.Separator))
}

// resolvePath returns the absolute path with symlinks resolved, when it exists.
func resolvePath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	return p
}
