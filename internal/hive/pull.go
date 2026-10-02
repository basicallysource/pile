package hive

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

type machine struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnerID string `json:"owner_id"`
}

// piece is one piece as GET /api/machines/{id}/pieces gives it.
type piece struct {
	UUID       string   `json:"piece_uuid"`
	LocalID    int64    `json:"local_id"`
	SeenAt     *string  `json:"seen_at"`
	Status     *string  `json:"classification_status"`
	Part       *string  `json:"part_id"`
	PartName   *string  `json:"part_name"`
	Color      *string  `json:"color_id"`
	ColorName  *string  `json:"color_name"`
	Confidence *float64 `json:"confidence"`
	Dead       bool     `json:"dead"`
}

func (p piece) seen() any {
	if p.SeenAt == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, *p.SeenAt)
	if err != nil {
		return nil
	}
	return float64(t.UnixNano()) / 1e9
}

const (
	// The most a page may hold.
	pageSize = 200
	// A pull that is not full goes back this far below the newest piece it
	// has, so pieces still being classified when it last ran are read again.
	overlap = 2000
	// Between pages, so a pull never crowds the sorters reporting in.
	pause = 150 * time.Millisecond
)

// Pull copies every machine the client may read into the store, or only the
// ones named in only: all of each machine's pieces when full, else the ones
// newer than the store has (and the last few it has). It reports each machine
// on progress.
func Pull(ctx context.Context, c *Client, s *Store, full bool, only []string, progress func(name string, pieces int)) error {
	var ms []machine
	if err := c.Get(ctx, "/api/machines?scope=all&include_archived=true", &ms); err != nil {
		return fmt.Errorf("list machines: %w", err)
	}
	for _, m := range ms {
		if len(only) > 0 && !slices.ContainsFunc(only, func(n string) bool { return strings.EqualFold(n, m.Name) || n == m.ID }) {
			continue
		}
		n, err := pullMachine(ctx, c, s, m, full)
		if errors.Is(err, ErrNotFound) {
			// Someone else's machine, and this user is not an admin.
			continue
		}
		if err != nil {
			return fmt.Errorf("machine %s: %w", m.Name, err)
		}
		if err := s.putMachine(m, m.OwnerID == c.UserID); err != nil {
			return err
		}
		progress(m.Name, n)
	}
	return nil
}

func pullMachine(ctx context.Context, c *Client, s *Store, m machine, full bool) (int, error) {
	newest, oldest, done, err := s.held(m.ID)
	if err != nil {
		return 0, err
	}
	if full || newest == 0 {
		return pages(ctx, c, s, m.ID, nil, -1)
	}
	// What is new since, from the top down to a little below the newest held.
	n, err := pages(ctx, c, s, m.ID, nil, newest-overlap)
	if err != nil || done {
		return n, err
	}
	// A pull that stopped part way through this machine: on down from the
	// oldest piece it got.
	more, err := pages(ctx, c, s, m.ID, &oldest, -1)
	return n + more, err
}

// pages reads a machine's pieces newest first from below cursor (nil for the
// top), until a page reaches stop or there are no more.
func pages(ctx context.Context, c *Client, s *Store, machine string, cursor *int64, stop int64) (int, error) {
	n := 0
	for {
		q := url.Values{"limit": {fmt.Sprint(pageSize)}}
		if cursor != nil {
			q.Set("cursor", fmt.Sprint(*cursor))
		}
		var page struct {
			Items      []piece `json:"items"`
			NextCursor *int64  `json:"next_cursor"`
		}
		if err := c.Get(ctx, "/api/machines/"+machine+"/pieces?"+q.Encode(), &page); err != nil {
			return n, err
		}
		if err := s.putPieces(machine, page.Items); err != nil {
			return n, err
		}
		n += len(page.Items)
		if page.NextCursor == nil || (len(page.Items) > 0 && page.Items[len(page.Items)-1].LocalID <= stop) {
			return n, nil
		}
		cursor = page.NextCursor
		select {
		case <-ctx.Done():
			return n, ctx.Err()
		case <-time.After(pause):
		}
	}
}
