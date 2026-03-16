package cmd

import (
	"fmt"
	"time"

	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/query"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's activity across all tracked repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		result, err := query.Today(s)
		if err != nil {
			return err
		}

		query.PrintResult(s, result, "Today — "+time.Now().Format(time.DateOnly))
		return nil
	},
}

var yesterdayCmd = &cobra.Command{
	Use:   "yesterday",
	Short: "Show yesterday's activity across all tracked repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		result, err := query.Yesterday(s)
		if err != nil {
			return err
		}

		query.PrintResult(s, result, "Yesterday — "+time.Now().AddDate(0, 0, -1).Format(time.DateOnly))
		return nil
	},
}

var weekCmd = &cobra.Command{
	Use:   "week",
	Short: "Show this week's activity across all tracked repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		result, err := query.ThisWeek(s)
		if err != nil {
			return err
		}

		now := time.Now()
		from := now.AddDate(0, 0, -6)
		title := fmt.Sprintf("This Week — %s to %s", from.Format(time.DateOnly), now.Format(time.DateOnly))
		query.PrintResult(s, result, title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(yesterdayCmd)
	rootCmd.AddCommand(weekCmd)
}
