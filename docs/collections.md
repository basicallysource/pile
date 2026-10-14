# Collections

<!-- human -->
A collection is one designer's custom models, gathered from their own site
or wherever their digital files are, and matched against your pieces like
sets. Each collection has its own section on a lot's overview listing every
one of its models with a piece there, most complete first, and is a theme in
the filter, so you can list only its models.
<!-- /human -->

A collection is a folder in the data folder's `collections/`, named for the
collection's id, holding `collection.json` and each model's file. Nothing is
fetched: gathering is by hand, from wherever the designer or others share
the files, and the folder stays in the data folder (a designer's terms may
not let the files be passed on).

```json
{
  "name": "The collection's name",
  "designer": "Who designed the models",
  "url": "https://the-designers-site.example",
  "models": [
    {
      "id": "short-id",
      "name": "The model's name",
      "year": 2024,
      "url": "https://its-page-on-the-designers-site.example",
      "image": "https://a-picture-of-it.example/x.jpg",
      "file": "short-id.csv",
      "author": "Who made the file (the designer, or whoever made a digital copy)",
      "source": "https://where-the-file-came-from.example",
      "notes": ""
    }
  ]
}
```

A model's number in pile is `FOLDER/id`. Its parts are read from `file`,
by its extension (`internal/catalog/collections.go`):

- `.csv`: a parts list exported from BrickLink Studio (`BLItemNo`,
  `BLColorId`, `LdrawId`, `LDrawColorId`, `Qty`; the totals at its end are
  skipped) or Rebrickable (`Part`, `Color`, `Quantity`, `Is Spare`).
- `.xml`: a BrickLink wanted list or inventory (`ITEMID`, `COLOR`, `MINQTY`
  or `QTY`), as BrickLink, Mecabricks and Studio export it.
- `.ldr`, `.mpd`: an LDraw model, read as builds read one (builds.md),
  every part counted.

BrickLink ids go through the Hive's mapping (`parts.db`); a number it does
not know (an older BrickLink number) is taken as the catalog part it most
likely is (`Catalog.LikePart`: the same number, LDraw's file of that name,
or the same number but for a letter). What still has no catalog part
(custom printed parts, some printed minifigure parts) is left out and
listed by `bin/check-models`. A Studio `.io` file is a zip holding an LDraw
model; prefer the parts list exported beside it.

In pile a collection's models are custom models (`Kind` custom) with the
collection's name as their theme and theme group, and their designer.
`GetOverview` sends each collection as a section of its own
(`collections`): every model with a piece there, whatever its share, most
complete first, under the filter like any set.
