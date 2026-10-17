// The sets' changes: how each will be built where it differs from its
// inventory (a line's colors, other parts that make one of its part, whether
// a line counts), kept on the server with the sets (changes.json) so they
// last, show on every device and go into what is made from a set later
// (docs/changes.md). A change shows here at once, for the menus' checks;
// once the server has kept it, `revision` counts it, and every answer
// matched through the sets (they all may be) is asked for again. If keeping
// it fails, it changes back.
import { create, type MessageInitShape } from '@bufbuild/protobuf';
import { pile } from '$lib/client';
import { ChangeSchema, type Change } from '$lib/gen/pile/v1/pile_pb';

export type Kind = NonNullable<Change['what']['case']>;
export type ChangeInit = MessageInitShape<typeof ChangeSchema>;

// Whether two changes are for the same line and change the same thing.
const same = (a: Change, partNum: string, color: number, kind: Kind | undefined) =>
	a.partNum === partNum && a.color === color && a.what.case === kind;

class Changes {
	#bySet = $state.raw<Record<string, Change[]>>({});
	#loading: Promise<void> | undefined;
	revision = $state(0);

	#load() {
		this.#loading ??= pile.listChanges({}).then(
			(r) => {
				this.#bySet = Object.fromEntries(r.sets.map((s) => [s.setNum, s.changes]));
			},
			() => (this.#loading = undefined)
		);
	}

	// A set's changes, in the order they were made.
	of(setNum: string): Change[] {
		this.#load();
		return this.#bySet[setNum] ?? [];
	}

	// How many sets have a change.
	sets(): number {
		this.#load();
		return Object.keys(this.#bySet).length;
	}

	// A set's change of one kind for a line (partNum empty: for every line in
	// the color).
	find(setNum: string, partNum: string, color: number, kind: Kind): Change | undefined {
		return this.of(setNum).find((c) => same(c, partNum, color, kind));
	}

	// Makes a change, replacing the set's change of the same line and kind.
	make(setNum: string, init: ChangeInit) {
		const c = create(ChangeSchema, init);
		const cs = this.of(setNum);
		const at = cs.findIndex((x) => same(x, c.partNum, c.color, c.what.case));
		this.#save(setNum, at < 0 ? [...cs, c] : cs.map((x, i) => (i === at ? c : x)), {
			setNum,
			change: c
		});
	}

	// The colors a line takes (partNum empty: every line in the color):
	// some, any, or (none, or only its own) its own. A part's own color,
	// while its color is changed, is a change of its own.
	recolor(setNum: string, partNum: string, color: number, colors: number[], anyColor = false) {
		const mine = this.find(setNum, partNum, color, 'colors');
		if (!anyColor && (colors.length === 0 || (colors.length === 1 && colors[0] === color))) {
			if (partNum && this.find(setNum, '', color, 'colors'))
				this.make(setNum, { partNum, color, what: { case: 'colors', value: { colors: [] } } });
			else if (mine) this.undo(setNum, mine);
			return;
		}
		this.make(setNum, {
			partNum,
			color,
			what: { case: 'colors', value: { colors, any: anyColor } }
		});
	}

	// Takes back the set's change of the same line and kind.
	undo(setNum: string, c: Change) {
		const cs = this.of(setNum).filter((x) => !same(x, c.partNum, c.color, c.what.case));
		this.#save(setNum, cs, { setNum, change: c, undo: true });
	}

	#save(setNum: string, next: Change[], req: { setNum: string; change: Change; undo?: boolean }) {
		const before = this.#bySet;
		this.#bySet = { ...before, [setNum]: next };
		pile.setChange(req).then(
			(r) => {
				const kept = r.set?.changes ?? [];
				const rest = Object.entries(this.#bySet).filter(([n]) => n !== setNum);
				this.#bySet = Object.fromEntries(kept.length ? [...rest, [setNum, kept]] : rest);
				this.revision++;
			},
			() => (this.#bySet = before)
		);
	}
}

export const changes = new Changes();
