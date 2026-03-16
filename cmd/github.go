package cmd

import (
	"fmt"

	"github.com/rlespinasse/ihatt/internal/config"
	gitscanner "github.com/rlespinasse/ihatt/internal/git"
	ghclient "github.com/rlespinasse/ihatt/internal/github"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub integration commands",
}

var githubSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch issues and PRs from GitHub for tracked repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoFilter, _ := cmd.Flags().GetString("repo")

		s, err := store.NewBoltStore(config.DBPath())
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

			owner, repoName, err := ghclient.ParseGitHubRemote(remoteURL)
			if err != nil {
				fmt.Printf("  %-40s SKIP (not a GitHub repo)\n", repo.Name)
				continue
			}

			fmt.Printf("  %-40s syncing %s/%s...\n", repo.Name, owner, repoName)
			issues, prs, err := ghclient.SyncRepo(s, repo.ID, owner, repoName)
			if err != nil {
				fmt.Printf("  %-40s ERROR: %v\n", repo.Name, err)
				continue
			}
			fmt.Printf("  %-40s %d issues, %d PRs\n", repo.Name, issues, prs)
		}

		return nil
	},
}

func init() {
	githubSyncCmd.Flags().String("repo", "", "Sync only this repository (by name)")
	githubCmd.AddCommand(githubSyncCmd)
	rootCmd.AddCommand(githubCmd)
}
