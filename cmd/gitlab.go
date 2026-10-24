package cmd

import (
	"github.com/rlespinasse/ihatt/internal/forge"
	"github.com/rlespinasse/ihatt/internal/gitlab"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var gitlabCmd = &cobra.Command{
	Use:   "gitlab",
	Short: "GitLab integration commands",
}

var gitlabSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch issues and merge requests from GitLab for tracked repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runForgeSync(cmd, "GitLab", gitlab.ParseGitLabRemote,
			func(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, int, int, error) {
				return gitlab.SyncRepo(s, repoID, owner, repoName, opts)
			})
	},
}

func init() {
	addForgeSyncFlags(gitlabSyncCmd)
	gitlabCmd.AddCommand(gitlabSyncCmd)
	rootCmd.AddCommand(gitlabCmd)
}
