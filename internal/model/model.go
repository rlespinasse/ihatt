package model

import (
	"strings"
	"time"
)

type Repository struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	LastScan time.Time `json:"last_scan"`
	Active   bool      `json:"active"`
}

type Commit struct {
	Hash          string    `json:"hash"`
	RepoID        string    `json:"repo_id"`
	Author        string    `json:"author"`
	Email         string    `json:"email"`
	Message       string    `json:"message"`
	Date          time.Time `json:"date"`
	FilesAdded    int       `json:"files_added"`
	FilesModified int       `json:"files_modified"`
	FilesDeleted  int       `json:"files_deleted"`
}

// GitHubItem is an issue or pull/merge request synced from a forge.
type GitHubItem struct {
	Number    int        `json:"number"`
	RepoID    string     `json:"repo_id"`
	Provider  string     `json:"provider,omitempty"` // "gitlab"; empty means GitHub
	Owner     string     `json:"owner"`
	RepoName  string     `json:"repo_name"`
	Type      string     `json:"type"` // "issue" or "pr"
	Title     string     `json:"title"`
	State     string     `json:"state"` // open, closed, merged
	Author    string     `json:"author"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	Labels    []string   `json:"labels,omitempty"`
	URL       string     `json:"url"`
	Assignees []string   `json:"assignees,omitempty"`
	Milestone string     `json:"milestone,omitempty"`
	Draft     bool       `json:"draft,omitempty"`
	Comments  int        `json:"comments,omitempty"`
	// ReviewDecision is GitHub's reviewDecision (APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED).
	ReviewDecision string `json:"review_decision,omitempty"`
	// MergeCommit is the SHA that landed the PR/MR on the target branch (merge or squash).
	MergeCommit string `json:"merge_commit,omitempty"`
}

// Tag is a git tag pointing at a commit, read from the local clone.
type Tag struct {
	RepoID string    `json:"repo_id"`
	Name   string    `json:"name"`
	Commit string    `json:"commit"`
	Date   time.Time `json:"date"`
}

// CIRun is one CI run (GitHub workflow run or GitLab pipeline) for a commit.
type CIRun struct {
	RepoID   string    `json:"repo_id"`
	Provider string    `json:"provider,omitempty"` // "gitlab"; empty means GitHub
	ID       string    `json:"id"`
	Commit   string    `json:"commit"`
	Name     string    `json:"name"`
	Status   string    `json:"status"` // success, failure, cancelled, in_progress, queued, ...
	Date     time.Time `json:"date"`
	URL      string    `json:"url"`
}

// Comment is an issue or PR/MR comment, stored only when comment sync is enabled.
type Comment struct {
	RepoID     string    `json:"repo_id"`
	Provider   string    `json:"provider,omitempty"` // "gitlab"; empty means GitHub
	ItemType   string    `json:"item_type"`          // "issue" or "pr"
	ItemNumber int       `json:"item_number"`
	ID         string    `json:"id"`
	Author     string    `json:"author"`
	Body       string    `json:"body"`
	Date       time.Time `json:"date"`
	URL        string    `json:"url,omitempty"`
}

// ForgeSchemaVersion is bumped when synced fields change; a mismatch forces a full resync.
const ForgeSchemaVersion = 2

// SyncState records how far a forge sync got for one repo and kind.
type SyncState struct {
	Watermark time.Time `json:"watermark"`
	Schema    int       `json:"schema"`
}

// TypeLabel returns the forge-specific label for the item type (GitLab calls PRs "MR").
func (i *GitHubItem) TypeLabel() string {
	if i.Type == "pr" && i.Provider == "gitlab" {
		return "MR"
	}
	return strings.ToUpper(i.Type)
}

type CrossReference struct {
	SourceRepoID string    `json:"source_repo_id"`
	SourceType   string    `json:"source_type"` // "commit", "issue", "pr"
	SourceID     string    `json:"source_id"`
	TargetRepoID string    `json:"target_repo_id"`
	TargetType   string    `json:"target_type"`
	TargetID     string    `json:"target_id"`
	Date         time.Time `json:"date"`
}
