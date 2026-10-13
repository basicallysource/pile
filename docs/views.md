# Views

<!-- human -->
A page looks at one lot or several taken together, in an order you set: the
top of every page lists them, numbered, and "Add a lot" adds another. The
order is a priority: when a set is sorted out, its pieces are pulled from the
first lot that has them, then the next, so put the lot you would rather pull
from first. The sets, the parts and the totals are of all the lots together;
the sort-out queue and a set's panel say how many of its pieces are in each
lot. The address carries the lots and their order, so a link opens the same
view. How long a sorter takes to collect a set is worked out for one lot at a
time; with several, pick which.
<!-- /human -->

A view is how lots are looked at: which lots, in what order, exact or any
color, whether sets of loose bricks show, and the sort-out queue. Every
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

## The rest of the view

Kept in the browser (`view.svelte.ts`), so it lasts across pages and
reloads: any-color matching, whether sets of loose bricks show, and each
stack's sort-out queue, keyed by the stack as the route has it (a stack of
one by the lot's id). The queue's sets take their pieces in order, each
against what the ones before it left; everything else is matched against
what is left after all of them (`internal/server/view.go`).
