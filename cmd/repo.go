package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/rlespinasse/ihatt/internal/config"
	gitscanner "github.com/rlespinasse/ihatt/internal/git"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage tracked repositories",
}

var repoAddCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Add a git repository to track",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}

		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		repo := &model.Repository{
			ID:     gitscanner.RepoID(path),
			Name:   gitscanner.RepoName(path),
			Path:   path,
			Active: true,
		}

		if err := s.PutRepo(repo); err != nil {
			return err
		}

		count, err := gitscanner.IndexRepo(s, repo)
		if err != nil {
			fmt.Printf("Added %s (indexing failed: %v)\n", repo.Name, err)
			return nil
		}

		fmt.Printf("Added %s (%d commits indexed)\n", repo.Name, count)
		return nil
	},
}

var repoRemoveCmd = &cobra.Command{
	Use:   "remove <path-or-name>",
	Short: "Remove a tracked repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		repos, err := s.ListRepos()
		if err != nil {
			return err
		}

		target := args[0]
		for _, repo := range repos {
			if repo.Name == target || repo.Path == target || repo.ID == target {
				if err := s.DeleteRepo(repo.ID); err != nil {
					return err
				}
				fmt.Printf("Removed %s (%s)\n", repo.Name, repo.Path)
				return nil
			}
		}
		return fmt.Errorf("repository not found: %s", target)
	},
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracked repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
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
			fmt.Println("No repositories tracked. Use 'ihatt repo add <path>' or 'ihatt scan' to add repos.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "NAME\tPATH\tCOMMITS\tLAST SCAN\n")
		for _, repo := range repos {
			count, _ := s.CountCommitsByRepo(repo.ID)
			lastScan := "never"
			if !repo.LastScan.IsZero() {
				lastScan = repo.LastScan.Format("2006-01-02 15:04")
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", repo.Name, repo.Path, count, lastScan)
		}
		w.Flush()
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoRemoveCmd)
	repoCmd.AddCommand(repoListCmd)
	rootCmd.AddCommand(repoCmd)
}
