package xref

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

var (
	// Matches owner/repo#123
	reFullRef = regexp.MustCompile(`([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+)#(\d+)`)
	// Matches #123 (same-repo reference)
	reLocalRef = regexp.MustCompile(`(?:^|\s)#(\d+)(?:\s|$|[,.)])`)
)

// DetectFromCommit analyzes a commit message for cross-references.
func DetectFromCommit(commit *model.Commit, repos []*model.Repository) []*model.CrossReference {
	var xrefs []*model.CrossReference
	msg := commit.Message

	// Build lookup maps
	repoByOwnerName := make(map[string]*model.Repository) // "owner/name" -> repo
	repoByID := make(map[string]*model.Repository)

	for _, r := range repos {
		repoByID[r.ID] = r
	}

	// Find full references: owner/repo#123
	for _, match := range reFullRef.FindAllStringSubmatch(msg, -1) {
		owner := match[1]
		repoName := match[2]
		number := match[3]
		key := owner + "/" + repoName

		targetRepo := repoByOwnerName[key]
		if targetRepo == nil {
			// Try to find a matching tracked repo by name
			for _, r := range repos {
				if strings.EqualFold(r.Name, repoName) {
					targetRepo = r
					break
				}
			}
		}

		targetRepoID := ""
		if targetRepo != nil {
			targetRepoID = targetRepo.ID
		} else {
			targetRepoID = key // Use owner/repo as fallback ID
		}

		if targetRepoID != commit.RepoID { // Skip self-references
			xrefs = append(xrefs, &model.CrossReference{
				SourceRepoID: commit.RepoID,
				SourceType:   "commit",
				SourceID:     commit.Hash,
				TargetRepoID: targetRepoID,
				TargetType:   "issue", // Could be issue or PR
				TargetID:     number,
				Date:         commit.Date,
			})
		}
	}

	// Find local references: #123 (reference to same repo)
	for _, match := range reLocalRef.FindAllStringSubmatch(msg, -1) {
		number := match[1]
		xrefs = append(xrefs, &model.CrossReference{
			SourceRepoID: commit.RepoID,
			SourceType:   "commit",
			SourceID:     commit.Hash,
			TargetRepoID: commit.RepoID,
			TargetType:   "issue",
			TargetID:     number,
			Date:         commit.Date,
		})
	}

	return xrefs
}

// AnalyzeRepo finds cross-references in all commits for a repo.
func AnalyzeRepo(s store.Store, repoID string, allRepos []*model.Repository) (int, error) {
	commits, err := s.GetCommitsByRepo(repoID, 0)
	if err != nil {
		return 0, fmt.Errorf("get commits: %w", err)
	}

	count := 0
	for _, commit := range commits {
		xrefs := DetectFromCommit(commit, allRepos)
		for _, xref := range xrefs {
			if err := s.PutCrossReference(xref); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}
