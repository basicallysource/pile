// How lots are being looked at (docs/views.md). The lots are the route's:
// `/lots/big-box,goodwill-box` is a stack of two, in priority order, so a
// link carries them. The rest is kept in the browser, so it lasts across
// pages and reloads: any-color matching, what counts toward a set's
// completeness (docs/views.md#counting), the filter (which sets are listed,
// docs/views.md#the-filter), whether sets are grouped by theme, and the
// sort-out queue (most wanted first), which is the view's whatever lots it
// looks at: adding, taking out or moving a lot keeps it, and the server
// counts it against the new ones. Every page reads it and the server works
// from it.
import { page } from '$app/state';
import { Kind, type View } from './gen/pile/v1/pile_pb';
import type { MessageInitShape } from '@bufbuild/protobuf';
import type { CountingSchema, FilterSchema, ViewSchema } from './gen/pile/v1/pile_pb';

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

// Which kinds of part count toward a set's completeness, a plain part
// always; the rest are still found, and sorted out with their set.
export type Counting = { printed: boolean; figureParts: boolean; accessories: boolean };
export const defaultCounting: Counting = { printed: false, figureParts: false, accessories: true };

type Saved = {
	anyColor: boolean;
	byTheme: boolean;
	counting: Counting;
	filter: Filter;
	queue: string[];
};

// The queue as it was kept before it was the view's: one per stack of lots.
// The last one kept that has sets in it becomes the view's.
function lastStackQueue(sortOut: unknown): string[] {
	if (!sortOut || typeof sortOut !== 'object') return [];
	const queues = Object.values(sortOut).filter((q) => Array.isArray(q) && q.length);
	return (queues.at(-1) as string[] | undefined) ?? [];
}

function load(): Saved {
	try {
		const s = JSON.parse(localStorage.getItem(KEY) ?? '');
		if (s && typeof s === 'object')
			return {
				anyColor: !!s.anyColor,
				byTheme: !!s.byTheme,
				counting: { ...defaultCounting, ...(s.counting ?? {}) },
				filter: { ...fresh, ...(s.filter ?? {}) },
				queue: Array.isArray(s.queue) ? s.queue : lastStackQueue(s.sortOut)
			};
	} catch {
		// Nothing saved, or storage is off.
	}
	return { anyColor: false, byTheme: false, counting: defaultCounting, filter: fresh, queue: [] };
}

class ViewState {
	anyColor = $state(false);
	// Sets grouped by theme, in each section and list.
	byTheme = $state(false);
	counting = $state<Counting>(defaultCounting);
	filter = $state<Filter>(fresh);
	// Set numbers, most wanted first.
	queue = $state<string[]>([]);

	constructor() {
		const s = load();
		this.anyColor = s.anyColor;
		this.byTheme = s.byTheme;
		this.counting = s.counting;
		this.filter = s.filter;
		this.queue = s.queue;
	}

	private save() {
		try {
			localStorage.setItem(
				KEY,
				JSON.stringify({
					anyColor: this.anyColor,
					byTheme: this.byTheme,
					counting: this.counting,
					filter: this.filter,
					queue: this.queue
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

	setCounting(change: Partial<Counting>) {
		this.counting = { ...this.counting, ...change };
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

	// A set's place in the queue, from 1; 0 when it is not in it.
	placeOf(num: string): number {
		return this.queue.indexOf(num) + 1;
	}

	private setQueue(q: string[]) {
		this.queue = q;
		this.save();
	}

	toggle(num: string) {
		const q = this.queue;
		this.setQueue(q.includes(num) ? q.filter((n) => n !== num) : [...q, num]);
	}

	move(num: string, step: -1 | 1) {
		const q = [...this.queue];
		const i = q.indexOf(num);
		const j = i + step;
		if (i < 0 || j < 0 || j >= q.length) return;
		[q[i], q[j]] = [q[j], q[i]];
		this.setQueue(q);
	}

	clear() {
		this.setQueue([]);
	}

	// The View the server works from, for a stack.
	of(stack: string): MessageInitShape<typeof ViewSchema> {
		return {
			lots: lotsOf(stack),
			anyColor: this.anyColor,
			counting: { ...this.counting } as MessageInitShape<typeof CountingSchema>,
			filter: { ...this.filter } as MessageInitShape<typeof FilterSchema>,
			sortOut: this.queue
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
