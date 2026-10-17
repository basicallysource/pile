// Each of the PileService's calls a page reads, as a Source it asks through
// (lib/data, docs/loading-data.md).
import { pile } from './client';
import { Source } from './data/query.svelte';
import { changes } from './pile/changes.svelte';
import { stars } from './pile/stars.svelte';

// Answers matched through the sets depend on the sets' changes, and those
// that list only starred sets on the stars: neither is in the request.
const changed = (req: { view?: { filter?: { starredOnly?: boolean } } }) => [
	changes.revision,
	req.view?.filter?.starredOnly ? stars.revision : 0
];

// pile's data changes only when it restarts or reads fresh records of its
// lots' machines from a Hive (a reload picks either up), so an answer lasts
// the visit; the bins, the stars and the sets' changes are the exceptions.
export const sources = {
	lots: new Source(pile.listLots, { doing: 'Loading the collection' }),
	overview: new Source(pile.getOverview, { doing: 'Matching the sets', version: changed }),
	sets: new Source(pile.listSets, { doing: 'Matching every set', version: changed }),
	parts: new Source(pile.listParts, { doing: 'Counting the parts' }),
	set: new Source(pile.getSet, { doing: "Counting the set's pieces", version: changed }),
	standIns: new Source(pile.getStandIns, {
		doing: 'Finding parts to make it from',
		version: changed
	}),
	collectTimes: new Source(pile.getCollectTimes, {
		doing: 'Working out the times',
		settled: (r) => r.fittedUnix !== 0n && !r.sets.some((s) => s.pending)
	}),
	pieces: new Source(pile.findPieces, { doing: 'Searching the bins', changes: true }),
	findSets: new Source(pile.findSets, { doing: 'Finding sets' }),
	findParts: new Source(pile.findParts, { doing: 'Finding parts' })
};
