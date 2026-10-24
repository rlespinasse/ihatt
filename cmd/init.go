package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ihatt configuration and database",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := config.DBPath()
		if _, err := os.Stat(dbPath); err == nil {
			erase, err := confirmErase(cmd.InOrStdin(), cmd.OutOrStdout(), dbPath)
			if err != nil {
				return fmt.Errorf("read answer: %w", err)
			}
			if !erase {
				fmt.Printf("Database kept: %s\n", dbPath)
				return nil
			}
			if err := os.Remove(dbPath); err != nil {
				return fmt.Errorf("erase database: %w", err)
			}
		}

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

		s, err := store.NewBoltStore(dbPath)
		if err != nil {
			return fmt.Errorf("init database: %w", err)
		}
		s.Close()

		fmt.Printf("Initialized ihatt:\n")
		fmt.Printf("  Config: %s\n", config.ConfigPath())
		fmt.Printf("  Data:   %s\n", dbPath)
		return nil
	},
}

// confirmErase asks before replacing an existing database. Anything other
// than an explicit yes, including an empty answer, keeps the database.
func confirmErase(in io.Reader, out io.Writer, dbPath string) (bool, error) {
	fmt.Fprintf(out, "A database already exists at %s.\nErase it and start empty? [y/N] ", dbPath)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
