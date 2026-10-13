// Package match works out which LEGO sets and custom models a pile of sorted
// pieces makes up.
//
// A set is scored by the pieces of the pile it could claim, line by line,
// capped at what it needs. A plain fraction would rank every small box of
// 2x4 bricks first, so each part (in exact-color matching, each part in a
// color) is weighted by how rare it is across all sets (its inverse document
// frequency): a set's unusual pieces are the evidence that it is there.
//
// Matching is exact (a line needs its part in its color) or any-color (any
// color of its part will do; the pieces of the right color are taken first,
// and the rest are counted as wrong-color). A part stands for its mold
// variants and alternates, which look alike to a camera.
//
// Each line of a set has a class (plain, printed, minifigure part,
// accessory); a view's Counting says which count toward completeness. One
// that does not is still found, and taken with its set.
//
// Index holds every set's lines, built once from the catalog. Pile
// holds a lot's pieces; Take takes a set's pieces out of it, so a sort-out
// queue leaves a smaller pile for everything after it. Stack is several
// lots' piles in priority order and their sum: sets are matched against the
// sum, and what one takes is drawn from the first lot that has it. Explain
// orders the sets a pile most likely came from.
package match
