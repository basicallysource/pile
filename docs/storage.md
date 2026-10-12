# Storage: bins and finding pieces

<!-- human -->
Every bin your sorted pieces go into has an address, the label you find it
by on the shelf ("3D-04"), and holds pieces from one lot: the pieces a sorter
sent to one or more of its categories, or the rest of a lot that no other bin
holds. When a category is sorted again into finer ones, each finer category
gets a bin of its own.

Find pieces, at the top of every page, searches what the bins hold as you
type: a part's name or number, a color, a type of part, or a bin's address. Filters
for color, type, bin and lot narrow it down. Each part in a color shows how
many there are and a badge for every bin they are in, with how many each
holds. It works on a phone.
<!-- /human -->

## The storage

`storage.json` in the data folder, read at start (`internal/storage`). With
no file the storage is empty and Find pieces says so. It is replaced whole at
run time with `SetStorage`, which checks it, puts the pieces in the new bins,
writes the file (to `storage.json.new`, then renamed over it) and only then
serves it, so a restart serves what was last set.

```json
{
	"bins": [
		{ "address": "3D-04", "name": "Studs on side & brackets", "lot": "big-box", "categories": ["snot"] },
		{ "address": "1A-12", "name": "Everything else", "lot": "big-box" }
	]
}
```

- `address`: required, unique (case does not matter). Free text: whatever
  the shelf label says.
- `name`: what it holds, in words. Bins of the same name across lots are one
  choice of the Bin filter.
- `lot`: a lot id from `lots.json`.
- `categories`: the sorter profile's category ids whose pieces it holds (a
  sorter's bin emptied into it, or several). No categories: the lot's pieces
  that no other bin of that lot holds. A lot's category, and a lot's rest,
  can be in one bin only; `Storage.Check` refuses anything else, at start and
  in `SetStorage`.

A piece's category is `piece_records.category_id` from a sorter's own records
(`records.Piece.Category`). Records pulled from a Hive have no category, so a
Hive lot's pieces can only be in its rest bin. A finer category from a later
pass is in the records of that pass, so it fills a bin only for a lot whose
records include that pass.

## The calls

All on `PileService` (`proto/pile/v1/pile.proto`), served with Connect, so
plain JSON over POST works too:

```sh
curl -s -H 'Content-Type: application/json' -d '{}' localhost:8790/pile.v1.PileService/GetStorage
curl -s -H 'Content-Type: application/json' -d @bins.json localhost:8790/pile.v1.PileService/SetStorage
curl -s -H 'Content-Type: application/json' -d '{"query":"red","type":"Bricks Special"}' localhost:8790/pile.v1.PileService/FindPieces
```

`SetStorage` takes `{"bins": [{"address", "name", "lotId", "categories"}]}`
and answers with each bin and the pieces it holds. `{"bins": []}` empties the
storage.

## Finding pieces

`internal/server/stock.go`. At start and on every `SetStorage`, every piece of
each lot a bin names goes to its lot's bin for its category, else its lot's
rest bin, else nowhere; pieces the catalog cannot place are left out. The
result is each part in a color with its count in each bin, kept in memory and
swapped whole, so a search never sees half a storage.

`FindPieces` splits the query into lowercase words of letters and digits;
each must start a word of the part's name or number, its BrickLink id, its
color, its type (Rebrickable's part category), or the bin's address. Bin
names are left to the Bin filter: a bin named "Slopes: red, yellow, orange,
brown" would otherwise answer "red" with its yellow slopes.
A part in several bins keeps only the bins that match, so a query naming a
bin shows only what that bin holds. Parts where every query word is a whole
word come first ("red" puts Red before Reddish Brown), then most pieces. It
sends at most `limit` parts (200 by default) and the totals past it.

Each filter's choices come back with the pieces each would find given the
query and the other filters, so a choice that would find nothing is not
offered (the app keeps a chosen one in its list at 0).

The app's side is `app/src/lib/pile/FindPieces.svelte`, opened from the top
bar (`+layout.svelte`) or with `/`. It asks again on every keystroke and
shows only the latest answer.
