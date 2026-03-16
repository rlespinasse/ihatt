package cmd

import (
	"fmt"
	"time"

	"github.com/araddon/dateparse"
	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/query"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var atCmd = &cobra.Command{
	Use:   "at <date>",
	Short: "Show what happened at a specific date across all tracked repos",
	Long:  `Show commits and activity for a specific date. Accepts various date formats like "2024-03-15", "march 5", etc.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dateStr := args[0]
		// Join multiple args in case of "march 5 2024"
		if len(args) > 1 {
			for _, a := range args[1:] {
				dateStr += " " + a
			}
		}

		date, err := dateparse.ParseLocal(dateStr)
		if err != nil {
			return fmt.Errorf("cannot parse date %q: %w", dateStr, err)
		}

		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		result, err := query.ForDate(s, date)
		if err != nil {
			return err
		}

		title := date.Format(time.DateOnly)
		query.PrintResult(s, result, title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(atCmd)
}
