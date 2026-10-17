# Docs

<!-- human -->
How this project's docs are written, and what each one covers.
<!-- /human -->

## One doc, two readers

Every doc is a Markdown file written for an agent working in the repo:
complete and exact, with file paths, commands, and the reasons behind
decisions. The parts that are also for the person the project belongs to are
marked inline as human sections:

```markdown
# Sync

<!-- human -->
Your changes reach every device within a second, and nothing is lost if
the network drops.
<!-- /human -->

Every change is a mutation queued in `client/queue.rs` and applied by the
same merge rules on both sides...
```

- A marker is a line of its own: `<!-- human -->` opens a human section and
  `<!-- /human -->` closes it. Sections do not nest. Markers inside code
  blocks (like the example above) do not count.
- The markers are HTML comments, so the file reads whole in any editor or on
  GitHub.
- Agents read the whole file, human sections included. Do not repeat in the
  agent text what a human section already says.
- **The human view** of a doc is its first heading and its human sections,
  in order, and nothing else. A renderer of the docs shows only that; a doc
  with no human sections does not appear in it.
- A human section says what something is, what it does for the person, how
  to use it, and what is theirs to decide. Plain words: no file paths, code
  or internals. It must read on its own, since everything around it is
  hidden, and it stays short.

## What is here

- `writing-docs.md`: this.
- `running.md`: building pile and what its data folder needs.
- `custom-models.md`: free custom models from Rebrickable, gathered by hand.
- `builds.md`: the builds within sets (a building, a vehicle), read from 3D
  models of the sets, and their pictures.
- `collections.md`: designers' collections of custom models.
- `hive.md`: reading records from a Hive (the records API with a key, kept
  fresh while pile runs, and the fleet pull), and lots made of them.
- `collect-time.md`: how long one sorter takes to come across a set's pieces.
- `storage.md`: bins at addresses, what each holds, and finding pieces in them.
- `views.md`: looking at lots: a stack of them in priority order, the
  color mode, the sort-out queue.
- `changes.md`: changes to sets, kept with them: a line's colors, other
  parts that make one of its part, whether it counts.
- `loading-data.md`: how the app loads and refreshes what it shows, and keeps
  the reader's place.

- `design-system/`: how the UI looks and behaves (projects with a UI).
