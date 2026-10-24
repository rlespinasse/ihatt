package cmd

import (
	"fmt"

	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/forge"
	gitscanner "github.com/rlespinasse/ihatt/internal/git"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

// forgeSyncFunc syncs issues, PRs/MRs and CI runs of one repo and returns their counts.
type forgeSyncFunc func(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, int, int, error)

// runForgeSync is the sync loop shared by `github sync` and `gitlab sync`.
// parseRemote reports ok=false for remotes that belong to another forge.
func runForgeSync(cmd *cobra.Command, forgeLabel string, parseRemote func(string) (string, string, error), sync forgeSyncFunc) error {
	repoFilter, _ := cmd.Flags().GetString("repo")
	opts := forge.Options{}
	opts.Full, _ = cmd.Flags().GetBool("full")
	opts.Comments, _ = cmd.Flags().GetBool("with-comments")

	s, err := store.OpenBoltStore(config.DBPath())
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()

	repos, err := s.ListRepos()
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories tracked; use 'ihatt scan' or 'ihatt repo add' first")
	}

	for _, repo := range repos {
		if repoFilter != "" && repo.Name != repoFilter && repo.ID != repoFilter {
			continue
		}

		remoteURL, err := gitscanner.GetRemoteURL(repo.Path)
		if err != nil {
			fmt.Printf("  %-40s SKIP (no origin remote)\n", repo.Name)
			continue
		}

		owner, repoName, err := parseRemote(remoteURL)
		if err != nil {
			fmt.Printf("  %-40s SKIP (not a %s repo)\n", repo.Name, forgeLabel)
			continue
		}

		fmt.Printf("  %-40s syncing %s/%s...\n", repo.Name, owner, repoName)
		issues, prs, runs, err := sync(s, repo.ID, owner, repoName, opts)
		if err != nil {
			fmt.Printf("  %-40s ERROR: %v\n", repo.Name, err)
			continue
		}
		fmt.Printf("  %-40s %d issues, %d %s, %d CI runs\n", repo.Name, issues, prs, prNoun(forgeLabel), runs)
	}

	return nil
}

func prNoun(forgeLabel string) string {
	if forgeLabel == "GitLab" {
		return "MRs"
	}
	return "PRs"
}

// addForgeSyncFlags registers the flags shared by both sync commands.
func addForgeSyncFlags(c *cobra.Command) {
	c.Flags().String("repo", "", "Sync only this repository (by name)")
	c.Flags().Bool("full", false, "Ignore the last sync watermark and fetch everything")
	c.Flags().Bool("with-comments", false, "Also fetch comments (slower: one request per changed item)")
}
