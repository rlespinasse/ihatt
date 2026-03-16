package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rlespinasse/ihatt/internal/model"
	bolt "go.etcd.io/bbolt"
)

var (
	bucketRepos       = []byte("repos")
	bucketCommits     = []byte("commits")
	bucketGitHubItems = []byte("github_items")
	bucketXRefs       = []byte("xrefs")
)

type BoltStore struct {
	db *bolt.DB
}

func NewBoltStore(path string) (*BoltStore, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bolt db: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketRepos, bucketCommits, bucketGitHubItems, bucketXRefs} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create buckets: %w", err)
	}

	return &BoltStore{db: db}, nil
}

func (s *BoltStore) Close() error {
	return s.db.Close()
}

// --- Repositories ---

func (s *BoltStore) PutRepo(repo *model.Repository) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(repo)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketRepos).Put([]byte(repo.ID), data)
	})
}

func (s *BoltStore) GetRepo(id string) (*model.Repository, error) {
	var repo model.Repository
	err := s.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(bucketRepos).Get([]byte(id))
		if data == nil {
			return fmt.Errorf("repo not found: %s", id)
		}
		return json.Unmarshal(data, &repo)
	})
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (s *BoltStore) ListRepos() ([]*model.Repository, error) {
	var repos []*model.Repository
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketRepos).ForEach(func(k, v []byte) error {
			var repo model.Repository
			if err := json.Unmarshal(v, &repo); err != nil {
				return err
			}
			repos = append(repos, &repo)
			return nil
		})
	})
	return repos, err
}

func (s *BoltStore) DeleteRepo(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketRepos).Delete([]byte(id))
	})
}

// --- Commits ---

// commitKey builds a key: <unix_timestamp_padded>:<repo_id>:<hash_prefix>
func commitKey(c *model.Commit) []byte {
	return []byte(fmt.Sprintf("%020d:%s:%s", c.Date.Unix(), c.RepoID, c.Hash[:8]))
}

func (s *BoltStore) PutCommit(commit *model.Commit) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(commit)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketCommits).Put(commitKey(commit), data)
	})
}

func (s *BoltStore) GetCommitsByTimeRange(from, to time.Time) ([]*model.Commit, error) {
	var commits []*model.Commit
	fromKey := []byte(fmt.Sprintf("%020d", from.Unix()))
	toKey := []byte(fmt.Sprintf("%020d", to.Unix()))

	err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketCommits).Cursor()
		for k, v := c.Seek(fromKey); k != nil; k, v = c.Next() {
			// Compare only the timestamp prefix
			if string(k[:20]) > string(toKey) {
				break
			}
			var commit model.Commit
			if err := json.Unmarshal(v, &commit); err != nil {
				return err
			}
			commits = append(commits, &commit)
		}
		return nil
	})
	return commits, err
}

func (s *BoltStore) GetCommitsByRepo(repoID string, limit int) ([]*model.Commit, error) {
	var commits []*model.Commit
	err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketCommits).Cursor()
		// Iterate in reverse (most recent first)
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var commit model.Commit
			if err := json.Unmarshal(v, &commit); err != nil {
				return err
			}
			if commit.RepoID == repoID {
				commits = append(commits, &commit)
				if limit > 0 && len(commits) >= limit {
					break
				}
			}
		}
		return nil
	})
	return commits, err
}

func (s *BoltStore) SearchCommits(query string, from, to *time.Time, author, repoID string) ([]*model.Commit, error) {
	var commits []*model.Commit
	queryLower := strings.ToLower(query)

	err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketCommits).Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var commit model.Commit
			if err := json.Unmarshal(v, &commit); err != nil {
				return err
			}
			if from != nil && commit.Date.Before(*from) {
				continue
			}
			if to != nil && commit.Date.After(*to) {
				continue
			}
			if repoID != "" && commit.RepoID != repoID {
				continue
			}
			if author != "" && !strings.Contains(strings.ToLower(commit.Author), strings.ToLower(author)) {
				continue
			}
			if query != "" && !strings.Contains(strings.ToLower(commit.Message), queryLower) {
				continue
			}
			commits = append(commits, &commit)
		}
		return nil
	})
	return commits, err
}

func (s *BoltStore) CountCommitsByRepo(repoID string) (int, error) {
	count := 0
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketCommits).ForEach(func(k, v []byte) error {
			// Check if the key contains the repo ID
			if strings.Contains(string(k), ":"+repoID+":") {
				count++
			}
			return nil
		})
	})
	return count, err
}

// --- GitHub Items ---

func githubItemKey(item *model.GitHubItem) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d", item.RepoID, item.Type, item.Number))
}

func (s *BoltStore) PutGitHubItem(item *model.GitHubItem) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(item)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketGitHubItems).Put(githubItemKey(item), data)
	})
}

func (s *BoltStore) GetGitHubItemsByTimeRange(from, to time.Time) ([]*model.GitHubItem, error) {
	var items []*model.GitHubItem
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketGitHubItems).ForEach(func(k, v []byte) error {
			var item model.GitHubItem
			if err := json.Unmarshal(v, &item); err != nil {
				return err
			}
			// Include if created or updated within range
			if (item.CreatedAt.After(from) || item.CreatedAt.Equal(from)) && (item.CreatedAt.Before(to) || item.CreatedAt.Equal(to)) {
				items = append(items, &item)
			} else if (item.UpdatedAt.After(from) || item.UpdatedAt.Equal(from)) && (item.UpdatedAt.Before(to) || item.UpdatedAt.Equal(to)) {
				items = append(items, &item)
			}
			return nil
		})
	})
	return items, err
}

func (s *BoltStore) GetGitHubItemsByRepo(repoID string) ([]*model.GitHubItem, error) {
	var items []*model.GitHubItem
	prefix := []byte(repoID + ":")
	err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketGitHubItems).Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var item model.GitHubItem
			if err := json.Unmarshal(v, &item); err != nil {
				return err
			}
			items = append(items, &item)
		}
		return nil
	})
	return items, err
}

// --- Cross-References ---

func xrefKey(xref *model.CrossReference) []byte {
	return []byte(fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		xref.SourceRepoID, xref.SourceType, xref.SourceID,
		xref.TargetRepoID, xref.TargetType, xref.TargetID))
}

func (s *BoltStore) PutCrossReference(xref *model.CrossReference) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(xref)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketXRefs).Put(xrefKey(xref), data)
	})
}

func (s *BoltStore) GetCrossReferencesByRepo(repoID string) ([]*model.CrossReference, error) {
	var xrefs []*model.CrossReference
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketXRefs).ForEach(func(k, v []byte) error {
			var xref model.CrossReference
			if err := json.Unmarshal(v, &xref); err != nil {
				return err
			}
			if xref.SourceRepoID == repoID || xref.TargetRepoID == repoID {
				xrefs = append(xrefs, &xref)
			}
			return nil
		})
	})
	return xrefs, err
}

func (s *BoltStore) GetCrossReferencesBySource(repoID, sourceType, sourceID string) ([]*model.CrossReference, error) {
	var xrefs []*model.CrossReference
	prefix := []byte(fmt.Sprintf("%s:%s:%s:", repoID, sourceType, sourceID))
	err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketXRefs).Cursor()
		for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
			var xref model.CrossReference
			if err := json.Unmarshal(v, &xref); err != nil {
				return err
			}
			xrefs = append(xrefs, &xref)
		}
		return nil
	})
	return xrefs, err
}
