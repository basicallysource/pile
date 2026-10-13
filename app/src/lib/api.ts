// The PileService client, and each of its calls as a Source the pages ask
// through (lib/data, docs/loading-data.md). Every request and response shape
// comes from proto/pile/v1/pile.proto, generated into lib/gen.
import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { PileService } from './gen/pile/v1/pile_pb';
import { Source } from './data/query.svelte';

export const pile = createClient(PileService, createConnectTransport({ baseUrl: '/' }));

// pile's data changes only when it restarts (a reload picks that up), so an
// answer lasts the visit; the bins are the exception.
export const sources = {
	lots: new Source(pile.listLots, { doing: 'Loading the collection' }),
	overview: new Source(pile.getOverview, { doing: 'Matching the sets' }),
	sets: new Source(pile.listSets, { doing: 'Matching every set' }),
	parts: new Source(pile.listParts, { doing: 'Counting the parts' }),
	set: new Source(pile.getSet, { doing: "Counting the set's pieces" }),
	collectTimes: new Source(pile.getCollectTimes, {
		doing: 'Working out the times',
		settled: (r) => r.fittedUnix !== 0n && !r.sets.some((s) => s.pending)
	}),
	pieces: new Source(pile.findPieces, { doing: 'Searching the bins', changes: true })
};
