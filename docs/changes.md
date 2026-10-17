# Changes to sets

<!-- human -->
A set is rarely built exactly as its inventory says: a piece you lack can
be another color, or be made of other pieces. These choices are changes to
the set, and pile keeps them with it, so they last, show on every device,
count wherever the set is matched (its panel, the sort-out queue, the
overview), and go into whatever is made from the set later, such as
building instructions.

Every part in a set's panel is a button: click it, or right-click it, for
its menu.

- **Color.** Its own color; one other color the lots have it in (each
  says how many pieces); several colors, checked in a list, so its pieces
  come from any mix of them (one yellow, one orange, one light yellow); or
  any color. A small mark by its count says which is in force: a dot of
  the color, a dot split into the colors, a ring for any.
- **Make it from other parts.** What could stand in for one of the part,
  each with how many the lots could make: for a plain plate, tile or brick,
  smaller ones of its kind that cover the same footprint (a 2 x 4 plate
  from two 2 x 2s, or four 1 x 2s), however they would hold together, and
  "any that fit", which picks from whatever is there; for any part, the
  part it is a print of, its prints, and the parts it is assembled from;
  and a search to pick any parts and say how many of each make one (two
  1 x 1 clip plates for a 1 x 2 clip plate). The part's own pieces are
  used first, then the stand-ins checked, in turn. A part found this way
  shows a small blocks mark, and says on hover which pieces made it.
- **Count it toward this set**, or leave it out, whatever What counts
  says for its kind (a printed part you will find, a minifigure part that
  is really part of the build). It moves between Missing, Found and Not
  counted, and the set's completeness follows.
- **See the picture up close.**

Each color of the set can take another color too, from Colors in its
panel. Every change is listed under Changes in the set's panel, in a few
words ("60470b Yellow: or 2 x 4085"), each with its own Undo.
<!-- /human -->

A change is how a set will be built where it differs from its inventory:
for one line (a part in a color, as the set has them) or, for colors only,
every line in a color. `Change` in `proto/pile/v1/pile.proto` says it on
the wire; `internal/changes` keeps it.

## Kept

`changes.json` in the data folder: an object of set numbers, each a list of
changes in the order they were made, read at start (`changes.Read`) and
written whole (to `changes.json.new`, then renamed over it) before a change
is answered, like `stars.json`. Each change has `part` (empty for a whole
color), `color` (as the set has it) and exactly one of:

- `colors`: `{"use": [ids]}`, the colors it takes instead of its own (one,
  or several to mix: the first counts as its color, the rest as another
  color), or `{"any": true}`; `{}` is its own color, which undoes its
  color's change for one part.
- `swap`: a list of stand-ins, each `{"parts": [{"part", "quantity"}]}`
  (so many of each make one of the line's part) or `{"fits": true}`.
- `counted`: `true` or `false`, whatever the view counts.

A set has at most one change of each kind per line; making one replaces it
in place (`Kept.Make`), and `Kept.Undo` takes it back. A set with none is
gone from the file.

Through the API: `ListChanges` (every set's), `GetSet` (`changes`, the
set's) and `SetChange` (make one, or `undo`). Every `Change` the server
sends carries `text`, the change in part numbers and color names, which
instructions made later can show as it is. The server checks a change
against the catalog: the set has the line, the colors and parts exist, and
`fits` is only for a part with a footprint. A swap with no stand-ins takes
the line's swap back. The app has no list of a set's lines to export, so
the changes reach anything made from a set through the API.

## Matching

`match.WithChanges` (`internal/match/changes.go`) makes a set's entry as its
changes make it, line for line in the same order as the entry as sold (so
`LineOf` finds a set's line either way, and two lines made one part in one
color stay two lines, sharing its pieces as the matcher draws them). Each
line's changes sit beside the lines (`how`): the colors it takes beside its
own and whether any, whether it counts, and its stand-ins as recipes. The
view (`internal/server/view.go`) makes every changed set's entry at each
request, from the kept changes at that moment, and scores it in place of
the set's own everywhere: the queue, the overview, every set, a set's panel.

`Pile.allot` fills an entry's lines in three passes, each over every line
before the next:

1. Its own part in its color (the first color it takes).
2. Its own part in the other colors it takes (`Colors.others`): any color
   (the view's or its change's) the most plentiful first; else the colors
   its change mixes in, the most plentiful first, then in similar colors
   the near ones. Counted as `wrong_color`.
3. Its stand-ins, in turn (`Pile.make`): each recipe as many times as what
   is left allows, every piece of it drawn from the colors the line takes,
   in that order. A recipe is made whole or not at all. Counted as found
   (`Found.Made`, and the pieces taken in `Found.Used`), not as another
   color.

What a stand-in takes is drawn like any other piece, so the sort-out queue
takes it out of the pile, from the first lot that has it, and the sets
after it see it gone. A counted line's change is read by `Entry.Counts`,
which `Score` asks instead of the view's `Counting`.

## Footprints

`internal/catalog/shapes.go` reads a part's shape from its name when it is
a plain rectangle of whole studs: Rebrickable's `Plate W x L`,
`Tile W x L` (with or without groove) and `Brick W x L` or `Brick W x L x H`
(81 parts in the catalog of October 2026). `internal/match/footprints.go` keeps, for
each shape, the part in the most sets with it, and works out a part's ways
the first time it is asked for: the footprint cut straight across, and each
side again and again, down to sizes that are parts of the same kind and
height (plates with plates, 1-high bricks with 1-high bricks), every way as
the pieces it takes, each once, the part itself left out. A way has at most
8 pieces, and each size keeps the 32 with the fewest. Cutting straight
across finds every way of a few pieces; a way only a pinwheel can make (five
or more pieces around a middle) is left out, which no one misses.
Structure is ignored, as asked: a 2 x 4 plate from two 1 x 4s is a way.
"Any that fit" is every way, the fewest pieces first, each made as often as
the pieces allow.

## Stand-ins offered

`GetStandIns` (`internal/server/standins.go`) lists, for a set's line:

- `fits`, when the part has a footprint.
- Every way to cover its footprint.
- From `part_relationships.csv` (`catalog.Related`, of the part and of the
  part standing for its variants): the part it is a print or pattern of
  (rel types P and T), its prints and patterns, and the parts it is
  assembled from (B, one of each: the file gives no quantities). Molds and
  alternates (M and A) are one part already, so they need no stand-in.
- The line's chosen stand-ins, first, in its order (a hand-picked one as
  `PICKED`).

Each with `makes` (how many of the line's part the lots could make, up to
its need) and each part's pieces, both of what is left once the queue
before the set and the set itself (without this line's stand-ins) have
taken theirs, in the colors the line takes. A way that makes none is left
out, and at most 12 are listed besides those chosen, the most made first,
then the fewest pieces. `FindParts` searches the catalog's parts by name or
number for the hand-picked kind, counting pieces in the lots in the colors
asked for.

## In the app

`app/src/lib/pile/changes.svelte.ts` holds every set's changes: a change
shows there at once (the menus' checks read it), is sent with `SetChange`,
and once kept bumps `revision`, which every Source matched through the sets
carries as its version (`api.ts`), so their answers are asked for again.
`LineTile.svelte` is the part's button and menu, `SeveralColors.svelte` and
`StandIns.svelte` the two dialogs it opens (one each, in `SetDetail.svelte`),
and `SetDetail.svelte` lists the changes with their undo. Changes the view
kept in the browser before (`choices`) are sent to the server once and
forgotten (`keepWithSets` in `view.svelte.ts`).
