package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"
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

type ghIssue struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Author    ghAuthor   `json:"author"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	ClosedAt  *time.Time `json:"closedAt"`
	Labels    ghLabels   `json:"labels"`
	URL       string     `json:"url"`
}

type ghAuthor struct {
	Login string `json:"login"`
}

type ghLabels struct {
	Nodes []ghLabel `json:"nodes"`
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghPR struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Author    ghAuthor   `json:"author"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	ClosedAt  *time.Time `json:"closedAt"`
	MergedAt  *time.Time `json:"mergedAt"`
	Labels    ghLabels   `json:"labels"`
	URL       string     `json:"url"`
}

// SyncRepo fetches issues and PRs for a GitHub repo and stores them.
func SyncRepo(s store.Store, repoID, owner, repoName string) (int, int, error) {
	issueCount, err := syncIssues(s, repoID, owner, repoName)
	if err != nil {
		return 0, 0, fmt.Errorf("sync issues: %w", err)
	}

	prCount, err := syncPRs(s, repoID, owner, repoName)
	if err != nil {
		return issueCount, 0, fmt.Errorf("sync PRs: %w", err)
	}

	return issueCount, prCount, nil
}

func syncIssues(s store.Store, repoID, owner, repoName string) (int, error) {
	stdout, _, err := gh.Exec("issue", "list",
		"--repo", owner+"/"+repoName,
		"--state", "all",
		"--limit", "100",
		"--json", "number,title,state,author,createdAt,updatedAt,closedAt,labels,url",
	)
	if err != nil {
		return 0, err
	}

	var issues []ghIssue
	if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil {
		return 0, fmt.Errorf("parse issues: %w", err)
	}

	for _, issue := range issues {
		labels := make([]string, len(issue.Labels.Nodes))
		for i, l := range issue.Labels.Nodes {
			labels[i] = l.Name
		}

		item := &model.GitHubItem{
			Number:    issue.Number,
			RepoID:    repoID,
			Owner:     owner,
			RepoName:  repoName,
			Type:      "issue",
			Title:     issue.Title,
			State:     issue.State,
			Author:    issue.Author.Login,
			CreatedAt: issue.CreatedAt,
			UpdatedAt: issue.UpdatedAt,
			ClosedAt:  issue.ClosedAt,
			Labels:    labels,
			URL:       issue.URL,
		}
		if err := s.PutGitHubItem(item); err != nil {
			return 0, err
		}
	}
	return len(issues), nil
}

func syncPRs(s store.Store, repoID, owner, repoName string) (int, error) {
	stdout, _, err := gh.Exec("pr", "list",
		"--repo", owner+"/"+repoName,
		"--state", "all",
		"--limit", "100",
		"--json", "number,title,state,author,createdAt,updatedAt,closedAt,mergedAt,labels,url",
	)
	if err != nil {
		return 0, err
	}

	var prs []ghPR
	if err := json.Unmarshal(stdout.Bytes(), &prs); err != nil {
		return 0, fmt.Errorf("parse PRs: %w", err)
	}

	for _, pr := range prs {
		labels := make([]string, len(pr.Labels.Nodes))
		for i, l := range pr.Labels.Nodes {
			labels[i] = l.Name
		}

		state := strings.ToLower(pr.State)
		if pr.MergedAt != nil {
			state = "merged"
		}

		item := &model.GitHubItem{
			Number:    pr.Number,
			RepoID:    repoID,
			Owner:     owner,
			RepoName:  repoName,
			Type:      "pr",
			Title:     pr.Title,
			State:     state,
			Author:    pr.Author.Login,
			CreatedAt: pr.CreatedAt,
			UpdatedAt: pr.UpdatedAt,
			ClosedAt:  pr.ClosedAt,
			Labels:    labels,
			URL:       pr.URL,
		}
		if err := s.PutGitHubItem(item); err != nil {
			return 0, err
		}
	}
	return len(prs), nil
}
