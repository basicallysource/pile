# How long to collect a set

<!-- human -->
How long one sorter, fed bulk LEGO at random, would take to come across every
piece of a set: in exact colors, in near colors (old and new grays and
browns count as one), and in any color. The mix of pieces comes from what
your sorters actually sorted; the speed from how fast they sort. It also says
how long to half, nine tenths and 99% of the set, how sure the answer is, and
which pieces hold it up the longest. Every lot has it on its "Time to
collect" page, worked out again whenever pile starts with fresh data; add any
set there by its number.
<!-- /human -->

The page (`app/src/routes/lots/[lot]/collect`) asks `GetCollectTimes`. The
server (`internal/server/collect.go`) works it out in the background from
the data it loaded: at start it fits every lot's model (smallest lot first),
then times the standard sets (`internal/collect/sets.txt`), then any set a
page asks for, one at a time, with 50 resamplings; the page shows what is
done and asks again while anything is on its way.

`bin/collect-time -data DIR -lots LOT,LOT [-sets NUM,NUM] [-rate N] [-json FILE]`
(the standard sets when `-sets` is empty) prints, for each lot (a stream) and each set, the hours of sorting to 50%,
90% and 99% of the set's pieces, and to all of them; `-json` writes every
number, the fitted model, the sorting time per machine and each set's slowest
pieces. The method is in the package comment of `internal/collect`; in short:

- **Stream.** The lot's classified pieces, in Rebrickable's parts (a part
  stands for its mold variants) and colors. Pieces of unknown color count in
  any-color mode only; every classified piece takes sorting time.
- **Shares.** A part-color's share of the stream is its count smoothed toward
  its share of every set LEGO sold (empirical Bayes, a Gamma-Poisson model
  whose three parameters are fitted to the stream by maximum likelihood). A
  part-color the sorters never saw gets a share from LEGO's use of it, scaled
  to the stream. Shares are fitted for exact part-colors; near and any
  color add up the shares of the part-colors they stand for.
- **Time.** Counting pieces as a Poisson process, the chance the set is
  complete after t pieces is the product over its part-colors of
  P(Poisson(p t) >= need): a coupon collector with quotas, solved for its
  median and its 10th and 90th percentile. The share found by t is the sum of
  E[min(Poisson(p t), need)] over the set's pieces.
- **Uncertainty.** "runs" is the chance spread of one sorter's run with the
  shares known; "shares" is the 5th to 95th percentile of the median over
  draws of the shares from their posterior (`-samples`).
- **Left out of a set.** Printed parts and stickers, minifigures and their
  parts, minidoll parts, parts that are not building pieces (Duplo,
  electronics, string, cloth and the like: `specialtyCategories`), and molds
  in fewer than `MinSets` (10) sets LEGO sold.
- **Near colors.** `NearColors`: Light Bluish Gray with Light Gray, Dark
  Bluish Gray with Dark Gray, Reddish Brown with Brown, Flat Silver with
  Pearl Light Gray.
- **Rate.** Classified pieces an hour of sorting, from the lot's Hive
  records: a machine sorts while the gap between one piece and the next is a
  minute or less, as Hive counts active time. `-rate` sets one instead.
- **Check.** `-simulate N` also runs the sorter piece by piece N times per
  set with the same shares, to check the closed form; `go test
  ./internal/collect` checks it on a small set.

The model assumes well-mixed bulk. Real bulk comes in lots, so a set's rare
pieces tend to arrive together, and a lot that held the set makes it far
faster than these times.

`bin/export-pieces -data DIR -lot LOT` writes a lot's pieces as CSV (counted
by machine, part and color, in BrickLink's and Rebrickable's ids, with each
part's category and how many sets have it), for analysis outside pile.
