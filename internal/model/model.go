package model

import "time"

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

type GitHubItem struct {
	Number    int        `json:"number"`
	RepoID    string     `json:"repo_id"`
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
