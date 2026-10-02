# Records from a Hive

<!-- human -->
pile can read what your sorters reported to Hive, all of it, instead of a
copy taken off one sorter. Pull the records, then make a lot of everything
your sorters ever sorted, of one sorter, or of a stretch of days. A lot can be
left off the collection page; it then opens only from the menu on a lot's
name.
<!-- /human -->

## Pulling

`bin/pull-hive -data DIR` signs in to a Hive as a user and copies every
machine that user may read into `DIR/hive.sqlite`: a user's own machines,
or with an admin's sign-in every machine on the Hive. It only reads.

- The sign-in is a Hive user's email and password: `HIVE_EMAIL` and
  `HIVE_PASSWORD`, or with `-login-stdin` two lines on stdin (email, then
  password) so the password never rides on a command line. A run signs in
  once and refreshes the session as it goes.
- It lists machines with `GET /api/machines?scope=all&include_archived=true`
  and reads each one's pieces from `GET /api/machines/{id}/pieces`, newest
  first, 200 a page with a pause between pages. A machine the user may not
  read answers 404 and is skipped.
- A later pull reads only what is new: it stops a little below the newest
  piece it has (`overlap` in `internal/hive/pull.go`), so pieces still being
  classified then are read again. `-full` reads everything again.
- `-hive URL` picks the Hive (the default is the production one).

`hive.sqlite` holds `machines` (id, name, whether it is the user's own, when
pulled, pieces) and `pieces` (as Hive gave them: BrickLink part and color
ids, unix times). `internal/hive/store.go` has the schema.

Hive's piece list gives no person's corrections (a part marked wrong, a
corrected color), which a sorter's own records have, so a lot from Hive
counts the classifier's answer for every piece.

## Lots from Hive

A lot in `lots.json` takes its pieces from one of:

- `"machine": "NAME"`: a sorter's own records copied into
  `machines/NAME/local_state.sqlite`;
- `"hive": "mine"`, `"all"` or a machine's name or id: the pulled records of
  the signed-in user's machines, of every machine pulled, or of one.

`from` and `until` bound either by day. `"unlisted": true` keeps a lot off the
collection page (it opens from the lot menu on a lot's name). A lot names the
machines its pieces came from only when there are three or fewer.

```json
[
  {"id": "mine", "name": "My sorters", "description": "Everything they ever sorted.", "hive": "mine"},
  {"id": "everything", "name": "All records", "description": "Every piece in the records.", "hive": "all", "unlisted": true}
]
```
