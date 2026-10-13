// The starred sets and custom models, kept on the server (stars.json) so they
// last and show on every device. A star changes here at once and is saved
// after; if saving fails, it changes back. `revision` counts the changes, so
// answers that list only starred sets are asked for again
// (docs/views.md#stars).
import { SvelteSet } from 'svelte/reactivity';
import { pile } from '$lib/client';

class Stars {
	#sets = new SvelteSet<string>();
	#loading: Promise<void> | undefined;
	revision = $state(0);

	#load() {
		this.#loading ??= pile.listStars({}).then(
			(r) => this.#take(r.setNums),
			() => (this.#loading = undefined)
		);
	}

	// The server's stars, when they differ from these.
	#take(nums: string[]) {
		if (nums.length === this.#sets.size && nums.every((n) => this.#sets.has(n))) return;
		this.#sets.clear();
		for (const n of nums) this.#sets.add(n);
		this.revision++;
	}

	has(num: string): boolean {
		this.#load();
		return this.#sets.has(num);
	}

	toggle(num: string) {
		const on = !this.#sets.has(num);
		if (on) this.#sets.add(num);
		else this.#sets.delete(num);
		this.revision++;
		pile.setStar({ setNum: num, starred: on }).then(
			(r) => this.#take(r.setNums),
			() => {
				if (on) this.#sets.delete(num);
				else this.#sets.add(num);
				this.revision++;
			}
		);
	}
}

export const stars = new Stars();
