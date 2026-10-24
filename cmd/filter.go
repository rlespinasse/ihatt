package cmd

import (
	"github.com/rlespinasse/ihatt/internal/query"
)

// Item filters apply to issues and PRs/MRs in every time-window command.
func init() {
	f := &query.Filter
	rootCmd.PersistentFlags().StringVar(&f.Provider, "provider", "", "Only forge items from this provider: github or gitlab")
	rootCmd.PersistentFlags().StringVar(&f.State, "state", "", "Only forge items in this state: open, closed or merged")
	rootCmd.PersistentFlags().StringVar(&f.Type, "type", "", "Only forge items of this type: issue or pr")
	rootCmd.PersistentFlags().StringVar(&f.Label, "label", "", "Only forge items carrying this label")
}
