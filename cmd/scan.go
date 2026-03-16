package cmd

import (
	"fmt"

	"github.com/rlespinasse/ihatt/internal/config"
	gitscanner "github.com/rlespinasse/ihatt/internal/git"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan directories for git repositories and index their commits",
	RunE: func(cmd *cobra.Command, args []string) error {
		roots, _ := cmd.Flags().GetStringSlice("root")
		maxDepth, _ := cmd.Flags().GetInt("depth")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if len(roots) == 0 {
			roots = cfg.ScanRoots
		}
		if len(roots) == 0 {
			return fmt.Errorf("no scan roots specified; use --root or configure scan_roots in config")
		}

		// Save roots to config for future use
		if len(cfg.ScanRoots) == 0 {
			cfg.ScanRoots = roots
			config.Save(cfg)
		}

		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		fmt.Printf("Scanning %d root(s) for git repositories...\n", len(roots))
		repoPaths, err := gitscanner.DiscoverRepos(roots, maxDepth)
		if err != nil {
			return fmt.Errorf("discover repos: %w", err)
		}
		fmt.Printf("Found %d repositories\n\n", len(repoPaths))

		totalCommits := 0
		for _, path := range repoPaths {
			repo := &model.Repository{
				ID:     gitscanner.RepoID(path),
				Name:   gitscanner.RepoName(path),
				Path:   path,
				Active: true,
			}

			count, err := gitscanner.IndexRepo(s, repo)
			if err != nil {
				fmt.Printf("  %-40s ERROR: %v\n", repo.Name, err)
				continue
			}
			fmt.Printf("  %-40s %d commits\n", repo.Name, count)
			totalCommits += count
		}

		fmt.Printf("\nIndexed %d commits across %d repositories\n", totalCommits, len(repoPaths))
		return nil
	},
}

func init() {
	scanCmd.Flags().StringSlice("root", nil, "Root directories to scan for git repos")
	scanCmd.Flags().Int("depth", 3, "Maximum directory depth to scan")
	rootCmd.AddCommand(scanCmd)
}
