// The PileService client. Every request and response shape comes from
// proto/pile/v1/pile.proto, generated into lib/gen.
import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { PileService } from './gen/pile/v1/pile_pb';

export const pile = createClient(PileService, createConnectTransport({ baseUrl: '/' }));

// A call whose answers are kept for the visit, by request. A page come back
// to draws from them at once, so it lands where it was scrolled instead of
// loading again. pile's data changes only when it restarts, and a reload
// picks that up.
function remember<Req, Res>(call: (req: Req) => Promise<Res>) {
	const answers = new Map<string, Res>();
	const key = (req: Req) => JSON.stringify(req);
	return {
		// The answer already had, if any.
		peek: (req: Req): Res | undefined => answers.get(key(req)),
		get: (req: Req): Promise<Res> => {
			const had = answers.get(key(req));
			if (had) return Promise.resolve(had);
			return call(req).then((r) => {
				answers.set(key(req), r);
				return r;
			});
		}
	};
}

export const lotAnswers = remember((req: Parameters<typeof pile.getLot>[0]) => pile.getLot(req));
export const setsAnswers = remember((req: Parameters<typeof pile.listSets>[0]) => pile.listSets(req));
export const partsAnswers = remember((req: Parameters<typeof pile.listParts>[0]) => pile.listParts(req));
export const setAnswers = remember((req: Parameters<typeof pile.getSet>[0]) => pile.getSet(req));
