# Views

<!-- human -->
A page looks at one lot or several taken together, in an order you set: the
top of every page lists them, numbered, and "Add a lot" adds another. The
order is a priority: when a set is sorted out, its pieces are pulled from the
first lot that has them, then the next, so put the lot you would rather pull
from first. The sets, the parts and the totals are of all the lots together;
the sort-out queue and a set's panel say how many of its pieces are in each
lot. The queue stays as you add, take out or move lots, and each set in it
is counted again against them. The address carries the lots and their order, so a link opens the same
view. How long a sorter takes to collect a set is worked out for one lot at a
time; with several, pick which.

Which sets are listed is up to the filter, above the sets on the overview and
on Every set, the same on both. It opens on Interesting, which hides sets of
loose bricks and basic sets: those almost all plain bricks (like The Brick
Apple), those of ten or fewer different parts, and those with nothing unusual
in them. Models hides only the loose bricks, Everything hides nothing, and
each rule is a switch of its own. A theme, a size, sets, custom models or builds, and
licensed only narrow it further, and "Group by theme" puts each theme's sets
together. A set the filter hides says why in its panel. Each section of sets
folds from the button in its head, and stays folded after a reload.

What counts toward how complete a set is is yours to say too ("What counts",
beside the filter): a plain part always counts; printed parts and stickers,
minifigure body parts (heads, torsos, legs, hair) and minifigure accessories
(taps, cups, tools, often part of the build) each count only when switched
on. It opens counting the accessories and not the other two. A part that does
not count is still found: a set's panel shows it apart, as found or missing,
and sorting the set out takes it too. Whole minifigures are always apart.

A star on a set (on its tile, or at the top of its panel) keeps it to come
back to: stars are saved in pile itself, so they last and show on every
device, and "Starred only" lists nothing else.
<!-- /human -->

A view is how lots are looked at: which lots, in what order, exact or any
color, what counts, the filter, and the sort-out queue. Every
request about lots carries one (`View` in `proto/pile/v1/pile.proto`).

## The lots: a stack

- **In the route.** `/lots/big-box,goodwill-box` (and `/sets`, `/parts`,
  `/collect` under it) is a stack of two, the first first. One lot is a stack
  of one, so `/lots/big-box` is what it always was. `routeStack()` and
  `lotsOf()` in `app/src/lib/view.svelte.ts` read it; `stackHref(lots)` is the
  same page for other lots, the open set kept. `StackBar.svelte`, at the top
  of every page of lots, shows them and changes them (move earlier or later,
  take out, add), going to the new address without scrolling; the page keeps
  what it shows until the new numbers are in (`docs/loading-data.md`).
- **On the server.** `internal/server/stack.go`: a `stack` is the lots taken
  together: their parts in each color with each lot's count, the unmatched
  pieces, the totals (`Totals`: pieces, part-colors, first and last seen,
  the sorters, colors) and categories, the piles sets are matched against,
  and the likely order of the sets (`match.Explain` on the sum). `stacks`
  makes each lot's own stack at start (a lot's totals on the collection page
  are its stack's) and a stack of several when a view first asks for it,
  keeping the last 8; the likely order of a big stack takes a while, so the
  first request for it does.
- **Matching.** `match.Stack` holds each lot's pile and their sum. Sets are
  scored against the sum, so the order never changes completeness or which
  section a set is in. What a set takes (`Take`, for the sort-out queue) is
  drawn from the sum, one part and color at a time, from the first lot that
  has it, then the next; `Sources` says the same without taking it. The
  overview's queue (`QueuedSet.lots`), a set's lines (`SetLine.lots`) and the
  parts (`PartCount.lots`) carry those counts per lot, leaving out a lot
  with none. A set not in the queue is drawn against what the queue left.
- **Collect times** are fitted per lot (a sorter's mix), so
  `GetCollectTimes` takes one lot; the page times the stack's first lot until
  another is picked.
- **Find pieces** searches the bins, each holding one lot's pieces; its lot
  filter is its own.

## Counting

Every line of a set has a class (`catalog.Class`, from Rebrickable's part
categories in `internal/catalog/catalog.go`): *printed* (the Stickers
category, and any part that is the printed child of another in
part_relationships), *figure part* (Minifigs, Minifig Heads, Upper Body,
Lower Body, Headwear, Headwear Accessories, Hipwear, Neckwear),
*accessory* (Minifig Accessories, and Minifig Shields, Weapons, & Tools) or
*plain*. The View's `Counting` says which of the first three count toward a
set's completeness (`match.Counting`; unset, `match.DefaultCounting`:
accessories only). The index keeps every line (`Entry.Lines`), and
`Score` counts only the counted ones: a Match's `Need`, `Have`, rarity and
evidence follow the view, so the sections and the order do too. `find` and
`Take` work on every line, so a set's panel shows what was found of the
lines not counted (`SetLine.counted`) and the sort-out queue takes them
with their set. What is judged once per set (rarity, basic or loose bricks,
a minifigure set, the likely order) rests on the default counting; collect
times count plain parts only, as they always did.

The app keeps the counting with the view (`view.counting`), shows it in
"What counts" in `FilterBar.svelte`, and says it in the overview's header
(`counting.ts`). A set's panel (`SetDetail.svelte`) stacks its lines as
small tiles filled by how much was found (`LineTile.svelte`): missing, then
found, then not counted, then its whole minifigures.

## The filter

`Filter` in the View: which sets and custom models the overview and Every
set list. A set it hides is in no section and not on Every set, but its
panel opens and says why (`GetSetResponse.hidden_by`, `PLACE_HIDDEN`); a set
in the sort-out queue is never hidden.

- **Basic sets**, judged once per set in `internal/match/plain.go`, each a
  rule the filter can hide by: *mostly basic* (85% or more of its counted
  pieces in Rebrickable's Bricks, Plates, Tiles and Bricks Sloped
  categories), *few parts* (ten or fewer different parts) and *nothing
  unusual* (the average weight of its five rarest pieces, exact colors,
  below the tenth percentile of the sets LEGO sold). Calibrated on the 2026
  catalog: The Brick Apple is 90% plain; Space Skulls 37% and Police Boat 42%
  are not; the Monthly Mini Model Builds fall to few parts or nothing
  unusual. With *loose bricks* (`Entry.Bulk`, as before) these are the
  presets' rules: Interesting hides all four (the default), Models only loose
  bricks, Everything none (`presets` in `view.svelte.ts`).
- **Narrowing**: kind (sets, custom models, builds), one theme group (a
  designer's collection is one), licensed
  only, a least size in counted pieces, and starred only (below). The overview and Every set send
  the theme groups the rest of the filter lets through, with their counts,
  for the theme menu.
- **Grouping by theme** is the app's alone: `SetGrid` puts each theme's sets
  under its name, themes by name, each in the order given. So is folding a
  section (`folded.svelte.ts`, by section title, in the browser; the tiny
  sets start folded).
- `FilterBar.svelte` shows it all; `rules.ts` says the rules in words, for
  the switches and for a set's panel.

## Stars

The starred sets and custom models are the server's, in the data folder's
`stars.json` (`internal/stars`: most recent first, written whole or not at
all), read by `ListStars` and changed one at a time by `SetStar`. The app's
`stars.svelte.ts` holds them: a star changes at once and is saved after,
changing back if saving fails. A filter with starred only is filtered on
the server by its stars at the moment, and the request does not carry them,
so the overview, every set and a set's answers under it are keyed by the
stars' revision too (`version` on a Source, `docs/loading-data.md`): a new
star asks again.

## The rest of the view

Kept in the browser (`view.svelte.ts`), so it lasts across pages and
reloads: any-color matching, what counts, the filter, grouping by theme,
and the sort-out queue. The queue is the view's, not a stack's: adding,
taking out or moving a lot keeps its sets and their order, and the server
counts them again against the new lots (a set they cannot help stays, with
what they have of it). Before it was the view's, each stack kept one
(`sortOut` by stack); the last of those with sets in it became the view's.
The queue's sets take their pieces in order, each against what the ones
before it left; everything else is matched against what is left after all
of them (`internal/server/view.go`).

On a lot's overview the queue is a grid of cards (`SortOutQueue.svelte`),
put in order by dragging: a card picked up floats under the pointer as the
raised plane (no shadow, no lift), the others slide aside (`animate:flip`,
none under reduced motion), its empty place is a fill, and it glides into
that place when let go; Escape puts it back, and its grip moves it with the
arrow keys. The order shown is the view's from the moment a card is let go
(`view.reorder`); the cards' numbers, which depend on the order, fade until
the new count is in. A card's layout is read from its place in the grid
(`offsetLeft`, `offsetTop`), not from where a slide has it, so the gaps never
chase a moving card.
