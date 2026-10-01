// The pile as a BrickLink XML list (for a wanted list or an inventory upload),
// in the sorter's own BrickLink part ids and colors.
import type { PartCount } from '$lib/gen/pile/v1/pile_pb';

const escape = (s: string) =>
	s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

export function bricklinkXML(lots: PartCount[]): string {
	const items = lots
		.filter((l) => l.color?.bricklinkId)
		.map(
			(l) =>
				`<ITEM><ITEMTYPE>P</ITEMTYPE><ITEMID>${escape(l.bricklinkId)}</ITEMID><COLOR>${l.color!.bricklinkId}</COLOR><MINQTY>${l.count}</MINQTY></ITEM>`
		);
	return `<INVENTORY>\n${items.join('\n')}\n</INVENTORY>\n`;
}
