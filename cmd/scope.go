package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/spf13/cobra"
)

// addScopeFlags registers the flags that pick which tracked repos a command acts on.
func addScopeFlags(c *cobra.Command) {
	c.Flags().String("repo", "", "Act only on this repository (by name or ID)")
	c.Flags().Bool("all", false, "Act on every tracked repository, even from inside one")
}

// repoScope returns the repo filter a command should apply, by name or ID.
// --repo wins; --all and an unknown working directory mean no filter (""); from
// inside a tracked repo, the filter is that repo. label describes the scope for the user.
func repoScope(cmd *cobra.Command, repos []*model.Repository) (filter, label string) {
	if repo, _ := cmd.Flags().GetString("repo"); repo != "" {
		return repo, "repository " + repo
	}
	if all, _ := cmd.Flags().GetBool("all"); all {
		return "", "all repositories"
	}
	if current := currentRepo(repos); current != nil {
		return current.ID, "repository " + current.Name + " (current directory; use --all for every repo)"
	}
	return "", "all repositories"
}

// currentRepo returns the tracked repo that contains the working directory, or nil.
// With nested repos, the innermost one wins.
func currentRepo(repos []*model.Repository) *model.Repository {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	if resolved, err := filepath.EvalSymlinks(wd); err == nil {
		wd = resolved
	}

	var best *model.Repository
	for _, repo := range repos {
		path := repo.Path
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		if wd != path && !strings.HasPrefix(wd, path+string(filepath.Separator)) {
			continue
		}
		if best == nil || len(repo.Path) > len(best.Path) {
			best = repo
		}
	}
	return best
}
