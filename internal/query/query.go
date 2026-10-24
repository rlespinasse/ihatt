package query

import (
	"fmt"
	"os"
	"slices"
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

	return &Result{Commits: commits, GitHubItems: Filter.Apply(items)}, nil
}

// ItemFilter narrows issues and PRs/MRs. Empty fields match everything.
// Commits are never filtered. The CLI binds its flags to Filter directly.
type ItemFilter struct {
	Provider string // "github" or "gitlab"
	State    string // open, closed, merged
	Type     string // issue or pr
	Label    string
}

// Filter is the active item filter for every time-window query.
var Filter ItemFilter

// Apply returns the items matching the filter.
func (f ItemFilter) Apply(items []*model.GitHubItem) []*model.GitHubItem {
	if f == (ItemFilter{}) {
		return items
	}
	var out []*model.GitHubItem
	for _, item := range items {
		if f.matches(item) {
			out = append(out, item)
		}
	}
	return out
}

func (f ItemFilter) matches(item *model.GitHubItem) bool {
	provider := item.Provider
	if provider == "" {
		provider = "github"
	}
	if f.Provider != "" && !strings.EqualFold(f.Provider, provider) {
		return false
	}
	if f.State != "" && !strings.EqualFold(f.State, item.State) {
		return false
	}
	if f.Type != "" && !strings.EqualFold(f.Type, item.Type) {
		return false
	}
	if f.Label != "" && !slices.ContainsFunc(item.Labels, func(l string) bool {
		return strings.EqualFold(l, f.Label)
	}) {
		return false
	}
	return true
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

	// Index what is known about each commit: the PR/MR that landed it,
	// its tags and its CI runs.
	mergedBy := make(map[string]*model.GitHubItem)
	tagsBy := make(map[string][]string)
	ciBy := make(map[string][]*model.CIRun)
	for _, r := range repos {
		items, _ := s.GetGitHubItemsByRepo(r.ID)
		for _, item := range items {
			if item.MergeCommit != "" {
				mergedBy[item.MergeCommit] = item
			}
		}
		tags, _ := s.GetTagsByRepo(r.ID)
		for _, tag := range tags {
			tagsBy[tag.Commit] = append(tagsBy[tag.Commit], tag.Name)
		}
		runs, _ := s.GetCIRunsByRepo(r.ID)
		for _, run := range runs {
			ciBy[run.Commit] = append(ciBy[run.Commit], run)
		}
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
			if item := mergedBy[c.Hash]; item != nil {
				msg = fmt.Sprintf("%s  [%s #%d]", msg, item.TypeLabel(), item.Number)
			}
			for _, tag := range tagsBy[c.Hash] {
				msg += "  (tag " + tag + ")"
			}
			if runs := ciBy[c.Hash]; len(runs) > 0 {
				msg += "  [ci " + ciSummary(runs) + "]"
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

	// Print issues and PRs/MRs if any
	if len(result.GitHubItems) > 0 {
		fmt.Printf("--- Issues & PRs Activity ---\n\n")
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
				item.TypeLabel(),
				repoName,
				item.Number,
				item.State,
				title,
			)
		}
		w.Flush()
		fmt.Println()
	}

	fmt.Printf("Total: %d commits, %d issues/PRs\n", len(result.Commits), len(result.GitHubItems))
}

// ciSummary reports the worst status among a commit's CI runs.
func ciSummary(runs []*model.CIRun) string {
	for _, run := range runs {
		if run.Status == "failure" || run.Status == "cancelled" || run.Status == "timed_out" {
			return run.Status
		}
	}
	for _, run := range runs {
		if run.Status != "success" {
			return "pending"
		}
	}
	return "success"
}

// SearchComments returns issue and PR/MR comments whose body or author contains
// q (case-insensitive), optionally limited to one repo.
func SearchComments(s store.Store, q, repoID string) ([]*model.Comment, error) {
	repos, err := s.ListRepos()
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(q)
	var out []*model.Comment
	for _, r := range repos {
		if repoID != "" && r.ID != repoID {
			continue
		}
		comments, err := s.GetCommentsByRepo(r.ID)
		if err != nil {
			return nil, err
		}
		for _, c := range comments {
			if strings.Contains(strings.ToLower(c.Body), needle) || strings.Contains(strings.ToLower(c.Author), needle) {
				out = append(out, c)
			}
		}
	}
	return out, nil
}
