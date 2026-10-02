package server

import (
	"context"
	"errors"
	"log"
	"math"
	"slices"
	"sort"
	"sync"
	"time"

	"connectrpc.com/connect"

	pilev1 "github.com/basicallysource/pile/gen/pile/v1"
	"github.com/basicallysource/pile/internal/catalog"
	"github.com/basicallysource/pile/internal/collect"
	"github.com/basicallysource/pile/internal/hive"
	"github.com/basicallysource/pile/internal/records"
)

// resamples is how many resamplings of a lot's machine-days the page's
// ranges rest on: fewer than the research command's, to keep a busy
// machine free.
const resamples = 50

// collecting works out every lot's collect times in the background, one
// thing at a time: at start it fits each lot's model (smallest lot first) and
// times the standard sets, then times any other set a page asks for. What is
// done stays in memory until the server restarts with fresh data.
type collecting struct {
	cat  *collect.Catalog
	hive string
	// An image of each part, in some color, for the any-color pieces.
	partImage map[string]string

	mu    sync.Mutex
	lots  map[string]*lotTimes
	order []*lotTimes
	asked chan struct{}
}

type lotTimes struct {
	lot    *lot
	stream *collect.Stream
	basis  *pilev1.CollectBasis
	// Fitted (exact, near, any), and when; nil until then.
	models [3]*collect.Model
	fitted int64
	sets   map[string]*pilev1.SetCollectTimes
	// Asked for beyond the standard sets, in the order asked.
	extra []string
}

// newCollecting counts every lot's pieces (here, before serving, since it
// grows the index) and starts the background work.
func newCollecting(cat *catalog.Catalog, c *collect.Catalog, hivePath string, ls []*lot) *collecting {
	k := &collecting{cat: c, hive: hivePath, partImage: map[string]string{}, lots: map[string]*lotTimes{}, asked: make(chan struct{}, 1)}
	for pc, img := range cat.Images {
		if _, ok := k.partImage[pc.Part]; !ok {
			k.partImage[pc.Part] = img
		}
	}
	for _, l := range ls {
		t := &lotTimes{lot: l, stream: c.NewStream(l.Records), sets: map[string]*pilev1.SetCollectTimes{}}
		k.lots[l.ID] = t
		k.order = append(k.order, t)
	}
	sort.SliceStable(k.order, func(i, j int) bool { return k.order[i].stream.Pieces < k.order[j].stream.Pieces })
	go k.run()
	return k
}

func (k *collecting) run() {
	for _, t := range k.order {
		start := time.Now()
		basis := k.basis(t)
		var models [3]*collect.Model
		for i, m := range []collect.Mode{collect.Exact, collect.Near, collect.Any} {
			models[i] = k.cat.Fit(t.stream, m)
		}
		k.mu.Lock()
		t.basis, t.models, t.fitted = basis, models, time.Now().Unix()
		k.mu.Unlock()
		log.Printf("collect times: %s fitted in %s", t.lot.ID, time.Since(start).Round(time.Second))
	}
	for _, t := range k.order {
		for _, num := range collect.Standard() {
			k.asks()
			k.time(t, num)
		}
		log.Printf("collect times: %s, the standard sets done", t.lot.ID)
	}
	for range k.asked {
		k.asks()
	}
}

// asks times every set a page asked for that is not timed yet.
func (k *collecting) asks() {
	for {
		t, num := k.next()
		if t == nil {
			return
		}
		k.time(t, num)
	}
}

// next is a set a page asked for and not yet timed.
func (k *collecting) next() (*lotTimes, string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, t := range k.order {
		for _, num := range t.extra {
			if t.sets[num] == nil {
				return t, num
			}
		}
	}
	return nil, ""
}

func (k *collecting) time(t *lotTimes, num string) {
	k.mu.Lock()
	done := t.sets[num] != nil
	k.mu.Unlock()
	if done {
		return
	}
	out := &pilev1.SetCollectTimes{SetNum: num}
	if set := k.cat.Ix.Entry(num); set == nil {
		out.Unknown = true
	} else {
		s := set.Set
		out.Name, out.Year, out.Theme, out.ImageUrl = s.Name, s.Year, s.Theme, s.ImageURL
		for i, md := range t.models {
			r, err := k.cat.Time(md, num, resamples, 1)
			if err != nil {
				out.Unknown = true
				break
			}
			ct := k.times(r, t.basis.RatePerHour)
			switch i {
			case 0:
				out.Exact = ct
				out.Pieces, out.LeftOut, out.Minifigures = int32(r.Need.Pieces), r.Need.Left, r.Need.Figures
			case 1:
				out.Near = ct
			case 2:
				out.Any = ct
			}
		}
	}
	k.mu.Lock()
	t.sets[num] = out
	k.mu.Unlock()
}

func (k *collecting) times(r *collect.Result, rate float64) *pilev1.CollectTime {
	h := func(pieces float64) float64 {
		if math.IsInf(pieces, 1) || math.IsNaN(pieces) {
			return -1
		}
		return pieces / rate
	}
	ct := &pilev1.CollectTime{
		Half: h(r.At.Half), Most: h(r.At.Most), Nearly: h(r.At.Nearly),
		All: h(r.At.All), AllFast: h(r.At.Fast), AllSlow: h(r.At.Slow),
		MostLow: h(r.MostLow), MostHigh: h(r.MostHigh), AllLow: h(r.AllLow), AllHigh: h(r.AllHigh),
		AllSeen: h(r.SeenAll), AllRarer: h(r.RarerAll),
		Keys: int32(r.Keys), UnseenKeys: int32(r.Unseen), UnseenPieces: int32(r.UnseenPieces),
	}
	for _, s := range r.Slowest {
		num := k.cat.Ix.PartNum(s.Key.Part)
		p := &pilev1.SlowPiece{PartNum: num, Need: int32(s.Need), Seen: int32(s.Seen), WaitHours: h(s.Wait), ImageUrl: k.partImage[num]}
		if part := k.cat.Cat.Parts[num]; part != nil {
			p.Name = part.Name
		}
		if r.Mode != collect.Any {
			// The near mode's key is its group's first color.
			p.Color = color(k.cat.Cat.Colors[s.Key.Color])
			if img := k.cat.Cat.Images[catalog.PartColor{Part: num, Color: s.Key.Color}]; img != "" {
				p.ImageUrl = img
			}
		}
		ct.Slowest = append(ct.Slowest, p)
	}
	return ct
}

// basis is what a lot's times rest on. Its rate is classified pieces an
// hour of sorting: from Hive's records of every piece when the lot is from
// Hive, else from its own classified pieces.
func (k *collecting) basis(t *lotTimes) *pilev1.CollectBasis {
	r := t.lot.Records
	b := &pilev1.CollectBasis{
		Pieces: int32(t.stream.Pieces), MachineName: machineName(r.Machines),
		FirstSeenUnix: r.FirstSeen, LastSeenUnix: r.LastSeen, SnapshotUnix: r.CopiedAt,
	}
	classified := 0
	if t.lot.Hive != "" {
		st, err := hive.SortingTime(k.hive, t.lot.Hive)
		if err != nil {
			log.Printf("collect times: %s: %v", t.lot.ID, err)
		}
		for _, m := range st {
			b.SortingHours += m.Seconds / 3600
			classified += m.Classified
		}
	} else {
		b.SortingHours = sortingHours(r)
		classified = len(r.Pieces)
	}
	if b.SortingHours > 0 {
		b.RatePerHour = float64(classified) / b.SortingHours
	}
	return b
}

// sortingHours is how long a lot's sorters spent on its classified pieces:
// the gaps of a minute or less between one and the next on each machine.
func sortingHours(r *records.Records) float64 {
	last := map[int32]int64{}
	var s int64
	for _, p := range r.Pieces {
		if t, ok := last[p.Machine]; ok && p.SeenAt > t && p.SeenAt-t <= 60 {
			s += p.SeenAt - t
		}
		last[p.Machine] = p.SeenAt
	}
	return float64(s) / 3600
}

func (s *Service) GetCollectTimes(_ context.Context, req *connect.Request[pilev1.GetCollectTimesRequest]) (*connect.Response[pilev1.GetCollectTimesResponse], error) {
	k := s.collecting
	t := k.lots[req.Msg.LotId]
	if t == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no lot "+req.Msg.LotId))
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	out := &pilev1.GetCollectTimesResponse{Basis: t.basis, FittedUnix: t.fitted}
	asked := false
	nums := collect.Standard()
	for _, num := range req.Msg.Extra {
		if !slices.Contains(nums, num) {
			nums = append(nums, num)
			if !slices.Contains(t.extra, num) {
				t.extra = append(t.extra, num)
				asked = true
			}
		}
	}
	for _, num := range nums {
		if st := t.sets[num]; st != nil {
			out.Sets = append(out.Sets, st)
			continue
		}
		p := &pilev1.SetCollectTimes{SetNum: num, Pending: true}
		if e := k.cat.Ix.Entry(num); e != nil {
			p.Name, p.Year, p.Theme, p.ImageUrl = e.Set.Name, e.Set.Year, e.Set.Theme, e.Set.ImageURL
		}
		out.Sets = append(out.Sets, p)
	}
	if asked {
		select {
		case k.asked <- struct{}{}:
		default:
		}
	}
	return connect.NewResponse(out), nil
}
