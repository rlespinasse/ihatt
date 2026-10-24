// Package forge holds the sync bookkeeping shared by the GitHub and GitLab clients.
package forge

import (
	"fmt"
	"os"
	"time"

	"github.com/rlespinasse/ihatt/internal/model"
	"github.com/rlespinasse/ihatt/internal/store"
)

// overlap re-fetches a little before the last watermark so items updated while
// a sync was running are not missed.
const overlap = 5 * time.Minute

// Plan is one sync of one kind (issues or pull/merge requests) for one repo.
type Plan struct {
	key     string
	started time.Time
	// Since is the updated-after bound; the zero value means a full sync.
	Since time.Time
	// commentsFailed keeps the watermark from advancing, so a failed comment
	// fetch is retried by the next sync.
	commentsFailed bool
}

// Start decides how a sync runs. It is incremental when a watermark exists for
// the same schema version and full is false; otherwise it fetches everything.
func Start(s store.Store, repoID, provider, kind string, full bool) (Plan, error) {
	p := Plan{
		key:     repoID + ":" + provider + ":" + kind,
		started: time.Now().UTC(),
	}
	if full {
		return p, nil
	}
	state, err := s.GetSyncState(p.key)
	if err != nil {
		return p, err
	}
	if state != nil && state.Schema == model.ForgeSchemaVersion && !state.Watermark.IsZero() {
		p.Since = state.Watermark.Add(-overlap)
	}
	return p, nil
}

// UpdatedSince reports whether an item updated at t falls inside this plan's
// window, so its comments must be re-fetched. A full sync always does.
func (p Plan) UpdatedSince(t time.Time) bool {
	return !p.Incremental() || !t.Before(p.Since)
}

// Incremental reports whether the plan only fetches items updated since Since.
func (p Plan) Incremental() bool {
	return !p.Since.IsZero()
}

// CommentFailed records that one item's comments could not be fetched. The
// item sync goes on; the watermark stays put so the next sync retries them.
func (p *Plan) CommentFailed(itemType string, number int, err error) {
	p.commentsFailed = true
	fmt.Fprintf(os.Stderr, "warning: comments for %s #%d not synced: %v\n", itemType, number, err)
}

// Done advances the watermark. Call it only after the kind fully succeeded.
func (p Plan) Done(s store.Store) error {
	if p.commentsFailed {
		return nil
	}
	return s.PutSyncState(p.key, &model.SyncState{
		Watermark: p.started,
		Schema:    model.ForgeSchemaVersion,
	})
}
