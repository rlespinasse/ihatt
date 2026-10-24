package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"
	"github.com/rlespinasse/ihatt/internal/forge"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

// ParseGitHubRemote extracts owner and repo from a GitHub remote URL.
func ParseGitHubRemote(remoteURL string) (owner, repo string, err error) {
	// Handle SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		path := strings.TrimPrefix(remoteURL, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid GitHub SSH URL: %s", remoteURL)
		}
		return parts[0], parts[1], nil
	}

	// Handle HTTPS URLs: https://github.com/owner/repo.git
	u, err := url.Parse(remoteURL)
	if err != nil {
		return "", "", err
	}
	if u.Host != "github.com" {
		return "", "", fmt.Errorf("not a GitHub URL: %s", remoteURL)
	}
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid GitHub URL path: %s", remoteURL)
	}
	return parts[0], parts[1], nil
}

type ghAuthor struct {
	Login string `json:"login"`
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghMilestone struct {
	Title string `json:"title"`
}

type ghIssue struct {
	Number    int               `json:"number"`
	Title     string            `json:"title"`
	State     string            `json:"state"`
	Author    ghAuthor          `json:"author"`
	Assignees []ghAuthor        `json:"assignees"`
	Milestone *ghMilestone      `json:"milestone"`
	Comments  []json.RawMessage `json:"comments"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	ClosedAt  *time.Time        `json:"closedAt"`
	Labels    []ghLabel         `json:"labels"`
	URL       string            `json:"url"`
}

type ghPR struct {
	Number         int               `json:"number"`
	Title          string            `json:"title"`
	State          string            `json:"state"`
	IsDraft        bool              `json:"isDraft"`
	Author         ghAuthor          `json:"author"`
	Assignees      []ghAuthor        `json:"assignees"`
	Milestone      *ghMilestone      `json:"milestone"`
	Comments       []json.RawMessage `json:"comments"`
	ReviewDecision string            `json:"reviewDecision"`
	MergeCommit    *struct {
		Oid string `json:"oid"`
	} `json:"mergeCommit"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	ClosedAt  *time.Time `json:"closedAt"`
	MergedAt  *time.Time `json:"mergedAt"`
	Labels    []ghLabel  `json:"labels"`
	URL       string     `json:"url"`
}

// SyncRepo fetches issues and PRs for a GitHub repo and stores them.
// A full sync fetches everything; otherwise only items updated since the last sync.
func SyncRepo(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, int, int, error) {
	issueCount, err := syncIssues(s, repoID, owner, repoName, opts)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sync issues: %w", err)
	}

	prCount, err := syncPRs(s, repoID, owner, repoName, opts)
	if err != nil {
		return issueCount, 0, 0, fmt.Errorf("sync PRs: %w", err)
	}

	ciCount, err := syncCI(s, repoID, owner, repoName, opts)
	if err != nil {
		return issueCount, prCount, 0, fmt.Errorf("sync CI: %w", err)
	}

	return issueCount, prCount, ciCount, nil
}

// listArgs builds the gh list arguments for a sync plan.
func listArgs(kind string, owner, repoName string, plan forge.Plan, fields string) []string {
	args := []string{kind, "list",
		"--repo", owner + "/" + repoName,
		"--state", "all",
		"--limit", strconv.Itoa(listLimit),
		"--json", fields,
	}
	if plan.Incremental() {
		args = append(args, "--search", "updated:>="+plan.Since.Format(time.RFC3339))
	}
	return args
}

// listLimit caps one gh list call. gh pages internally up to this many items.
const listLimit = 10000

func syncIssues(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, error) {
	plan, err := forge.Start(s, repoID, "github", "issue", opts.Full)
	if err != nil {
		return 0, err
	}

	stdout, _, err := gh.Exec(listArgs("issue", owner, repoName, plan,
		"number,title,state,author,assignees,milestone,comments,createdAt,updatedAt,closedAt,labels,url")...)
	if err != nil {
		return 0, err
	}

	var issues []ghIssue
	if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil {
		return 0, fmt.Errorf("parse issues: %w", err)
	}

	for _, issue := range issues {
		item := &model.GitHubItem{
			Number:    issue.Number,
			RepoID:    repoID,
			Owner:     owner,
			RepoName:  repoName,
			Type:      "issue",
			Title:     issue.Title,
			State:     issue.State,
			Author:    issue.Author.Login,
			Assignees: logins(issue.Assignees),
			Milestone: milestoneTitle(issue.Milestone),
			Comments:  len(issue.Comments),
			CreatedAt: issue.CreatedAt,
			UpdatedAt: issue.UpdatedAt,
			ClosedAt:  issue.ClosedAt,
			Labels:    labelNames(issue.Labels),
			URL:       issue.URL,
		}
		if err := s.PutGitHubItem(item); err != nil {
			return 0, err
		}
		if opts.Comments && plan.UpdatedSince(issue.UpdatedAt) {
			if err := syncGitHubComments(s, repoID, owner, repoName, "issue", issue.Number); err != nil {
				plan.CommentFailed("issue", issue.Number, err)
			}
		}
	}

	return len(issues), plan.Done(s)
}

func syncPRs(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, error) {
	plan, err := forge.Start(s, repoID, "github", "pr", opts.Full)
	if err != nil {
		return 0, err
	}

	stdout, _, err := gh.Exec(listArgs("pr", owner, repoName, plan,
		"number,title,state,isDraft,author,assignees,milestone,comments,reviewDecision,mergeCommit,createdAt,updatedAt,closedAt,mergedAt,labels,url")...)
	if err != nil {
		return 0, err
	}

	var prs []ghPR
	if err := json.Unmarshal(stdout.Bytes(), &prs); err != nil {
		return 0, fmt.Errorf("parse PRs: %w", err)
	}

	for _, pr := range prs {
		state := strings.ToLower(pr.State)
		if pr.MergedAt != nil {
			state = "merged"
		}

		item := &model.GitHubItem{
			Number:         pr.Number,
			RepoID:         repoID,
			Owner:          owner,
			RepoName:       repoName,
			Type:           "pr",
			Title:          pr.Title,
			State:          state,
			Draft:          pr.IsDraft,
			Author:         pr.Author.Login,
			Assignees:      logins(pr.Assignees),
			Milestone:      milestoneTitle(pr.Milestone),
			Comments:       len(pr.Comments),
			ReviewDecision: pr.ReviewDecision,
			MergeCommit:    mergeOid(pr.MergeCommit),
			CreatedAt:      pr.CreatedAt,
			UpdatedAt:      pr.UpdatedAt,
			ClosedAt:       pr.ClosedAt,
			Labels:         labelNames(pr.Labels),
			URL:            pr.URL,
		}
		if err := s.PutGitHubItem(item); err != nil {
			return 0, err
		}
		if opts.Comments && plan.UpdatedSince(pr.UpdatedAt) {
			if err := syncGitHubComments(s, repoID, owner, repoName, "pr", pr.Number); err != nil {
				plan.CommentFailed("pr", pr.Number, err)
			}
		}
	}

	return len(prs), plan.Done(s)
}

func logins(authors []ghAuthor) []string {
	out := make([]string, len(authors))
	for i, a := range authors {
		out[i] = a.Login
	}
	return out
}

func labelNames(labels []ghLabel) []string {
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = l.Name
	}
	return out
}

func milestoneTitle(m *ghMilestone) string {
	if m == nil {
		return ""
	}
	return m.Title
}

func mergeOid(c *struct {
	Oid string `json:"oid"`
}) string {
	if c == nil {
		return ""
	}
	return c.Oid
}

type ghRun struct {
	DatabaseID   int64     `json:"databaseId"`
	HeadSha      string    `json:"headSha"`
	WorkflowName string    `json:"workflowName"`
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	CreatedAt    time.Time `json:"createdAt"`
	URL          string    `json:"url"`
}

// syncCI fetches the repo's recent workflow runs, keyed by commit.
func syncCI(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, error) {
	plan, err := forge.Start(s, repoID, "github", "ci", opts.Full)
	if err != nil {
		return 0, err
	}

	args := []string{"run", "list",
		"--repo", owner + "/" + repoName,
		"--limit", strconv.Itoa(listLimit),
		"--json", "databaseId,headSha,workflowName,status,conclusion,createdAt,url",
	}
	if plan.Incremental() {
		// gh's --created takes dates only, so this may re-fetch a day of runs.
		args = append(args, "--created", ">="+plan.Since.Format("2006-01-02"))
	}
	stdout, _, err := gh.Exec(args...)
	if err != nil {
		return 0, err
	}

	var runs []ghRun
	if err := json.Unmarshal(stdout.Bytes(), &runs); err != nil {
		return 0, fmt.Errorf("parse runs: %w", err)
	}

	for _, run := range runs {
		status := run.Status
		if run.Conclusion != "" {
			status = run.Conclusion
		}
		item := &model.CIRun{
			RepoID: repoID,
			ID:     strconv.FormatInt(run.DatabaseID, 10),
			Commit: run.HeadSha,
			Name:   run.WorkflowName,
			Status: strings.ToLower(status),
			Date:   run.CreatedAt,
			URL:    run.URL,
		}
		if err := s.PutCIRun(item); err != nil {
			return 0, err
		}
	}
	return len(runs), plan.Done(s)
}

type ghComment struct {
	ID        int64     `json:"id"`
	User      ghAuthor  `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	HTMLURL   string    `json:"html_url"`
}

// syncGitHubComments stores the conversation comments of one issue or PR.
// PRs share the issues comments endpoint, so review comments are not included.
func syncGitHubComments(s store.Store, repoID, owner, repoName, itemType string, number int) error {
	stdout, _, err := gh.Exec("api", "--paginate",
		fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repoName, number))
	if err != nil {
		return err
	}

	comments, err := forge.DecodePages[ghComment](stdout.Bytes())
	if err != nil {
		return err
	}
	for _, c := range comments {
		comment := &model.Comment{
			RepoID:     repoID,
			ItemType:   itemType,
			ItemNumber: number,
			ID:         strconv.FormatInt(c.ID, 10),
			Author:     c.User.Login,
			Body:       c.Body,
			Date:       c.CreatedAt,
			URL:        c.HTMLURL,
		}
		if err := s.PutComment(comment); err != nil {
			return err
		}
	}
	return nil
}
