// How lots are being looked at: any-color matching, whether sets of loose
// bricks show, and each lot's sort-out queue (most wanted first). Kept in the browser, so it lasts across pages
// and reloads; every page reads it and the server works from it.
import { page } from '$app/state';
import type { View } from './gen/pile/v1/pile_pb';
import type { MessageInitShape } from '@bufbuild/protobuf';
import type { ViewSchema } from './gen/pile/v1/pile_pb';

const KEY = 'pile-view';

type Saved = { anyColor: boolean; showBulk: boolean; sortOut: Record<string, string[]> };

function load(): Saved {
	try {
		const s = JSON.parse(localStorage.getItem(KEY) ?? '');
		if (s && typeof s === 'object')
			return { anyColor: !!s.anyColor, showBulk: !!s.showBulk, sortOut: s.sortOut ?? {} };
	} catch {
		// Nothing saved, or storage is off.
	}
	return { anyColor: false, showBulk: false, sortOut: {} };
}

class ViewState {
	anyColor = $state(false);
	showBulk = $state(false);
	sortOut = $state<Record<string, string[]>>({});

	constructor() {
		const s = load();
		this.anyColor = s.anyColor;
		this.showBulk = s.showBulk;
		this.sortOut = s.sortOut;
	}

	private save() {
		try {
			localStorage.setItem(
				KEY,
				JSON.stringify({ anyColor: this.anyColor, showBulk: this.showBulk, sortOut: this.sortOut })
			);
		} catch {
			// Storage can be off; the view lasts the visit.
		}
	}

	setAnyColor(anyColor: boolean) {
		this.anyColor = anyColor;
		this.save();
	}

	setShowBulk(showBulk: boolean) {
		this.showBulk = showBulk;
		this.save();
	}

	queue(lotId: string): string[] {
		return this.sortOut[lotId] ?? [];
	}

	private setQueue(lotId: string, q: string[]) {
		this.sortOut = { ...this.sortOut, [lotId]: q };
		this.save();
	}

	toggle(lotId: string, num: string) {
		const q = this.queue(lotId);
		this.setQueue(lotId, q.includes(num) ? q.filter((n) => n !== num) : [...q, num]);
	}

	move(lotId: string, num: string, step: -1 | 1) {
		const q = [...this.queue(lotId)];
		const i = q.indexOf(num);
		const j = i + step;
		if (i < 0 || j < 0 || j >= q.length) return;
		[q[i], q[j]] = [q[j], q[i]];
		this.setQueue(lotId, q);
	}

	clear(lotId: string) {
		this.setQueue(lotId, []);
	}

	// The View the server works from, for the lot in the route.
	forLot(lotId: string): MessageInitShape<typeof ViewSchema> {
		return { lotId, anyColor: this.anyColor, showBulk: this.showBulk, sortOut: this.queue(lotId) };
	}
}

export const view = new ViewState();

// The lot in the route.
export function lotId(): string {
	return page.params.lot ?? '';
}

export type { View };
