package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/basicallysource/pile/internal/hive"
	"github.com/basicallysource/pile/internal/lots"
	"github.com/basicallysource/pile/internal/server"
)

// keyVar is the environment variable holding the Hive API key (scope
// records:read). It is never a flag, so it never shows in a process list.
const keyVar = "PILE_HIVE_KEY"

// refresher keeps the lots that read a Hive machine fresh: it reads those
// machines again through Hive's records API and, when anything changed,
// builds the lots and a new Service off to the side and swaps it in.
type refresher struct {
	data     string
	client   *hive.KeyClient
	machines []string
	// Machines whose change is read but not yet served: a refresh whose
	// lots fail to build tries them again on the next.
	pending map[string]bool

	shared *server.Shared
	live   *server.Live
	lots   []*lots.Lot
}

// newRefresher is a refresher for the machines lots read from the Hive at
// base, or nil when there are none or no key to read them with.
func newRefresher(data, base, key string, machines []string) (*refresher, error) {
	if len(machines) == 0 {
		return nil, nil
	}
	if key == "" {
		log.Printf("%s is not set: the lots' machines are served as last pulled, and not read again", keyVar)
		return nil, nil
	}
	c, err := hive.NewKeyClient(base, key)
	if err != nil {
		return nil, err
	}
	return &refresher{data: data, client: c, machines: machines, pending: map[string]bool{}}, nil
}

// first is the read at start, before the lots are loaded. When it fails, the
// records read before are served; with none, pile cannot start.
func (r *refresher) first(ctx context.Context) error {
	start := time.Now()
	pulled, err := r.pull(ctx)
	if err == nil {
		log.Printf("read from Hive in %s: %s", time.Since(start).Round(time.Millisecond), describe(pulled))
		return nil
	}
	if _, st := os.Stat(filepath.Join(r.data, hive.RecordsFile)); st != nil {
		return err
	}
	log.Printf("reading from Hive failed: %v; serving the records read before", err)
	return nil
}

// serve is what the refresher swaps fresh lots into: everything read so far
// is served.
func (r *refresher) serve(sh *server.Shared, live *server.Live, ls []*lots.Lot) {
	r.shared, r.live, r.lots = sh, live, ls
	clear(r.pending)
}

// run refreshes every d.
func (r *refresher) run(ctx context.Context, d time.Duration) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		start := time.Now()
		what, err := r.once(ctx)
		took := time.Since(start).Round(time.Millisecond)
		if err != nil {
			log.Printf("refresh failed after %s: %v; still serving the lots from before", took, err)
			continue
		}
		log.Printf("refresh in %s: %s", took, what)
	}
}

// once reads the machines and, when any changed, serves fresh lots. It says
// what it did.
func (r *refresher) once(ctx context.Context) (string, error) {
	pulled, err := r.pull(ctx)
	if err != nil {
		return "", err
	}
	if len(r.pending) == 0 {
		return "nothing changed", nil
	}
	changed := make([]string, 0, len(r.pending))
	for m := range r.pending {
		changed = append(changed, m)
	}
	slices.Sort(changed)
	ls, err := lots.Reread(r.data, r.lots, changed)
	if err != nil {
		return "", err
	}
	next, err := server.New(r.shared, ls)
	if err != nil {
		return "", err
	}
	r.live.Swap(next)
	r.lots = ls
	clear(r.pending)
	return describe(pulled) + "; lots served anew", nil
}

// pull reads every machine into records.sqlite and notes the ones that
// changed, those read before an error too.
func (r *refresher) pull(ctx context.Context) ([]hive.Pulled, error) {
	pulled, err := hive.PullMachines(ctx, r.client, filepath.Join(r.data, hive.RecordsFile), r.machines)
	for _, p := range pulled {
		if p.Changed {
			r.pending[p.Asked] = true
		}
	}
	if err != nil {
		return nil, fmt.Errorf("read from Hive: %w", err)
	}
	return pulled, nil
}

// describe says what a pull read: each machine, its pieces, and whether it
// changed.
func describe(pulled []hive.Pulled) string {
	if len(pulled) == 0 {
		return "no machines"
	}
	var out []string
	for _, p := range pulled {
		what := "unchanged"
		if p.Changed {
			what = "changed"
		}
		out = append(out, fmt.Sprintf("%s %s (%d pieces)", p.Name, what, p.Pieces))
	}
	return strings.Join(out, ", ")
}
