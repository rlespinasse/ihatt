package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "ihatt",
	Short: "It Happens At That Time — temporal knowledge base over git repositories",
	Long: `ihatt is a CLI/TUI tool that helps you understand what happened across
your local git repositories at any point in time.

Answer questions like:
  - What was I working on last Tuesday?
  - What changed across all my projects this week?
  - Show me everything that happened on March 5th.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
}
