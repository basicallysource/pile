# Records from a Hive

<!-- human -->
pile reads what your sorters reported to Hive. Give it a Hive API key and
name your sorter in a lot: pile reads that sorter's pieces, your color
corrections and the sorter's own lots at start, and again every ten minutes,
so a lot grows as runs are sorted, with no copying and no restart. Make the
key in Hive's Settings with the records:read scope; it reads only your own
sorters.

pile can also pull everything a Hive user may read (every sorter of the
fleet, for an admin) for lots of everything your sorters ever sorted. A lot
can be left off the collection page; it then opens only from the menu on a
lot's name.
<!-- /human -->

There are two ways in, both in `internal/hive` and both read-only:

| | The records API | The fleet pull |
| --- | --- | --- |
| Lot source | `"hiveMachine": NAME` | `"hive": "mine"`, `"all"` or a machine |
| Reads | the key owner's machines that lots name | every machine a signed-in user may read |
| With | an API key (`records:read`) in `PILE_HIVE_KEY` | a user's email and password |
| Run by | `bin/pile` itself, at start and every `-refresh` | `bin/pull-hive`, by hand, then a restart |
| Into | `records.sqlite` | `hive.sqlite` |
| Has | people's color verdicts, color models' answers, sorter categories, the sorter's lots and runs | the classifier's answer only |

## The records API

pile reads one of the key owner's machines for each machine named by a lot's
`hiveMachine` (its Hive name, in any case, or its id).

- **The key.** A Hive API key with the `records:read` scope, made in Hive's
  Settings. It sees only its owner's machines. pile takes it from the
  environment variable `PILE_HIVE_KEY` and nowhere else (never a flag, so it
  never shows in a process list); it goes only in the `Authorization:
  Bearer` header (`internal/hive/key.go`) and is never logged or put in an
  error. A 401 or 403 says the key is wrong, revoked or under-scoped. Every
  request names itself `User-Agent: pile`: the proxy in front of a Hive may
  refuse a client's default one, which shows as a 403 with
  `error code: 1010` and reads as a blocked client, not a bad key.
- **The routes** (`internal/hive/machines.go`): `GET /api/records/machines`
  lists the key's machines; `GET /api/records/machines/{id}/pieces?limit=500`
  gives a machine's pieces newest first (by `local_id`), with `next_cursor`
  passed back as `cursor` until it is null, a pause between pages so a pull
  never crowds a Hive; `GET /api/records/machines/{id}/sorting-runs` gives its
  lots and its runs (each from `started_at` until the next run starts,
  `ends_at`, null for the newest).
- **A pull** (`hive.PullMachines`) reads each machine whole, every time:
  corrections and labels change on old pieces too. It keeps a digest of what
  it read per machine; when the digest is the last pull's, nothing changed
  and only the pull time is written. Otherwise the machine's rows are
  replaced in one transaction, so pieces gone from Hive go. It says per
  machine whether it changed.
- **The store** `records.sqlite` (`internal/hive/machinestore.go` has the
  schema): `machines` (id, Hive name, `pulled_at`, digest), `pieces` (as
  Hive gave them: BrickLink ids, unix times, the people's labels and the
  models' answers as JSON lists), `lots` and `runs`. A lot's snapshot time is
  its machine's `pulled_at`, when what it serves was read.

### Refreshing

With `PILE_HIVE_KEY` set and a lot naming `hiveMachine`, `bin/pile`
(`cmd/pile/refresh.go`):

1. At start, pulls before it loads the lots. If that fails and
   `records.sqlite` exists, it logs the error and serves what is there; with
   no `records.sqlite` it exits with the error.
2. Every `-refresh` (default `10m`), pulls again. When a machine changed, it
   reads again only the lots naming it (the others, and `hive.sqlite`'s, are
   kept as they are), builds a new Service from them off to the side, and
   swaps it in behind the handler (`server.Live`): the page never goes down,
   and a request is answered by one whole Service. The old one's collect-time
   work stops; the new one's starts over. The catalog, the storage and the
   stars are shared by both (`server.Shared`), so a bin or star set while a
   refresh runs is kept.
3. Logs each refresh in one line: each machine, its pieces, whether it
   changed, and how long it took. A refresh that fails is logged and the
   lots from before stay served; a change read but not yet served is tried
   again on the next refresh.

`-hive URL` picks the Hive (the production one by default). Without the key,
pile serves `records.sqlite` as it is and says so. `lots.json` itself is
read only at start.

### A piece's color

Pieces lost (`dead`), not `classified`, marked the wrong part
(`part_correct` false) or without a part are left out. A piece's color
(`internal/hive/color.go`) is, in order:

1. A person's verdict: the newer of the machine operator's correction
   (`color_corrected_id` at `correction_updated_at`) and the newest of the
   machine owner's labels on Hive (`by_owner`); with neither, the newest
   label anyone else gave. A label that says the color can't be told, or
   gives none, is no verdict.
2. The newest answer (by `made_at`) of a color model run over the piece's
   pictures after the sort, Hive's own (`source` "hive") among them.
3. The machine's own `color_id`, from while it sorted.

The overview says how many pieces each colored: a person, each model by
name, or the sorter.

## The fleet pull

`bin/pull-hive -data DIR` signs in to a Hive as a user and copies every
machine that user may read into `DIR/hive.sqlite`.

- The sign-in is a Hive user's email and password: `HIVE_EMAIL` and
  `HIVE_PASSWORD`, or with `-login-stdin` two lines on stdin (email, then
  password) so the password never rides on a command line. A run signs in
  once and refreshes the session as it goes (`internal/hive/session.go`).
- It lists machines with `GET /api/machines?scope=all&include_archived=true`
  and reads each one's pieces from `GET /api/machines/{id}/pieces`, newest
  first, 200 a page with a pause between pages. A machine the user may not
  read answers 404 and is skipped.
- A later pull reads only what is new: it stops a little below the newest
  piece it has (`overlap` in `internal/hive/fleet.go`), so pieces still being
  classified then are read again. A machine whose pull stopped part way (it
  has pieces and no `machines` row) is then read on down from its oldest
  piece. `-full` reads everything again.
- `-hive URL` picks the Hive (the default is the production one);
  `-machines NAME,NAME` pulls only those machines (by name or id).

`hive.sqlite` holds `machines` (id, name, whether it is the user's own, when
pulled, pieces) and `pieces` (as Hive gave them: BrickLink part and color
ids, unix times). `internal/hive/fleetstore.go` has the schema. Its piece
list gives no person's corrections, so a lot from it counts the classifier's
answer for every piece. pile reads it only at start: restart after a pull.

## Lots from Hive

A lot in `lots.json` takes its pieces from exactly one of:

- `"hiveMachine": "NAME"`: one of the key owner's machines, read through the
  records API;
- `"hive": "mine"`, `"all"` or a machine's name or id: the fleet pull's
  records of the signed-in user's machines, of every machine, or of one.

`from` and `until` bound either by day. `"sorterLot": "NAME"` beside
`hiveMachine` takes the pieces of the runs that add to that machine's own
lot (by name): a run (from when it starts until the next one starts) adds
its pieces to a lot, or is another pass over pieces the lot already counted
and adds none. `"unlisted": true` keeps a lot off the collection page (it is
added with "Add a lot" at the top of any page of lots). A lot names the
machines its pieces came from only when there are three or fewer.

`"exclude"` leaves out stretches of one sorter's pieces, for pieces sorted a
second time (a box already counted, fed through again) that would otherwise
count twice: each is `{"machine": NAME, "from": RFC3339, "until": RFC3339 or
left out for no end, "why": ...}`, the machine by the name Hive gives it.

```json
[
  {"id": "big-box", "name": "Big box", "description": "The big box.", "hiveMachine": "mysorter", "sorterLot": "Big box"},
  {"id": "mine", "name": "My sorters", "description": "Everything they ever sorted.", "hive": "mine"}
]
```
