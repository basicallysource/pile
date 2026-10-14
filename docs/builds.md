# Builds

<!-- human -->
Many sets build several things: a gas station and a car, a building and a
fountain, a ship and its stand. A build is one of those things on its own,
with its own parts list, matched against your pieces like a set: you can see
that one building of a set is nearly all here while the whole set is not,
and sort out just that building. Builds come from 3D models of the sets that the LDraw
community made, so only sets someone modeled have them (under a tenth of
sets, most of them older ones). Each has a picture drawn from its model.
<!-- /human -->

## Where they come from

The [LDraw Official Model Repository](https://library.ldraw.org/omr) holds
fan-made LDraw models of official sets (about 1,500 in 2026), shared under
CCAL 2.0 (attribution: each model's author is kept and shown).
`scripts/builds/gather.py DATA/set-models` gathers each set's main model
into `set-models/` and lists them in `set-models/index.json` (the set, the
file, its author, its page, and its maker's notes on what it lacks). It goes
easy on the site (two requests a second, about 50 minutes for all of it) and
fetches only what it lacks when run again.

Reading the models leans on the LDraw parts library (`complete.zip` from
ldraw.org, unpacked) in the data folder's `ldraw/`: what a part is made of,
its shape, and its colors (`LDConfig.ldr`). Without it there are fewer
builds and no pictures. It is read many times while loading: keep it on a
fast disk.

## How a model becomes builds

`internal/catalog/builds.go` and `objects.go`, at load:

1. **Parts.** Every part the model places (`internal/ldraw`), through its
   submodels, with its color (16 inherits) and where it ends up. Each is
   read as a catalog part: the LDraw ids Rebrickable lists (in `parts.db`),
   else the same number, else, for a part LDraw makes of other parts (a
   shortcut, such as a wheel with its tire), each of those parts. Primitives
   and subparts drawn on their own are geometry, not parts.
2. **Objects.** Parts hold together when a stud of one is inside the body of
   the other (the body is its geometry without its studs), or when their
   bodies go more than 1.5 LDraw units into each other (a pin in a hole, a
   bar in a clip). Faces that only touch, and a part resting on a smooth
   surface, do not hold: two buildings on neighboring baseplates are two
   objects, and a car parked on a road is apart from it. Each group of parts
   that hold together is an object, found from where the parts are, not from
   how the author grouped them (often by building step).
3. **Names.** An object takes the name of the author's submodel that covers
   the most of it, among those mostly inside it and covering at least a
   quarter of it, when the name says what it is (not a step number such as
   "step84-16", not a bent hose). An object holding most of the model is the
   set itself (its name), unless one submodel is most of it. The rest are
   numbered.
4. **Copies.** Objects with the same parts are one build with a count.
5. **Fitting to the set.** A model is not always the set's inventory: it may
   use an older mold's number, or draw an old set in today's colors. Where
   the model has more of a piece than the set's inventory and the inventory
   has more of another, the piece becomes that one: first a part alike (a
   mold variant, or the same number but for a letter) in the same color,
   then a part alike in any color. `Source.Placed` is how many of the set's
   pieces the model places after that (94% across the models in 2026).
6. **What counts as a build.** At least 10 counted pieces (plain parts and
   accessories), not mostly minifigure parts (a minifigure is apart, as in
   sets), and at least 40 when it has no name of its own (a stand, a detail
   that only rests on the rest). A set with fewer than two builds has none:
   one build is the set.

A build is a catalog entry like a set (`catalog.Build`, its number
`SET/slug`, such as `1234-1/station`), with its set's year and theme, so the
theme filter and every view treat it like one. It is never evidence of what
a lot came from (the likely order is sets only) and never weighs how rare a
part is. A set sold under two numbers shows the builds of one.

## Where they show

- The overview's Builds section: builds 60% or more here whose set is not
  complete (a complete set's builds say nothing more).
- A set's panel lists its builds, each against what the view leaves, and
  says how much of the set its model places.
- A build's panel is a set's: what is missing and found, its picture, its
  set, and the model it was read from.
- The filter's kind has Builds; a build is in the sort-out queue like a set.

## Pictures

`bin/render-builds -data DIR` draws every build not yet drawn into
`set-models/pictures/SET/slug.png` (320 pixels square by default, from
above and to the front left, in LDraw's colors with its edge lines;
`internal/ldraw/render.go`), on every core. pile serves them at
`/pictures/builds/SET/slug.png`: a build's picture once it is drawn, else
a redirect to its set's, so new pictures show without a restart. `-again` draws them all again (after
the drawing or the split changes), `-set NUM` one set's.

## Checking

`bin/check-models -data DIR` loads the catalog as pile does and reports, for
every set with a model, how many of its pieces the model places, its builds
and their sizes, and what it left out; for every collection model, its
pieces and what was left out (collections.md). `-set NUM` reports one;
`-cpuprofile FILE` profiles the load.
