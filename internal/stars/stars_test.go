package stars

import (
	"slices"
	"testing"
)

// Stars are kept in the folder, most recent first, and read back.
func TestKept(t *testing.T) {
	dir := t.TempDir()
	s, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"6291788-1", "10192-1", "7899-1"} {
		if err := s.Set(n, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Set("10192-1", false); err != nil {
		t.Fatal(err)
	}
	again, err := Read(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Sets(); !slices.Equal(got, []string{"7899-1", "6291788-1"}) {
		t.Fatalf("read back %v, want [7899-1 6291788-1]", got)
	}
}
