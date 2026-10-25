package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/araddon/dateparse"
	"github.com/rlespinasse/ihatt/internal/config"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/query"
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
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")

		s, err := store.OpenBoltStore(config.DBPath())
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

		// Resolve the repo scope (--repo, --dir, the current repo or folder, or all) to repo IDs
		repos, _ := s.ListRepos()
		inScope, scope := repoScope(cmd, repos)
		repoIDs := map[string]bool{}
		for _, r := range repos {
			if inScope(r) {
				repoIDs[r.ID] = true
			}
		}
		if name, _ := cmd.Flags().GetString("repo"); name != "" && len(repoIDs) == 0 {
			return fmt.Errorf("repository not found: %s", name)
		}
		fmt.Fprintf(os.Stderr, "Scope: %s\n", scope)

		allCommits, err := s.SearchCommits(q, fromTime, toTime, author, "")
		if err != nil {
			return err
		}
		var commits []*model.Commit
		for _, c := range allCommits {
			if repoIDs[c.RepoID] {
				commits = append(commits, c)
			}
		}

		allComments, err := query.SearchComments(s, q, "")
		if err != nil {
			return err
		}
		var comments []*model.Comment
		for _, c := range allComments {
			if repoIDs[c.RepoID] {
				comments = append(comments, c)
			}
		}

		if len(commits) == 0 && len(comments) == 0 {
			fmt.Printf("No results for %q\n", q)
			return nil
		}

		// Build repo name map
		repoNames := make(map[string]string)
		for _, r := range repos {
			repoNames[r.ID] = r.Name
		}

		if len(commits) > 0 {
			fmt.Printf("Found %d commit(s) for %q\n\n", len(commits), q)
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
		}

		if len(comments) > 0 {
			if len(commits) > 0 {
				fmt.Println()
			}
			fmt.Printf("Found %d comment(s) for %q\n\n", len(comments), q)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "DATE\tREPO\tITEM\tAUTHOR\tCOMMENT\n")
			for _, c := range comments {
				body := strings.Join(strings.Fields(c.Body), " ")
				if len(body) > 60 {
					body = body[:57] + "..."
				}
				name := repoNames[c.RepoID]
				if name == "" {
					name = c.RepoID[:8]
				}
				item := fmt.Sprintf("%s #%d", strings.ToUpper(c.ItemType), c.ItemNumber)
				if c.ItemType == "pr" && c.Provider == "gitlab" {
					item = fmt.Sprintf("MR !%d", c.ItemNumber)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					c.Date.Format("2006-01-02 15:04"), name, item, c.Author, body)
			}
			w.Flush()
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().String("author", "", "Filter by author name")
	addScopeFlags(searchCmd)
	searchCmd.Flags().String("since", "", "Show results after this date")
	searchCmd.Flags().String("until", "", "Show results before this date")
	rootCmd.AddCommand(searchCmd)
}
