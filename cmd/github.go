package cmd

import (
	"github.com/rlespinasse/ihatt/internal/forge"
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
		return runForgeSync(cmd, "GitHub", ghclient.ParseGitHubRemote,
			func(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, int, int, error) {
				return ghclient.SyncRepo(s, repoID, owner, repoName, opts)
			})
	},
}

func init() {
	addForgeSyncFlags(githubSyncCmd)
	githubCmd.AddCommand(githubSyncCmd)
	rootCmd.AddCommand(githubCmd)
}
