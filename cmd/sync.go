package cmd

import (
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Index commits and sync GitHub and GitLab data for the repos in scope",
	Long: `Index the commits of the repos in scope, then sync their issues, PRs/MRs and CI
runs from GitHub and GitLab. From inside a tracked repo, only that repo is in scope;
use --repo to pick one or --all for every tracked repo.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSync(cmd, true, "github", "gitlab")
	},
}

func init() {
	addForgeSyncFlags(syncCmd)
	rootCmd.AddCommand(syncCmd)
}
