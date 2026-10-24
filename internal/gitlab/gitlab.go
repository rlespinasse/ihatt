package gitlab

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/rlespinasse/ihatt/internal/forge"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

const provider = "gitlab"

// perPage is the GitLab API page size; the maximum allowed is 100.
const perPage = 100

// ParseGitLabRemote extracts the namespace (owner, possibly nested groups) and
// project name from a gitlab.com remote URL.
func ParseGitLabRemote(remoteURL string) (owner, repo string, err error) {
	var path string
	if after, ok := strings.CutPrefix(remoteURL, "git@gitlab.com:"); ok {
		path = after
	} else {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", "", err
		}
		if u.Hostname() != "gitlab.com" {
			return "", "", fmt.Errorf("not a GitLab URL: %s", remoteURL)
		}
		path = u.Path
	}

	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	i := strings.LastIndex(path, "/")
	if i <= 0 || i == len(path)-1 {
		return "", "", fmt.Errorf("invalid GitLab URL path: %s", remoteURL)
	}
	return path[:i], path[i+1:], nil
}

type glUser struct {
	Username string `json:"username"`
}

type glMilestone struct {
	Title string `json:"title"`
}

type glIssue struct {
	IID            int          `json:"iid"`
	Title          string       `json:"title"`
	State          string       `json:"state"`
	Author         glUser       `json:"author"`
	Assignees      []glUser     `json:"assignees"`
	Milestone      *glMilestone `json:"milestone"`
	UserNotesCount int          `json:"user_notes_count"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	ClosedAt       *time.Time   `json:"closed_at"`
	Labels         []string     `json:"labels"`
	WebURL         string       `json:"web_url"`
}

type glMR struct {
	IID            int          `json:"iid"`
	Title          string       `json:"title"`
	State          string       `json:"state"`
	Draft          bool         `json:"draft"`
	Author         glUser       `json:"author"`
	Assignees      []glUser     `json:"assignees"`
	Milestone      *glMilestone `json:"milestone"`
	UserNotesCount int          `json:"user_notes_count"`
	MergeCommitSHA *string      `json:"merge_commit_sha"`
	SquashSHA      *string      `json:"squash_commit_sha"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	ClosedAt       *time.Time   `json:"closed_at"`
	MergedAt       *time.Time   `json:"merged_at"`
	Labels         []string     `json:"labels"`
	WebURL         string       `json:"web_url"`
}

// SyncRepo fetches issues, merge requests and pipelines for a GitLab project.
// A full sync fetches everything; otherwise only items updated since the last sync.
func SyncRepo(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, int, int, error) {
	if _, err := exec.LookPath("glab"); err != nil {
		return 0, 0, 0, fmt.Errorf("glab not found in PATH; install it and run 'glab auth login'")
	}

	issueCount, err := syncIssues(s, repoID, owner, repoName, opts)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sync issues: %w", err)
	}

	mrCount, err := syncMRs(s, repoID, owner, repoName, opts)
	if err != nil {
		return issueCount, 0, 0, fmt.Errorf("sync merge requests: %w", err)
	}

	ciCount, err := syncPipelines(s, repoID, owner, repoName, opts)
	if err != nil {
		return issueCount, mrCount, 0, fmt.Errorf("sync pipelines: %w", err)
	}

	return issueCount, mrCount, ciCount, nil
}

type glPipeline struct {
	ID        int64     `json:"id"`
	SHA       string    `json:"sha"`
	Ref       string    `json:"ref"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
	WebURL    string    `json:"web_url"`
}

// syncPipelines stores the project's pipelines as CI runs, keyed by commit.
func syncPipelines(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, error) {
	plan, err := forge.Start(s, repoID, provider, "ci", opts.Full)
	if err != nil {
		return 0, err
	}

	pipelines, err := fetchAll[glPipeline](projectPath(owner, repoName, "pipelines", plan))
	if err != nil {
		return 0, err
	}

	for _, p := range pipelines {
		run := &model.CIRun{
			RepoID:   repoID,
			Provider: provider,
			ID:       fmt.Sprint(p.ID),
			Commit:   p.SHA,
			Name:     p.Ref,
			Status:   normalizePipelineStatus(p.Status),
			Date:     p.UpdatedAt,
			URL:      p.WebURL,
		}
		if err := s.PutCIRun(run); err != nil {
			return 0, err
		}
	}
	return len(pipelines), plan.Done(s)
}

// normalizePipelineStatus maps GitLab pipeline statuses to the names GitHub uses.
func normalizePipelineStatus(status string) string {
	switch status {
	case "failed":
		return "failure"
	case "canceled":
		return "cancelled"
	default:
		return status
	}
}

func syncIssues(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, error) {
	plan, err := forge.Start(s, repoID, provider, "issue", opts.Full)
	if err != nil {
		return 0, err
	}

	issues, err := fetchAll[glIssue](projectPath(owner, repoName, "issues", plan))
	if err != nil {
		return 0, err
	}

	for _, issue := range issues {
		item := &model.GitHubItem{
			Number:    issue.IID,
			RepoID:    repoID,
			Provider:  provider,
			Owner:     owner,
			RepoName:  repoName,
			Type:      "issue",
			Title:     issue.Title,
			State:     normalizeState(issue.State),
			Author:    issue.Author.Username,
			Assignees: usernames(issue.Assignees),
			Milestone: milestoneTitle(issue.Milestone),
			Comments:  issue.UserNotesCount,
			CreatedAt: issue.CreatedAt,
			UpdatedAt: issue.UpdatedAt,
			ClosedAt:  issue.ClosedAt,
			Labels:    issue.Labels,
			URL:       issue.WebURL,
		}
		if err := s.PutGitHubItem(item); err != nil {
			return 0, err
		}
		if opts.Comments && plan.UpdatedSince(issue.UpdatedAt) {
			if err := syncNotes(s, repoID, owner, repoName, "issues", "issue", issue.IID); err != nil {
				plan.CommentFailed("issue", issue.IID, err)
			}
		}
	}

	return len(issues), plan.Done(s)
}

func syncMRs(s store.Store, repoID, owner, repoName string, opts forge.Options) (int, error) {
	plan, err := forge.Start(s, repoID, provider, "pr", opts.Full)
	if err != nil {
		return 0, err
	}

	mrs, err := fetchAll[glMR](projectPath(owner, repoName, "merge_requests", plan))
	if err != nil {
		return 0, err
	}

	for _, mr := range mrs {
		state := normalizeState(mr.State)
		if mr.MergedAt != nil {
			state = "merged"
		}

		item := &model.GitHubItem{
			Number:      mr.IID,
			RepoID:      repoID,
			Provider:    provider,
			Owner:       owner,
			RepoName:    repoName,
			Type:        "pr",
			Title:       mr.Title,
			State:       state,
			Draft:       mr.Draft,
			Author:      mr.Author.Username,
			Assignees:   usernames(mr.Assignees),
			Milestone:   milestoneTitle(mr.Milestone),
			Comments:    mr.UserNotesCount,
			MergeCommit: mergeSHA(mr),
			CreatedAt:   mr.CreatedAt,
			UpdatedAt:   mr.UpdatedAt,
			ClosedAt:    mr.ClosedAt,
			Labels:      mr.Labels,
			URL:         mr.WebURL,
		}
		if err := s.PutGitHubItem(item); err != nil {
			return 0, err
		}
		if opts.Comments && plan.UpdatedSince(mr.UpdatedAt) {
			if err := syncNotes(s, repoID, owner, repoName, "merge_requests", "pr", mr.IID); err != nil {
				plan.CommentFailed("pr", mr.IID, err)
			}
		}
	}

	return len(mrs), plan.Done(s)
}

// projectPath builds the API path for a kind of project resource, with the
// project path URL-encoded as GitLab requires.
func projectPath(owner, repoName, resource string, plan forge.Plan) string {
	q := url.Values{}
	q.Set("state", "all")
	q.Set("per_page", fmt.Sprint(perPage))
	q.Set("order_by", "updated_at")
	q.Set("sort", "asc")
	if plan.Incremental() {
		q.Set("updated_after", plan.Since.Format(time.RFC3339))
	}
	return "projects/" + url.PathEscape(owner+"/"+repoName) + "/" + resource + "?" + q.Encode()
}

// fetchAll pages through a GitLab list endpoint and returns every item.
func fetchAll[T any](path string) ([]T, error) {
	stdout, err := glab("api", "--paginate", path)
	if err != nil {
		return nil, err
	}
	return decodePages[T](stdout)
}

// decodePages reads the JSON arrays that --paginate prints back to back.
func decodePages[T any](data []byte) ([]T, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var all []T
	for {
		var page []T
		err := dec.Decode(&page)
		if errors.Is(err, io.EOF) {
			return all, nil
		}
		if err != nil {
			return nil, fmt.Errorf("parse page: %w", err)
		}
		all = append(all, page...)
	}
}

// mergeSHA returns the SHA that landed the MR on the target branch: the squash
// commit when squashed, otherwise the merge commit.
func mergeSHA(mr glMR) string {
	if mr.SquashSHA != nil && *mr.SquashSHA != "" {
		return *mr.SquashSHA
	}
	if mr.MergeCommitSHA != nil {
		return *mr.MergeCommitSHA
	}
	return ""
}

func usernames(users []glUser) []string {
	out := make([]string, len(users))
	for i, u := range users {
		out[i] = u.Username
	}
	return out
}

func milestoneTitle(m *glMilestone) string {
	if m == nil {
		return ""
	}
	return m.Title
}

// normalizeState maps GitLab states (opened, closed, merged, locked) to ihatt's
// lowercase vocabulary (open, closed, merged).
func normalizeState(state string) string {
	state = strings.ToLower(state)
	if state == "opened" {
		return "open"
	}
	return state
}

// glab runs the glab CLI and returns its stdout. Stderr is folded into the error.
func glab(args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("glab", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("glab %s: %w: %s", strings.Join(args[:min(len(args), 2)], " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

type glNote struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Author    glUser    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	System    bool      `json:"system"`
}

// syncNotes stores the user comments of one issue or MR. GitLab's "notes" are
// comments; system notes (label changes, pushes, ...) are skipped.
func syncNotes(s store.Store, repoID, owner, repoName, resource, itemType string, number int) error {
	path := fmt.Sprintf("projects/%s/%s/%d/notes?per_page=%d&sort=asc",
		url.PathEscape(owner+"/"+repoName), resource, number, perPage)
	notes, err := fetchAll[glNote](path)
	if err != nil {
		return err
	}
	for _, n := range notes {
		if n.System {
			continue
		}
		comment := &model.Comment{
			RepoID:     repoID,
			Provider:   provider,
			ItemType:   itemType,
			ItemNumber: number,
			ID:         fmt.Sprint(n.ID),
			Author:     n.Author.Username,
			Body:       n.Body,
			Date:       n.CreatedAt,
		}
		if err := s.PutComment(comment); err != nil {
			return err
		}
	}
	return nil
}
