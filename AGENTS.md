# AGENTS.md

<!-- human -->
pile: what a sorter sorted, as a bulk catalog (every part in every color, with counts, downloadable as a BrickLink list), and which LEGO sets those pieces make up, ranked, with how complete each set is.
<!-- /human -->

Read `docs/writing-docs.md` first (how the docs are written, and what each
one is), then `AGENTS.local.md` and `docs.local/` if they exist (what stays off
the public record).

## How this program is kept

- **You maintain it.** This codebase is yours to keep in good shape. The
  person it belongs to will not read the code: keeping it clean, consistent
  and plainly correct is your job. When you see something wrong or untidy
  while you are in there, fix it.
- **Keep it laid out so it looks planned.** No file is allowed to become a
  heap, this one and the docs included. When a file grows long or starts
  covering two things, break it into files in a folder named for what they
  share, each named for what it holds. Nobody checks in on this for you:
  someone opening the repo at any moment should think it well laid out, as
  if its makers knew what they were building from the start.
- **Build every feature as if it had been planned from the start.** A new
  feature reshapes what is around it: names, types, files and docs change so
  the result reads as designed, not bolted on. No side paths, compatibility
  shims or flags grafted onto the old shape. When something is replaced, the
  old version goes in the same change. Git is the backup.
- **File the splinters.** When something outside your task gets in the way
  (a tool that fights you, a confusing error, a small bug, a missing test, a
  doc that is wrong) and fixing it is not part of the task, open an issue on
  the GitHub repo it belongs to and carry on. Say what you tried, the error
  verbatim, where it comes from, and what would have helped. Search open
  issues first and add to one that exists. If the fix is small and in scope,
  fix it instead.

## Docs

- **`README.md` is for people, and short**: the name, what it is and what
  it is for in a sentence or two, and nothing else. Directions,
  architecture and how to work on it go here and in `docs/`, never in the
  README.

- **`docs/` is committed and may be public.** Write it so a stranger could
  read it.
- **`docs.local/` and `AGENTS.local.md` are ignored by git** and never leave
  this machine: the local environment (machines, paths, accounts), and
  anything the owner would not say publicly. When in doubt, it goes local.
- **One doc serves both readers.** Every doc is written for you, with the
  parts meant for the person marked inline; the format is in
  `docs/writing-docs.md`. When the code changes, the docs that describe it change
  in the same commit.
- **A project with a UI keeps its design system in `docs/design-system/`.**
  Every screen uses its tokens and rules, and a style a screen needs that it
  lacks is added there first. It stays minimalist unless the owner says
  otherwise.

## Working on it

- **Shape.** One Go binary (`cmd/pile`) loads a data folder at start and
  serves the app and the API from memory: `rebrickable/` (Rebrickable's CSV
  downloads), `parts.db` (a sorter Hive's parts database, for
  BrickLink-to-Rebrickable part and color ids), `custom-models.json` (free
  custom models: `docs/custom-models.md`), `lots.json` (the collection's lots:
  each a named stretch of one sorter's records) and
  `machines/<name>/local_state.sqlite` (each sorter's records). Fresh data
  means a restart.
- **Lots and views.** The collection is lots (`internal/lots`); every request
  about a lot carries a View: the lot, any-color matching, and a sort-out
  queue whose sets take their pieces first (`internal/server/view.go`).
- **Contract.** `proto/pile/v1/pile.proto`, served with Connect;
  `buf generate` writes `gen/` (Go) and `app/src/lib/gen/` (TypeScript).
  Generated code is committed.
- **Matching** is `internal/match` (its package comment says how sets are
  scored, in exact or any color, and how the likely order is picked). Pieces
  come from `internal/machine`, the catalog (sets and custom models) from
  `internal/catalog`, the handlers from `internal/server`.
- **App.** SvelteKit 5 + Tailwind, static, in `app/`, embedded into the
  binary (`embed.go`). It follows the Sorter design system
  (`docs/design-system/README.md`).
- **Build:** `scripts/build.sh` (packages, generate, app build, vet, binary
  to `bin/pile`). Run: `bin/pile -data DIR -addr :PORT`.
