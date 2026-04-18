package store

import (
	"time"

	"github.com/rlespinasse/ihatt/internal/model"
)

type Store interface {
	Close() error

	// Repositories
	PutRepo(repo *model.Repository) error
	GetRepo(id string) (*model.Repository, error)
	ListRepos() ([]*model.Repository, error)
	DeleteRepo(id string) error

	// Commits
	PutCommit(commit *model.Commit) error
	GetCommitsByTimeRange(from, to time.Time) ([]*model.Commit, error)
	GetCommitsByRepo(repoID string, limit int) ([]*model.Commit, error)
	SearchCommits(query string, from, to *time.Time, author, repoID string) ([]*model.Commit, error)
	CountCommitsByRepo(repoID string) (int, error)

	// GitHub items
	PutGitHubItem(item *model.GitHubItem) error
	GetGitHubItemsByTimeRange(from, to time.Time) ([]*model.GitHubItem, error)
	GetGitHubItemsByRepo(repoID string) ([]*model.GitHubItem, error)

	// Cross-references
	PutCrossReference(xref *model.CrossReference) error
	GetCrossReferencesByRepo(repoID string) ([]*model.CrossReference, error)
	GetCrossReferencesBySource(repoID, sourceType, sourceID string) ([]*model.CrossReference, error)
}
