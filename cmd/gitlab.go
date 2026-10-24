package cmd

import (
	"github.com/spf13/cobra"
)

var gitlabCmd = &cobra.Command{
	Use:   "gitlab",
	Short: "GitLab integration commands",
}

var gitlabSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch issues, MRs and pipelines from GitLab for the repos in scope",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSync(cmd, false, "gitlab")
	},
}

func init() {
	addForgeSyncFlags(gitlabSyncCmd)
	gitlabCmd.AddCommand(gitlabSyncCmd)
	rootCmd.AddCommand(gitlabCmd)
}
