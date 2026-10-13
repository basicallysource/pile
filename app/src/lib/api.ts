// Each of the PileService's calls a page reads, as a Source it asks through
// (lib/data, docs/loading-data.md).
import { pile } from './client';
import { Source } from './data/query.svelte';
import { stars } from './pile/stars.svelte';

// Answers that list only starred sets depend on the stars, which the
// request does not carry.
const starred = (req: { view?: { filter?: { starredOnly?: boolean } } }) =>
	req.view?.filter?.starredOnly ? stars.revision : 0;

// pile's data changes only when it restarts (a reload picks that up), so an
// answer lasts the visit; the bins and the stars are the exceptions.
export const sources = {
	lots: new Source(pile.listLots, { doing: 'Loading the collection' }),
	overview: new Source(pile.getOverview, { doing: 'Matching the sets', version: starred }),
	sets: new Source(pile.listSets, { doing: 'Matching every set', version: starred }),
	parts: new Source(pile.listParts, { doing: 'Counting the parts' }),
	set: new Source(pile.getSet, { doing: "Counting the set's pieces", version: starred }),
	collectTimes: new Source(pile.getCollectTimes, {
		doing: 'Working out the times',
		settled: (r) => r.fittedUnix !== 0n && !r.sets.some((s) => s.pending)
	}),
	pieces: new Source(pile.findPieces, { doing: 'Searching the bins', changes: true })
};
