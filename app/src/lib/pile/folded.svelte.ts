// Which of the overview's sections are folded, by title, kept in this
// browser so a reload keeps them so. A section not yet folded or unfolded
// here is as it starts (the tiny sets start folded).
const KEY = 'pile-folded';

function load(): Record<string, boolean> {
	try {
		const v = JSON.parse(localStorage.getItem(KEY) ?? '{}');
		return v && typeof v === 'object' ? v : {};
	} catch {
		return {};
	}
}

class Folded {
	#by = $state<Record<string, boolean>>(load());

	is(title: string, start = false): boolean {
		return this.#by[title] ?? start;
	}

	set(title: string, folded: boolean) {
		this.#by = { ...this.#by, [title]: folded };
		try {
			localStorage.setItem(KEY, JSON.stringify(this.#by));
		} catch {
			// Storage can be off; it lasts the visit.
		}
	}
}

export const folded = new Folded();
