# Running pile

<!-- human -->
pile shows what your sorters sorted and which LEGO sets it adds up to. To
run it you give it three things: Rebrickable's catalog (free to download),
the parts catalog from a Hive, and a Hive API key that reads your sorters'
records.
<!-- /human -->

pile is one Go binary that reads a data folder into memory at start and
serves the app and its API from it. At run time it fetches only the records
of the machines its lots name, from a Hive's records API every few minutes
([hive.md](hive.md)); part and set pictures the browser loads from
Rebrickable's site. There is no demo data: without a sorter's records there
is nothing to show, and `parts.db` has to come from a Hive.

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
`import-custom-models`, `check-models`, `render-builds`. The generated code is committed, so a build that
changes it shows in `git status`.

## The data folder

`bin/pile -data DIR -addr :PORT` (defaults `data` and `:8790`) reads:

| In `DIR` | Needed | What it is and where it comes from |
| --- | --- | --- |
| `rebrickable/` | yes | Rebrickable's catalog, as gzipped CSVs from [rebrickable.com/downloads](https://rebrickable.com/downloads/): `colors`, `part_categories`, `parts`, `part_relationships`, `themes`, `sets`, `inventories`, `inventory_parts`, `minifigs`, `inventory_minifigs`, each as `NAME.csv.gz`. They need no account or key. |
| `parts.db` | yes | A sorter Hive's parts catalog (SQLite). pile reads four tables from it: `part_bricklink_ids` (which BrickLink id is which Rebrickable part), `colors` (its `extra` JSON lists each color's BrickLink and LDraw ids), `parts` (its `external_ids` list each part's LDraw files) and `part_geometry` (part sizes, and the LDraw file each was measured from). **This is not rebuilt by pile and is slow to rebuild at all:** the Hive builds it itself, by syncing from Rebrickable's API with a key, which takes hours. Take it from a running Hive (`data/profile_builder/parts.db` in the [Hive](https://github.com/basicallysource/sorter-v2/tree/main/software/hive); copy it with SQLite's backup API, not `cp`, while the Hive runs) or from whoever runs one. |
| `lots.json` | yes | The collection's lots: each a named stretch of sorted pieces. See below. |
| `records.sqlite` | for `hiveMachine` lots | The machines lots name, read through a Hive's records API with the key in `PILE_HIVE_KEY`. pile writes it itself, at start and every `-refresh`: [hive.md](hive.md). |
| `hive.sqlite` | for `hive` lots | Records of every machine a Hive user may read, pulled with `bin/pull-hive`: [hive.md](hive.md#the-fleet-pull). |
| `custom-models.json` | no | Free custom models, [custom-models.md](custom-models.md). Skipped when absent. |
| `set-models/` | no | 3D models of sets, which the builds within sets are read from, and the builds' pictures: [builds.md](builds.md). No builds when absent. |
| `ldraw/` | no | The LDraw parts library, unpacked (`complete.zip` from [ldraw.org](https://library.ldraw.org/updates?latest)): what LDraw parts are made of, their shapes and colors, for the builds and their pictures. Keep it on a fast disk: loading reads it many times. |
| `collections/` | no | Designers' collections of custom models, a folder each: [collections.md](collections.md). |
| `storage.json` | no | The bins the pieces are kept in, each at an address: [storage.md](storage.md). Empty when absent; pile writes it when the storage is set while it runs. |
| `stars.json` | no | The starred sets and custom models: [views.md](views.md#stars). Empty when absent; pile writes it when a star changes. |

`lots.json` is a list of lots. Each has an `id`, a `name`, a `description`,
and exactly one source: `"hiveMachine": "NAME"` (one of your machines on a
Hive, by its Hive name or id) or `"hive": "mine"` (the fleet pull; see
[hive.md](hive.md)), optionally bounded by `from` and `until` days
(`YYYY-MM-DD`). For one sorter:

```json
[
  {"id": "mine", "name": "My sorter", "description": "Everything it sorted.", "hiveMachine": "mysorter"}
]
```

A sorter keeps lots of its own: a run (from when it starts until the next
one starts) adds its pieces to a lot, or is another pass over pieces the lot
already counted. `"sorterLot": "NAME"` beside `"hiveMachine"` takes the
pieces of the runs that add to that lot, read from Hive with the machine:

```json
[
  {"id": "big-box", "name": "Big box", "description": "The big box.", "hiveMachine": "mysorter", "sorterLot": "Big box"}
]
```

A piece's color is a person's verdict when there is one, else the newest
color model's answer, else the sorter's own: the rule, and the pieces left
out, are in [hive.md](hive.md#a-pieces-color).

### The Hive key

Make an API key in the Hive's Settings with the `records:read` scope (it
reads only your own machines), keep it in a file only you can read, and give
it to pile in the environment, never as an argument:

```sh
PILE_HIVE_KEY="$(cat ~/.config/pile/hive-key)" bin/pile -data DIR -addr :8790
```

As a service, the unit reads it from such a file (`EnvironmentFile=`, a
line `PILE_HIVE_KEY=...`). `-hive URL` picks
the Hive (the production one by default) and `-refresh` how often pile reads
the machines again (default `10m`).

## Start and check

```sh
bin/pile -data DIR -addr :8790
```

It reads the whole folder before it listens: seconds for a small lot, about
3 minutes for a big one, and each lot's time-to-collect numbers work out in
the background after that (about 1.5 more minutes for large lots). The log
says what it loaded; the page is up once it prints that it is listening.
Records of the lots' machines are read again while it runs, a line in the
log each time; any other new data in the folder means a restart. To keep it
up as a service, see "Served as a service" in [AGENTS.md](../AGENTS.md).

Open the address in a browser: the collection page lists the lots, and a lot
opens to its parts, its sets and its time to collect. The API is Connect, in
`proto/pile/v1/pile.proto`.

`go test ./...` runs the tests.

## Where this came from

[README.md](../README.md) lists where the data, the pieces and the design
come from. The look is the Sorter design system, copied in: docs/design-system.
