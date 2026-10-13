package match

import (
	"slices"
	"testing"
)

func pileOf(counts map[[2]int32]int32) *Pile {
	p := &Pile{exact: map[uint64]int32{}, byPart: map[int32]int32{}, colors: map[int32][]int32{}}
	for pc, n := range counts {
		p.exact[key(pc[0], pc[1])] = n
		p.byPart[pc[0]] += n
		p.colors[pc[0]] = append(p.colors[pc[0]], pc[1])
	}
	return p
}

// A set's pieces come from the first lot that has them, then the next, in
// its own color first and then (any color) in others.
func TestStackDrawsFirstLotFirst(t *testing.T) {
	first := pileOf(map[[2]int32]int32{{1, 1}: 2})
	second := pileOf(map[[2]int32]int32{{1, 1}: 3, {1, 2}: 1})
	s := NewStack([]*Pile{first, second})
	e := &Entry{Lines: []Line{{Part: 1, Color: 1, Quantity: 4}, {Part: 1, Color: 3, Quantity: 2}}}

	if got := s.Sources(e, true); !slices.Equal(got[0], []int32{2, 2}) || !slices.Equal(got[1], []int32{0, 2}) {
		t.Fatalf("sources %v, want [[2 2] [0 2]]", got)
	}
	if s.Sum.exact[key(1, 1)] != 5 {
		t.Fatal("Sources took pieces")
	}
	got := s.Take(e, true)
	if !slices.Equal(got.Lots(2), []int32{2, 4}) {
		t.Fatalf("took %v from the lots, want [2 4]", got.Lots(2))
	}
	for _, p := range []*Pile{first, second, s.Sum} {
		if p.Pieces() != 0 {
			t.Fatalf("%d pieces left in a pile, want 0", p.Pieces())
		}
	}
}

// One lot is its own sum: taking from it takes once.
func TestStackOfOneTakesOnce(t *testing.T) {
	only := pileOf(map[[2]int32]int32{{1, 1}: 3})
	s := NewStack([]*Pile{only})
	s.Take(&Entry{Lines: []Line{{Part: 1, Color: 1, Quantity: 2}}}, false)
	if n := only.exact[key(1, 1)]; n != 1 {
		t.Fatalf("%d left, want 1", n)
	}
}
