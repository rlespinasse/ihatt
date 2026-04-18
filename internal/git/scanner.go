package git

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

func RepoID(path string) string {
	h := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", h[:8])
}

func RepoName(path string) string {
	return filepath.Base(path)
}

// DiscoverRepos walks root directories and finds git repositories.
func DiscoverRepos(roots []string, maxDepth int) ([]string, error) {
	var repos []string
	seen := make(map[string]bool)

	for _, root := range roots {
		root, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return filepath.SkipDir
			}
			if !d.IsDir() {
				return nil
			}

			// Check depth
			rel, _ := filepath.Rel(root, path)
			depth := len(strings.Split(rel, string(filepath.Separator)))
			if depth > maxDepth {
				return filepath.SkipDir
			}

			// Skip hidden dirs (except .git check below)
			if d.Name() != "." && strings.HasPrefix(d.Name(), ".") && d.Name() != ".git" {
				return filepath.SkipDir
			}

			if d.Name() == ".git" {
				repoPath := filepath.Dir(path)
				if !seen[repoPath] {
					seen[repoPath] = true
					repos = append(repos, repoPath)
				}
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			continue
		}
	}
	return repos, nil
}

// IndexRepo opens a git repo and indexes all its commits into the store.
func IndexRepo(s store.Store, repo *model.Repository) (int, error) {
	r, err := gogit.PlainOpen(repo.Path)
	if err != nil {
		return 0, fmt.Errorf("open repo %s: %w", repo.Path, err)
	}

	ref, err := r.Head()
	if err != nil {
		return 0, fmt.Errorf("get HEAD for %s: %w", repo.Path, err)
	}

	iter, err := r.Log(&gogit.LogOptions{From: ref.Hash(), All: true})
	if err != nil {
		return 0, fmt.Errorf("log for %s: %w", repo.Path, err)
	}

	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		commit := &model.Commit{
			Hash:    c.Hash.String(),
			RepoID:  repo.ID,
			Author:  c.Author.Name,
			Email:   c.Author.Email,
			Message: strings.TrimSpace(c.Message),
			Date:    c.Author.When,
		}

		// Count file changes from diff with parent
		stats, err := c.Stats()
		if err == nil {
			for _, stat := range stats {
				if stat.Addition > 0 {
					commit.FilesModified++
				}
				if stat.Deletion > 0 {
					commit.FilesModified++
				}
			}
			// Simplified: count unique files
			commit.FilesModified = len(stats)
		}

		if err := s.PutCommit(commit); err != nil {
			return err
		}
		count++
		return nil
	})

	repo.LastScan = time.Now()
	if err2 := s.PutRepo(repo); err2 != nil {
		return count, err2
	}

	return count, err
}

// GetRemoteURL returns the origin remote URL for a repo path.
func GetRemoteURL(repoPath string) (string, error) {
	r, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return "", err
	}

	remote, err := r.Remote("origin")
	if err != nil {
		return "", err
	}

	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", fmt.Errorf("no URLs for origin remote")
	}
	return urls[0], nil
}
