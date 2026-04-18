package query

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

type Result struct {
	Commits     []*model.Commit
	GitHubItems []*model.GitHubItem
}

func ByTimeRange(s store.Store, from, to time.Time) (*Result, error) {
	commits, err := s.GetCommitsByTimeRange(from, to)
	if err != nil {
		return nil, err
	}

	items, err := s.GetGitHubItemsByTimeRange(from, to)
	if err != nil {
		// Non-fatal: GitHub items may not be indexed yet
		items = nil
	}

	return &Result{Commits: commits, GitHubItems: items}, nil
}

func ForDate(s store.Store, date time.Time) (*Result, error) {
	from := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	to := from.Add(24*time.Hour - time.Second)
	return ByTimeRange(s, from, to)
}

func Today(s store.Store) (*Result, error) {
	return ForDate(s, time.Now())
}

func Yesterday(s store.Store) (*Result, error) {
	return ForDate(s, time.Now().AddDate(0, 0, -1))
}

func ThisWeek(s store.Store) (*Result, error) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -6)
	to := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	return ByTimeRange(s, from, to)
}

// PrintResult formats and prints query results grouped by repository.
func PrintResult(s store.Store, result *Result, title string) {
	fmt.Printf("=== %s ===\n\n", title)

	if len(result.Commits) == 0 && len(result.GitHubItems) == 0 {
		fmt.Println("No activity found.")
		return
	}

	// Group commits by repo
	byRepo := make(map[string][]*model.Commit)
	for _, c := range result.Commits {
		byRepo[c.RepoID] = append(byRepo[c.RepoID], c)
	}

	// Build repo name map
	repoNames := make(map[string]string)
	repos, _ := s.ListRepos()
	for _, r := range repos {
		repoNames[r.ID] = r.Name
	}

	// Sort repo IDs for consistent output
	repoIDs := make([]string, 0, len(byRepo))
	for id := range byRepo {
		repoIDs = append(repoIDs, id)
	}
	sort.Strings(repoIDs)

	for _, repoID := range repoIDs {
		commits := byRepo[repoID]
		name := repoNames[repoID]
		if name == "" {
			name = repoID
		}

		// Sort commits by date
		sort.Slice(commits, func(i, j int) bool {
			return commits[i].Date.Before(commits[j].Date)
		})

		fmt.Printf("[%s] %d commit(s)\n", name, len(commits))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, c := range commits {
			msg := c.Message
			if idx := strings.Index(msg, "\n"); idx != -1 {
				msg = msg[:idx]
			}
			if len(msg) > 72 {
				msg = msg[:69] + "..."
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
				c.Date.Format("15:04"),
				c.Hash[:8],
				c.Author,
				msg,
			)
		}
		w.Flush()
		fmt.Println()
	}

	// Print GitHub items if any
	if len(result.GitHubItems) > 0 {
		fmt.Printf("--- GitHub Activity ---\n\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "TYPE\tREPO\t#\tSTATE\tTITLE\n")
		for _, item := range result.GitHubItems {
			title := item.Title
			if len(title) > 60 {
				title = title[:57] + "..."
			}
			repoName := repoNames[item.RepoID]
			if repoName == "" {
				repoName = item.Owner + "/" + item.RepoName
			}
			fmt.Fprintf(w, "%s\t%s\t#%d\t%s\t%s\n",
				strings.ToUpper(item.Type),
				repoName,
				item.Number,
				item.State,
				title,
			)
		}
		w.Flush()
		fmt.Println()
	}

	fmt.Printf("Total: %d commits, %d GitHub items\n", len(result.Commits), len(result.GitHubItems))
}
