package server

import (
	"cmp"
	"slices"
	"strings"
	"unicode"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/storage"
)

// stock is what the storage's bins hold, ready to search: each part in a
// color, and how many of it each bin holds.
type stock struct {
	store    *storage.Storage
	binWords [][]string // each bin's address, as words
	pieces   []int32    // each bin's pieces
	parts    []*stored
	total    int32
}

// stored is a part in a color the bins hold.
type stored struct {
	part          *catalog.Part
	bricklinkPart string
	color         *catalog.Color
	image         string
	words         []string // its name, numbers, color and type, as words
	bins          []binCount
}

type binCount struct {
	bin int
	n   int32
}

// newStock puts every lot's pieces in the bins holding them: a piece goes to
// its lot's bin for the category its sorter sent it to, or else its lot's
// bin for the rest, or nowhere. Pieces the catalog cannot place are left out.
func newStock(st *storage.Storage, lots map[string]*lot, cat *catalog.Catalog) *stock {
	k := &stock{store: st, pieces: make([]int32, len(st.Bins))}
	for _, b := range st.Bins {
		k.binWords = append(k.binWords, words(b.Address))
	}
	holder := st.Holder()
	byPC := map[catalog.PartColor]*stored{}
	counts := map[catalog.PartColor]map[int]int32{}
	seen := map[string]bool{}
	for _, b := range st.Bins {
		if seen[b.Lot] {
			continue
		}
		seen[b.Lot] = true
		for _, p := range lots[b.Lot].Records.Pieces {
			bin, ok := holder(b.Lot, p.Category)
			if !ok {
				continue
			}
			pc, ok := cat.FromBrickLink(p.BrickLinkPart, p.BrickLinkColor, p.ColorKnown)
			if !ok {
				continue
			}
			s := byPC[pc]
			if s == nil {
				part, color := cat.Parts[pc.Part], cat.Colors[pc.Color]
				s = &stored{part: part, bricklinkPart: p.BrickLinkPart, color: color, image: cat.Images[pc],
					words: words(strings.Join([]string{part.Name, part.Num, p.BrickLinkPart, color.Name, part.Category}, " "))}
				byPC[pc] = s
				counts[pc] = map[int]int32{}
			}
			counts[pc][bin]++
			k.pieces[bin]++
			k.total++
		}
	}
	for pc, s := range byPC {
		for bin, n := range counts[pc] {
			s.bins = append(s.bins, binCount{bin, n})
		}
		slices.SortFunc(s.bins, func(a, b binCount) int { return cmp.Or(cmp.Compare(b.n, a.n), cmp.Compare(a.bin, b.bin)) })
		k.parts = append(k.parts, s)
	}
	return k
}

// stock is what the bins of the storage in use hold, worked out again when
// the storage was replaced since (through this Service or another sharing
// it).
func (s *Service) stock() *stock {
	st := s.storage.Now()
	if k := s.stocked.Load(); k != nil && k.store == st {
		return k
	}
	k := newStock(st, s.lots, s.cat)
	s.stocked.Store(k)
	return k
}

// words splits text into lowercase words of letters and digits.
func words(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// matches says whether every query word starts one of the words, and whether
// every one is one of them whole.
func matches(query []string, sets ...[]string) (ok, exact bool) {
	exact = true
	for _, q := range query {
		found, whole := false, false
		for _, ws := range sets {
			for _, w := range ws {
				if strings.HasPrefix(w, q) {
					found = true
					whole = whole || w == q
				}
			}
		}
		if !found {
			return false, false
		}
		exact = exact && whole
	}
	return true, exact
}

// tally counts pieces under each choice of one filter.
type tally struct {
	n     map[string]int32
	color map[string]*catalog.Color
}

func newTally() *tally { return &tally{n: map[string]int32{}, color: map[string]*catalog.Color{}} }

func (t *tally) add(value string, n int32) { t.n[value] += n }

func (t *tally) options(label func(string) string) []*pilev1.FilterOption {
	out := make([]*pilev1.FilterOption, 0, len(t.n))
	for v, n := range t.n {
		o := &pilev1.FilterOption{Value: v, Pieces: n, Color: color(t.color[v])}
		if label != nil {
			o.Label = label(v)
		}
		out = append(out, o)
	}
	slices.SortFunc(out, func(a, b *pilev1.FilterOption) int {
		return cmp.Or(cmp.Compare(b.Pieces, a.Pieces), cmp.Compare(a.Value, b.Value))
	})
	return out
}

// find is what the bins hold that req asks for, and each filter's choices
// given the search and the other filters.
func (k *stock) find(req *pilev1.FindPiecesRequest, lotName func(string) string) *pilev1.FindPiecesResponse {
	query := words(req.Query)
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 200
	}
	colors, types, bins, lots := newTally(), newTally(), newTally(), newTally()
	type found struct {
		s     *stored
		bins  []*pilev1.BinCount
		n     int32
		exact bool
	}
	var hits []found
	out := &pilev1.FindPiecesResponse{StoredPieces: k.total}
	for _, s := range k.parts {
		colorOK := req.Color == "" || s.color.Name == req.Color
		typeOK := req.Type == "" || s.part.Category == req.Type
		f := found{s: s}
		for _, bc := range s.bins {
			b := k.store.Bins[bc.bin]
			ok, exact := matches(query, s.words, k.binWords[bc.bin])
			if !ok {
				continue
			}
			binOK := req.Bin == "" || b.Name == req.Bin
			lotOK := req.LotId == "" || b.Lot == req.LotId
			if typeOK && binOK && lotOK {
				colors.add(s.color.Name, bc.n)
				colors.color[s.color.Name] = s.color
			}
			if colorOK && binOK && lotOK {
				types.add(s.part.Category, bc.n)
			}
			if colorOK && typeOK && lotOK && b.Name != "" {
				bins.add(b.Name, bc.n)
			}
			if colorOK && typeOK && binOK {
				lots.add(b.Lot, bc.n)
			}
			if colorOK && typeOK && binOK && lotOK {
				f.bins = append(f.bins, &pilev1.BinCount{Address: b.Address, Name: b.Name, LotId: b.Lot, Count: bc.n})
				f.n += bc.n
				f.exact = f.exact || exact
			}
		}
		if f.n > 0 {
			hits = append(hits, f)
			out.TotalPieces += f.n
		}
	}
	slices.SortFunc(hits, func(a, b found) int {
		if a.exact != b.exact {
			if a.exact {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(b.n, a.n), cmp.Compare(a.s.part.Name, b.s.part.Name), cmp.Compare(a.s.color.Name, b.s.color.Name))
	})
	out.TotalParts = int32(len(hits))
	for _, h := range hits[:min(limit, len(hits))] {
		out.Parts = append(out.Parts, &pilev1.StoredPart{
			PartNum:     h.s.part.Num,
			BricklinkId: h.s.bricklinkPart,
			Name:        h.s.part.Name,
			Color:       color(h.s.color),
			Type:        h.s.part.Category,
			ImageUrl:    h.s.image,
			Count:       h.n,
			Bins:        h.bins,
		})
	}
	out.Colors = colors.options(nil)
	out.Types = types.options(nil)
	out.Bins = bins.options(nil)
	out.Lots = lots.options(lotName)
	return out
}

// binMessages is the storage's bins, with what each holds.
func (k *stock) binMessages() []*pilev1.StorageBin {
	out := make([]*pilev1.StorageBin, len(k.store.Bins))
	for i, b := range k.store.Bins {
		out[i] = &pilev1.StorageBin{Address: b.Address, Name: b.Name, LotId: b.Lot, Categories: b.Categories, Pieces: k.pieces[i]}
	}
	return out
}
