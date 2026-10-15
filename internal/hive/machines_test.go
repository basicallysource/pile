package hive_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/basicallysource/pile/internal/hive"
	"github.com/basicallysource/pile/internal/hive/hivetest"
	"github.com/basicallysource/pile/internal/records"
)

// pieces is each piece's seen time, part and color as read back.
func pieces(t *testing.T, path, machine string) []string {
	t.Helper()
	r, err := hive.ReadMachine(path, machine, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range r.Pieces {
		out = append(out, strings.Join([]string{time.Unix(p.SeenAt, 0).UTC().Format("15:04:05"), p.BrickLinkPart, p.ColorName, p.ColorBy}, " "))
	}
	return out
}

// A pull reads every page, keeps what it read, and says whether it changed
// since the last; a piece gone from Hive goes from the store.
func TestPullMachines(t *testing.T) {
	m := &hivetest.Machine{ID: "m-1", Name: "MySorter"}
	for i := range int64(5) {
		m.Pieces = append(m.Pieces, hivetest.Classified(100+i, "3001", "11"))
	}
	h := hivetest.New(t, m, &hivetest.Machine{ID: "m-2", Name: "Other"})
	h.Page = 2
	c, err := hive.NewKeyClient(h.URL, hivetest.Key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), hive.RecordsFile)
	pull := func() []hive.Pulled {
		t.Helper()
		got, err := hive.PullMachines(context.Background(), c, path, []string{"mysorter"})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	got := pull()
	if want := []hive.Pulled{{Asked: "mysorter", Name: "MySorter", Pieces: 5, Changed: true}}; !reflect.DeepEqual(got, want) {
		t.Errorf("first pull %+v, want %+v", got, want)
	}
	if h.Pages() != 3 {
		t.Errorf("%d pages read, want 3 (2 a page)", h.Pages())
	}
	if n := len(pieces(t, path, "MYSORTER")); n != 5 {
		t.Errorf("%d pieces kept, want 5", n)
	}

	if got := pull(); got[0].Changed {
		t.Error("a pull of the same records said they changed")
	}

	// One piece gone, one corrected.
	m.Pieces = m.Pieces[1:]
	m.Pieces[0]["color_corrected_id"] = "5"
	m.Pieces[0]["correction_updated_at"] = hivetest.At(1000)
	m.Pieces[0]["color_predictions"] = []any{map[string]any{"source": "rerun", "model_name": "colormodel-v2", "color_id": "5", "color_name": "Red", "score": 0.8, "made_at": hivetest.At(900)}}
	h.Set(m)
	if got := pull(); !got[0].Changed || got[0].Pieces != 4 {
		t.Errorf("pull after a change %+v, want changed with 4 pieces", got)
	}
	want := []string{"00:01:41 3001 Red " + records.CorrectedColor, "00:01:42 3001 Color 11 ", "00:01:43 3001 Color 11 ", "00:01:44 3001 Color 11 "}
	if got := pieces(t, path, "m-1"); !reflect.DeepEqual(got, want) {
		t.Errorf("after the change %q, want %q", got, want)
	}
}

// Pieces lost, not classified, marked the wrong part or with no part are
// left out.
func TestReadMachineLeavesOut(t *testing.T) {
	keep := hivetest.Classified(100, "3001", "11")
	dead := hivetest.Classified(101, "3001", "11")
	dead["dead"] = true
	unclassified := hivetest.Classified(102, "3001", "11")
	unclassified["classification_status"] = "unknown"
	wrong := hivetest.Classified(103, "3001", "11")
	wrong["part_correct"] = false
	right := hivetest.Classified(104, "3002", "11")
	right["part_correct"] = true
	none := hivetest.Classified(105, "3001", "11")
	none["part_id"] = nil
	h := hivetest.New(t, &hivetest.Machine{ID: "m-1", Name: "mysorter", Pieces: []hivetest.Piece{keep, dead, unclassified, wrong, right, none}})
	c, _ := hive.NewKeyClient(h.URL, hivetest.Key)
	path := filepath.Join(t.TempDir(), hive.RecordsFile)
	if _, err := hive.PullMachines(context.Background(), c, path, []string{"m-1"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"00:01:40 3001 Color 11 ", "00:01:44 3002 Color 11 "}
	if got := pieces(t, path, "mysorter"); !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, want %q", got, want)
	}
}

// A key Hive refuses, and a proxy refusing the client, say so, and the key is
// in neither error.
func TestKeyErrors(t *testing.T) {
	h := hivetest.New(t)
	path := filepath.Join(t.TempDir(), hive.RecordsFile)
	c, _ := hive.NewKeyClient(h.URL, "wrong-key-123")
	_, err := hive.PullMachines(context.Background(), c, path, []string{"mysorter"})
	if err == nil || !strings.Contains(err.Error(), "refused the API key") || strings.Contains(err.Error(), "wrong-key-123") {
		t.Errorf("a wrong key: %v", err)
	}
	c, _ = hive.NewKeyClient(h.URL, hivetest.Key)
	if _, err := hive.PullMachines(context.Background(), c, path, []string{"mysorter"}); err == nil || !strings.Contains(err.Error(), `no machine "mysorter"`) {
		t.Errorf("a machine not the key owner's: %v", err)
	}
	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error code: 1010", http.StatusForbidden)
	}))
	defer blocked.Close()
	c, _ = hive.NewKeyClient(blocked.URL, hivetest.Key)
	if _, err := hive.PullMachines(context.Background(), c, path, []string{"mysorter"}); err == nil || !strings.Contains(err.Error(), "blocked this client") {
		t.Errorf("a client the proxy blocks: %v", err)
	}
}
