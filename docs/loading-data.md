# Loading data

<!-- human -->
Nothing you do on a page makes it jump or go blank. When a click changes the
numbers (a set put in the sort-out queue, another color mode, a filter), the
page keeps showing what it had until the new numbers are in, then changes
them in place: whatever you were looking at stays where it was on the
screen, and your open sections and filters stay as they were. If the new
numbers take more than a moment, what is updating dims and the top bar says
what it is working on. Coming back to a page, with Back or a reload, lands
where you were scrolled, even when the server is slow to answer.
<!-- /human -->

Every page loads through one layer, `app/src/lib/data/`. No page calls the
client (`pile` in `app/src/lib/client.ts`) itself; only the stars, which
the reader changes on the server, do (`docs/views.md#stars`).

## Sources and queries

- **A Source** (`query.svelte.ts`) is one of the server's calls with the
  answers this visit already has, by request. `app/src/lib/api.ts` makes one
  per call in `sources`. A call asked again while its answer is on its way
  shares that one question. Options, set by what the data is:
  - `doing`: what the top bar says while an answer is slow ("Matching the
    lot's sets").
  - `settled`: whether an answer is final. One that is not (collect times,
    worked out on the server a set at a time) is asked for again every 4
    seconds while it is shown.
  - `changes`: answers that can change while pile runs (what the bins hold,
    replaced by `SetStorage`): one had is shown at once and asked for again.
  - `version`: what else on the server an answer depends on that the
    request does not carry (the stars, when only starred sets are asked
    for). It is part of the answer's key and read where the query notices
    it change, so a change asks again.
  Everything else in pile changes only when it restarts, so an answer lasts
  the visit and a reload picks up a restart.
- **A Query** is a component asking a source:
  `query(sources.set, () => request, (req) => req.setNum)`. The request
  function is reactive: whenever what it reads changes (the view, a filter,
  the route), the query asks again. Returning `undefined` asks nothing (a
  sheet with nothing open). The third argument says what the answer is
  about (here a set: another set does not show this one's answer); left
  out, every request is about the same thing, as for an overview whose
  lots change. The query keeps `data` (what is shown), `error`, `asking` (a newer
  answer is on its way) and `slow` (it has taken more than 300 ms).
- **The rules.** An answer already had is drawn on the first render, so a
  page come back to is already its full height when the browser restores
  its scroll. A new request about the same thing keeps the last answer shown
  until its own is in; it never blanks the page. A request about something
  else (another set) shows nothing of the last one: it waits,
  in the shape of what is coming. A newer answer replaces what is shown
  through `keepPlace`, below.
- **Drawing it.** `Loaded.svelte` draws a query: `waiting` (a skeleton the
  shape of what is coming, so nothing moves when it lands) until the first
  answer, an Alert if that fails, then `children` with the answer. While a
  slow update is on its way the region dims and is `aria-busy`; an update
  that fails leaves the answer shown under an Alert. `Updating.svelte`, at
  the start of the top bar's right side, shows the Spinner and the slow
  source's `doing` (the sentence from 1280px up), so nothing on the page
  moves for it.
- **Actions change state first.** What an action changes is drawn from the
  state at once, not from the answer: a tile's and the sheet's sort-out
  button read the queue in `view.svelte.ts`. Lists are keyed by their
  item's id, so an answer updates the elements in place instead of
  remounting them.

## Keeping the place

`place.ts`. When what is on screen changes under the reader, the page must
not move. `keepPlace(change)` notes the place in every scrolling area (the
page, the sheet's body, the Find pieces list), makes the change, and once
it is drawn scrolls each area so its place is back where it was on the
screen. Changes in one turn share one hold.

- **A place** is an element marked `data-place`, and each element directly
  inside one marked `data-place="blocks"`: a `Loaded` region marks itself so,
  making the region and each of its blocks (a panel, a grid) a place. A place
  stays put while its insides change. An item of a list that reorders is
  never a place: the browser keeps its element across a reorder, and
  following it would carry the page away to wherever the item went. So a
  `Loaded` region draws blocks, never a list's items directly.
- **Room at the end.** When the change leaves an area too short to scroll
  back to the place (a filter emptied what was below it), the browser would
  pull the page down; instead the area gets an empty spacer at its end, as
  tall as it was short, which shrinks as the reader scrolls back up and goes
  when the page changes (`clearRoom`, from the root layout).
- **Which one is held**: in the area under the pointer, the smallest place
  under it; elsewhere, the place at the top of the area. The places nearest
  it are held as well, to fall back on when it is gone after the change (a
  section that emptied).
- `holdPlace(at)` is the same hold, for changes that are not an answer:
  opening a set's sheet narrows the page, so the tile clicked is held
  (`open-set.svelte.ts`); closing it widens the page, so the open tile is
  held, and when closing goes Back the hold is handed to the snapshot
  restore below instead of the browser's own scroll.
- **Coming back.** The root layout's `snapshot` keeps the scroll position of
  every history entry. On Back or a reload, `resumeAt` scrolls there again
  each time the page grows, until it gets there, the reader scrolls, or 20
  seconds pass: after a reload nothing is cached, and the page is short
  until its answers are in.

The browser's own scroll anchoring stays on; the hold measures after it and
has the last word.
