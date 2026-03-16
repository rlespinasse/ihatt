package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/araddon/dateparse"
	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/store"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search commit messages, authors, and files across repos",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q := strings.Join(args, " ")
		author, _ := cmd.Flags().GetString("author")
		repo, _ := cmd.Flags().GetString("repo")
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")

		s, err := store.NewBoltStore(config.DBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer s.Close()

		var fromTime, toTime *time.Time
		if since != "" {
			t, err := dateparse.ParseLocal(since)
			if err != nil {
				return fmt.Errorf("cannot parse --since %q: %w", since, err)
			}
			fromTime = &t
		}
		if until != "" {
			t, err := dateparse.ParseLocal(until)
			if err != nil {
				return fmt.Errorf("cannot parse --until %q: %w", until, err)
			}
			toTime = &t
		}

		// Resolve repo name to ID if provided
		repoID := ""
		if repo != "" {
			repos, _ := s.ListRepos()
			for _, r := range repos {
				if r.Name == repo || r.ID == repo {
					repoID = r.ID
					break
				}
			}
			if repoID == "" {
				return fmt.Errorf("repository not found: %s", repo)
			}
		}

		commits, err := s.SearchCommits(q, fromTime, toTime, author, repoID)
		if err != nil {
			return err
		}

		if len(commits) == 0 {
			fmt.Printf("No results for %q\n", q)
			return nil
		}

		// Build repo name map
		repoNames := make(map[string]string)
		repos, _ := s.ListRepos()
		for _, r := range repos {
			repoNames[r.ID] = r.Name
		}

		fmt.Printf("Found %d result(s) for %q\n\n", len(commits), q)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "DATE\tREPO\tHASH\tAUTHOR\tMESSAGE\n")
		for _, c := range commits {
			msg := c.Message
			if idx := strings.Index(msg, "\n"); idx != -1 {
				msg = msg[:idx]
			}
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
			name := repoNames[c.RepoID]
			if name == "" {
				name = c.RepoID[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				c.Date.Format("2006-01-02 15:04"),
				name,
				c.Hash[:8],
				c.Author,
				msg,
			)
		}
		w.Flush()
		return nil
	},
}

func init() {
	searchCmd.Flags().String("author", "", "Filter by author name")
	searchCmd.Flags().String("repo", "", "Filter by repository name")
	searchCmd.Flags().String("since", "", "Show results after this date")
	searchCmd.Flags().String("until", "", "Show results before this date")
	rootCmd.AddCommand(searchCmd)
}
