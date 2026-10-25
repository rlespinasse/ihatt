package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// isInteractive reports whether both stdin and stdout are terminals.
// It is a variable so tests can simulate a terminal.
var isInteractive = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// addOutputFlags registers the flags that force the interactive TUI or plain text output.
func addOutputFlags(c *cobra.Command) {
	c.Flags().Bool("tui", false, "Force the interactive TUI, even when not on a terminal")
	c.Flags().Bool("no-tui", false, "Force plain text output, even on a terminal")
	c.MarkFlagsMutuallyExclusive("tui", "no-tui")
}

// useTUI picks between the TUI and plain text output. The TUI is used on a terminal
// unless --no-tui is set; --tui forces it and fails when no terminal is attached.
func useTUI(cmd *cobra.Command) (bool, error) {
	forceTUI, _ := cmd.Flags().GetBool("tui")
	forcePlain, _ := cmd.Flags().GetBool("no-tui")

	switch {
	case forcePlain:
		return false, nil
	case forceTUI && !isInteractive():
		return false, fmt.Errorf("--tui needs an interactive terminal on stdin and stdout")
	case forceTUI:
		return true, nil
	default:
		return isInteractive(), nil
	}
}
