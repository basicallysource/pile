# Running pile

<!-- human -->
pile shows what your sorters sorted and which LEGO sets it adds up to. To
run it you give it three things: Rebrickable's catalog (free to download),
the parts catalog from a Hive, and the records of a sorter.
<!-- /human -->

pile is one Go binary that reads a data folder into memory at start and
serves the app and its API from it. Nothing is downloaded or fetched at run
time except part and set pictures, which the browser loads from Rebrickable's
site. There is no demo data: without a sorter's records there is nothing to
show, and `parts.db` has to come from a Hive.

## Build

Needs on `PATH`:

- Go 1.27 or newer (`go.mod`).
- `buf` ([buf.build](https://buf.build)).
- `protoc-gen-go` and `protoc-gen-connect-go`:
  `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` and
  `go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest`
  (they land in `$(go env GOPATH)/bin`).
- Node 22 or newer with corepack, which provides pnpm (`build.sh` runs
  `corepack pnpm`; corepack ships with Node 22, newer Node needs
  `npm install -g corepack`). The app's `protoc-gen-es` comes from its own
  `node_modules`.

Then, from the repo root:

```sh
scripts/build.sh
```

It installs the app's packages, regenerates the contract (`buf generate`),
builds the app, vets, and writes `bin/pile` (the app is embedded in it) and
the other commands beside it: `pull-hive`, `collect-time`, `export-pieces`,
`import-custom-models`. The generated code is committed, so a build that
changes it shows in `git status`.

## The data folder

`bin/pile -data DIR -addr :PORT` (defaults `data` and `:8790`) reads:

| In `DIR` | Needed | What it is and where it comes from |
| --- | --- | --- |
| `rebrickable/` | yes | Rebrickable's catalog, as gzipped CSVs from [rebrickable.com/downloads](https://rebrickable.com/downloads/): `colors`, `part_categories`, `parts`, `part_relationships`, `themes`, `sets`, `inventories`, `inventory_parts`, `minifigs`, `inventory_minifigs`, each as `NAME.csv.gz`. They need no account or key. |
| `parts.db` | yes | A sorter Hive's parts catalog (SQLite). pile reads three tables from it: `part_bricklink_ids` (which BrickLink id is which Rebrickable part), `colors` (its `extra` JSON lists each color's BrickLink ids) and `part_geometry` (part sizes). **This is not rebuilt by pile and is slow to rebuild at all:** the Hive builds it itself, by syncing from Rebrickable's API with a key, which takes hours. Take it from a running Hive (`data/profile_builder/parts.db` in the [Hive](https://github.com/basicallysource/sorter-v2/tree/main/software/hive); copy it with SQLite's backup API, not `cp`, while the Hive runs) or from whoever runs one. |
| `lots.json` | yes | The collection's lots: each a named stretch of sorted pieces. See below. |
| `machines/NAME/local_state.sqlite` | one of these two | A sorter's own records, copied off the machine (its `local_state.sqlite`; pile reads only the `piece_records` table). |
| `hive.sqlite` | one of these two | Records pulled from a Hive with `bin/pull-hive`: [hive.md](hive.md). |
| `custom-models.json` | no | Free custom models, [custom-models.md](custom-models.md). Skipped when absent. |

`lots.json` is a list of lots. Each has an `id`, a `name`, a `description`,
and exactly one source: `"machine": "NAME"` (a folder under `machines/`) or
`"hive": "mine"` (see [hive.md](hive.md)), optionally bounded by `from` and
`until` days (`YYYY-MM-DD`). For one copied sorter:

```json
[
  {"id": "mine", "name": "My sorter", "description": "Everything it sorted.", "machine": "mysorter"}
]
```

The records pile reads from `piece_records` are the rows with
`classification_status = 'classified'`, `dead = 0` and a `part_id`, using the
columns `part_id`, `part_name`, `color_id`, `color_corrected_id`,
`color_name`, `confidence`, `seen_at` (unix seconds) and `part_correct`.
Part and color ids are BrickLink's.

## Start and check

```sh
bin/pile -data DIR -addr :8790
```

It reads the whole folder before it listens: seconds for a small lot, about
3 minutes for a big one, and each lot's time-to-collect numbers work out in
the background after that (about 1.5 more minutes for large lots). The log
says what it loaded; the page is up once it prints that it is listening.
New data in the folder means a restart. To keep it up as a service, see
"Served as a service" in [AGENTS.md](../AGENTS.md).

Open the address in a browser: the collection page lists the lots, and a lot
opens to its parts, its sets and its time to collect. The API is Connect, in
`proto/pile/v1/pile.proto`.

`go test ./...` runs the tests.

## Where this came from

[README.md](../README.md) lists where the data, the pieces and the design
come from. The look is the Sorter design system, copied in: docs/design-system.
