package cmd

import (
	"fmt"

	"github.com/araddon/dateparse"
	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/query"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var rangeCmd = &cobra.Command{
	Use:   "range <from> <to>",
	Short: "Show activity within a time range",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		from, err := dateparse.ParseLocal(args[0])
		if err != nil {
			return fmt.Errorf("cannot parse start date %q: %w", args[0], err)
		}
		to, err := dateparse.ParseLocal(args[1])
		if err != nil {
			return fmt.Errorf("cannot parse end date %q: %w", args[1], err)
		}

		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		result, err := query.ByTimeRange(s, from, to)
		if err != nil {
			return err
		}

		title := fmt.Sprintf("%s to %s", from.Format("2006-01-02"), to.Format("2006-01-02"))
		query.PrintResult(s, result, title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rangeCmd)
}
