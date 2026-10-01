# Custom models

<!-- human -->
Free custom models (MOCs) from Rebrickable, matched against each lot like
the sets LEGO sold.
<!-- /human -->

Rebrickable's API gives a custom model's part list only to its owner, so the
models are gathered by hand and kept as data, not fetched by pile:

1. Make a buildable parts list of a lot in a Rebrickable account (its BrickLink
   XML from the Parts page, or through the API's part list endpoints).
2. Signed in, on rebrickable.com/build/, run `scripts/custom-models/1-search.js`
   in the browser console, then `2-part-lists.js` in the same page. The first
   runs Rebrickable's build search for free custom models in bands of part
   count (one search returns at most 500); the second fetches each good
   match's parts as Rebrickable's parts CSV, paced so the site's rate limit
   holds. Each saves its results as a download.
3. `go run ./cmd/import-custom-models -out DATA/custom-models.json FILE...`
   merges the downloads into the custom models file, which pile loads at
   start (restart it to see new ones).

pile scores custom models itself, from their parts, exactly as it scores sets:
Rebrickable's own percentages only chose which to gather. They are never
evidence of what a lot came from (the likely order is sets LEGO sold only).
