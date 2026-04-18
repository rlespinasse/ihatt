package cmd

import (
	"fmt"

	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ihatt configuration and database",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.EnsureDirs(); err != nil {
			return fmt.Errorf("create directories: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			cfg = &config.Config{}
		}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("init database: %w", err)
		}
		s.Close()

		fmt.Printf("Initialized ihatt:\n")
		fmt.Printf("  Config: %s\n", config.ConfigPath())
		fmt.Printf("  Data:   %s\n", config.DBPath())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
