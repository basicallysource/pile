// The PileService client. Every request and response shape comes from
// proto/pile/v1/pile.proto, generated into lib/gen. Pages read through the
// sources in api.ts; only what the reader changes on the server (stars)
// calls it directly.
import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { PileService } from './gen/pile/v1/pile_pb';

export const pile = createClient(PileService, createConnectTransport({ baseUrl: '/' }));
