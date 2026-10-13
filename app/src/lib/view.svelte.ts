// How lots are being looked at (docs/views.md). The lots are the route's:
// `/lots/big-box,goodwill-box` is a stack of two, in priority order, so a
// link carries them. The rest is kept in the browser, so it lasts across
// pages and reloads: any-color matching, the filter (which sets are listed,
// docs/views.md#the-filter), whether sets are grouped by theme, and each
// stack's sort-out queue (most wanted first). Every page reads it and the
// server works from it.
import { page } from '$app/state';
import { Kind, type View } from './gen/pile/v1/pile_pb';
import type { MessageInitShape } from '@bufbuild/protobuf';
import type { FilterSchema, ViewSchema } from './gen/pile/v1/pile_pb';

const KEY = 'pile-view';

export type Filter = {
	hideLooseBricks: boolean;
	hideMostlyBasic: boolean;
	hideFewParts: boolean;
	hideNothingUnusual: boolean;
	kind: Kind;
	themeGroup: string;
	licensedOnly: boolean;
	minPieces: number;
	starredOnly: boolean;
};

// The filter's presets: which kinds of set they hide. Each leaves the
// theme, kind, size, licensed and starred filters as they are.
export const presets = {
	interesting: { hideLooseBricks: true, hideMostlyBasic: true, hideFewParts: true, hideNothingUnusual: true },
	models: { hideLooseBricks: true, hideMostlyBasic: false, hideFewParts: false, hideNothingUnusual: false },
	everything: { hideLooseBricks: false, hideMostlyBasic: false, hideFewParts: false, hideNothingUnusual: false }
} as const;
export type Preset = keyof typeof presets;
type Rules = (typeof presets)[Preset];

// The filter a page opens with, until one is chosen.
const fresh: Filter = { ...presets.interesting, kind: Kind.UNSPECIFIED, themeGroup: '', licensedOnly: false, minPieces: 0, starredOnly: false };

type Saved = { anyColor: boolean; byTheme: boolean; filter: Filter; sortOut: Record<string, string[]> };

function load(): Saved {
	try {
		const s = JSON.parse(localStorage.getItem(KEY) ?? '');
		if (s && typeof s === 'object')
			return {
				anyColor: !!s.anyColor,
				byTheme: !!s.byTheme,
				filter: { ...fresh, ...(s.filter ?? {}) },
				sortOut: s.sortOut ?? {}
			};
	} catch {
		// Nothing saved, or storage is off.
	}
	return { anyColor: false, byTheme: false, filter: fresh, sortOut: {} };
}

class ViewState {
	anyColor = $state(false);
	// Sets grouped by theme, in each section and list.
	byTheme = $state(false);
	filter = $state<Filter>(fresh);
	// Each stack's queue, by the stack as the route has it.
	sortOut = $state<Record<string, string[]>>({});

	constructor() {
		const s = load();
		this.anyColor = s.anyColor;
		this.byTheme = s.byTheme;
		this.filter = s.filter;
		this.sortOut = s.sortOut;
	}

	private save() {
		try {
			localStorage.setItem(
				KEY,
				JSON.stringify({
					anyColor: this.anyColor,
					byTheme: this.byTheme,
					filter: this.filter,
					sortOut: this.sortOut
				})
			);
		} catch {
			// Storage can be off; the view lasts the visit.
		}
	}

	setAnyColor(anyColor: boolean) {
		this.anyColor = anyColor;
		this.save();
	}

	setByTheme(byTheme: boolean) {
		this.byTheme = byTheme;
		this.save();
	}

	setFilter(change: Partial<Filter>) {
		this.filter = { ...this.filter, ...change };
		this.save();
	}

	// The preset the filter's rules are, if any.
	preset(): Preset | undefined {
		const f = this.filter;
		return (Object.keys(presets) as Preset[]).find((p) =>
			(Object.keys(presets[p]) as (keyof Rules)[]).every((k) => presets[p][k] === f[k])
		);
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
			filter: { ...this.filter } as MessageInitShape<typeof FilterSchema>,
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
