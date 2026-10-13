// How lots are being looked at (docs/views.md). The lots are the route's:
// `/lots/big-box,goodwill-box` is a stack of two, in priority order, so a
// link carries them. The rest is kept in the browser, so it lasts across
// pages and reloads: any-color matching, whether sets of loose bricks show,
// and each stack's sort-out queue (most wanted first). Every page reads it
// and the server works from it.
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
	// Each stack's queue, by the stack as the route has it.
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

	queue(stack: string): string[] {
		return this.sortOut[stack] ?? [];
	}

	private setQueue(stack: string, q: string[]) {
		this.sortOut = { ...this.sortOut, [stack]: q };
		this.save();
	}

	toggle(stack: string, num: string) {
		const q = this.queue(stack);
		this.setQueue(stack, q.includes(num) ? q.filter((n) => n !== num) : [...q, num]);
	}

	move(stack: string, num: string, step: -1 | 1) {
		const q = [...this.queue(stack)];
		const i = q.indexOf(num);
		const j = i + step;
		if (i < 0 || j < 0 || j >= q.length) return;
		[q[i], q[j]] = [q[j], q[i]];
		this.setQueue(stack, q);
	}

	clear(stack: string) {
		this.setQueue(stack, []);
	}

	// The View the server works from, for a stack.
	of(stack: string): MessageInitShape<typeof ViewSchema> {
		return {
			lots: lotsOf(stack),
			anyColor: this.anyColor,
			showBulk: this.showBulk,
			sortOut: this.queue(stack)
		};
	}
}

export const view = new ViewState();

// The stack in the route, as it has it: lot ids, first first, by commas.
export function routeStack(): string {
	return page.params.lots ?? '';
}

export function lotsOf(stack: string): string[] {
	return stack.split(',').filter(Boolean);
}

// This page, the set open on it included, for other lots.
export function stackHref(lots: string[]): string {
	const rest = page.url.pathname.split('/').slice(3).join('/');
	return `/lots/${lots.join(',')}${rest ? `/${rest}` : ''}${page.url.search}`;
}

export type { View };
