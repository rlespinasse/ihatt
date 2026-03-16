package cmd

import (
	"fmt"

	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/rlespinasse/ihatt/internal/xref"
	"github.com/spf13/cobra"
)

var xrefCmd = &cobra.Command{
	Use:   "xref",
	Short: "Analyze commits for cross-references between projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoFilter, _ := cmd.Flags().GetString("repo")

		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		repos, err := s.ListRepos()
		if err != nil {
			return err
		}

		if len(repos) == 0 {
			return fmt.Errorf("no repositories tracked")
		}

		totalXrefs := 0
		for _, repo := range repos {
			if repoFilter != "" && repo.Name != repoFilter && repo.ID != repoFilter {
				continue
			}

			count, err := xref.AnalyzeRepo(s, repo.ID, repos)
			if err != nil {
				fmt.Printf("  %-40s ERROR: %v\n", repo.Name, err)
				continue
			}
			if count > 0 {
				fmt.Printf("  %-40s %d cross-references\n", repo.Name, count)
			}
			totalXrefs += count
		}

		fmt.Printf("\nFound %d cross-references\n", totalXrefs)
		return nil
	},
}

func init() {
	xrefCmd.Flags().String("repo", "", "Analyze only this repository")
	rootCmd.AddCommand(xrefCmd)
}
