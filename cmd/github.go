package cmd

import (
	"github.com/spf13/cobra"
)

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub integration commands",
}

var githubSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch issues, PRs and CI runs from GitHub for the repos in scope",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSync(cmd, false, "github")
	},
}

func init() {
	addForgeSyncFlags(githubSyncCmd)
	githubCmd.AddCommand(githubSyncCmd)
	rootCmd.AddCommand(githubCmd)
}
