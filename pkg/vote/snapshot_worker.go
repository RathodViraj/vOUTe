package vote

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// StartSnapshotWorker starts a background goroutine that snapshots active polls
// every 1 minute. It reads the vote counts from Redis and writes batch inserts
// into TimescaleDB. This function returns immediately and the worker respects
// the provided context for shutdown.
func StartSnapshotWorker(ctx context.Context, repo VoteRepository) {
	// convert to concrete implementation to access internal helpers
	r, ok := repo.(*voteRepo)
	if !ok {
		// unable to start worker without concrete repo
		return
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Snapshot pass
				if err := snapshotOnce(ctx, r); err != nil {
					// Log error — avoid crashing worker. Use fmt for minimal dependency.
					fmt.Printf("snapshot worker error: %v\n", err)
				}
			}
		}
	}()
}

func snapshotOnce(ctx context.Context, r *voteRepo) error {
	// Get active polls set from Redis
	ids, err := r.rdb.SMembers(ctx, "active_polls").Result()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	// For each poll, fetch vote counts and prepare batch insert
	type row struct {
		voteID   int64
		optionID int64
		count    int64
		ts       time.Time
	}

	var rows []row
	now := time.Now().UTC()
	for _, id := range ids {
		vid, err := parseVoteID(id)
		if err != nil {
			continue
		}

		counts, err := r.getVoteCount(ctx, vid)
		if err != nil {
			// skip this poll on error
			continue
		}

		for optID, cnt := range counts {
			rows = append(rows, row{voteID: vid, optionID: optID, count: cnt, ts: now})
		}
	}

	if len(rows) == 0 {
		return nil
	}

	// Batch insert into TimescaleDB
	// Build parameterized query
	// INSERT INTO vote_snapshots (vote_id, option_id, vote_count, created_at) VALUES ($1,$2,$3,$4),($5...)
	var b strings.Builder
	args := make([]interface{}, 0, len(rows)*4)
	b.WriteString("INSERT INTO vote_snapshots (vote_id, option_id, vote_count, created_at) VALUES ")
	for i, rrow := range rows {
		if i > 0 {
			b.WriteString(",")
		}
		idx := i * 4
		b.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d)", idx+1, idx+2, idx+3, idx+4))
		args = append(args, rrow.voteID, rrow.optionID, rrow.count, rrow.ts)
	}

	tx, err := r.timescaleDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, b.String(), args...); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
