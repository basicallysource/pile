// How every page loads what it shows (docs/loading-data.md). A Source is one of
// the server's calls with the answers this visit already has; a Query is a
// page asking it, for a request that changes as the page does.
//
// - An answer already had draws at once, so a page come back to lands where
//   it was scrolled.
// - A new request keeps the last answer on screen until its own is in: an
//   action never blanks the page. After a moment it is marked as updating.
// - A newer answer replaces what is shown in place, keeping the reader's
//   place (place.ts).
// - A request about something else (another lot, another set) does not show
//   the last one's answer: it waits, in the shape of what is coming.
import { untrack } from 'svelte';
import { SvelteSet } from 'svelte/reactivity';
import { keepPlace } from './place';

// How long an answer may take before what it replaces is marked as updating.
const MOMENT = 300;
// How often an answer the server is still working out is asked for again.
const POLL = 4000;

const keyOf = (v: unknown) => JSON.stringify(v, (_, x) => (typeof x === 'bigint' ? String(x) : x));

export type SourceOptions<Res> = {
	// What the top bar says while an answer takes more than a moment, as
	// "Matching the sets".
	doing: string;
	// Whether an answer is final. One that is not (the server works it out a
	// part at a time) is asked for again every few seconds while shown.
	settled?: (res: Res) => boolean;
	// Answers that can change while pile runs (what the bins hold): one had
	// is shown at once, and asked for again each time.
	changes?: boolean;
};

export class Source<Req, Res> {
	#answers = new Map<string, Res>();
	#asking = new Map<string, Promise<Res>>();
	#call: (req: Req) => Promise<Res>;
	options: SourceOptions<Res>;

	constructor(call: (req: Req) => Promise<Res>, options: SourceOptions<Res>) {
		this.#call = call;
		this.options = options;
	}

	// The answer this visit has for the request, if any.
	peek(req: Req): Res | undefined {
		return this.#answers.get(keyOf(req));
	}

	// Whether an answer had is good as it is, with nothing to ask again.
	final(res: Res): boolean {
		return !this.options.changes && (this.options.settled?.(res) ?? true);
	}

	// Asks the server; one question at a time for the same request.
	ask(req: Req): Promise<Res> {
		const key = keyOf(req);
		let asking = this.#asking.get(key);
		if (!asking) {
			asking = this.#call(req)
				.then((res) => {
					this.#answers.set(key, res);
					return res;
				})
				.finally(() => this.#asking.delete(key));
			this.#asking.set(key, asking);
		}
		return asking;
	}
}

// The queries whose answers are taking more than a moment, for the top bar.
const slow = new SvelteSet<Query<unknown, unknown>>();

// What is updating now, in words, or nothing.
export function updating(): string | undefined {
	for (const q of slow) return q.source.options.doing;
}

export class Query<Req, Res> {
	// What is shown: the latest answer in, for this subject.
	data = $state.raw<Res>();
	error = $state<string>();
	// An answer is on its way to replace what is shown.
	asking = $state(false);
	// And it has taken more than a moment.
	slow = $state(false);
	source: Source<Req, Res>;
	#subject: string | undefined;

	constructor(
		source: Source<Req, Res>,
		request: () => Req | undefined,
		about: (req: Req) => unknown = () => null
	) {
		this.source = source;
		const first = untrack(request);
		if (first !== undefined) {
			this.data = source.peek(first);
			this.#subject = keyOf(about(first));
		}
		$effect(() => {
			const req = request();
			return untrack(() => this.#follow(req, about));
		});
		$effect(() => () => {
			slow.delete(this as Query<unknown, unknown>);
		});
	}

	#mark(on: boolean) {
		this.slow = on;
		if (on) slow.add(this as Query<unknown, unknown>);
		else slow.delete(this as Query<unknown, unknown>);
	}

	#show(res: Res) {
		if (res === this.data && this.error === undefined) return;
		if (this.data === undefined) {
			this.data = res;
			this.error = undefined;
		} else
			keepPlace(() => {
				this.data = res;
				this.error = undefined;
			});
	}

	#fail(e: unknown) {
		const message = e instanceof Error ? e.message : String(e);
		if (this.data === undefined) this.error = message;
		else keepPlace(() => (this.error = message));
	}

	#follow(req: Req | undefined, about: (req: Req) => unknown) {
		if (req === undefined) {
			this.#subject = undefined;
			this.data = undefined;
			this.error = undefined;
			this.asking = false;
			this.#mark(false);
			return;
		}
		const had = this.source.peek(req);
		const subject = keyOf(about(req));
		if (subject !== this.#subject) {
			// Something else: never the last one's answer.
			this.#subject = subject;
			this.data = had;
			this.error = undefined;
		} else if (had) this.#show(had);

		if (had && this.source.final(had)) {
			this.asking = false;
			this.#mark(false);
			return;
		}
		// An answer had is shown while it is asked for again, unmarked; one
		// not had yet is waited for, marked after a moment.
		this.asking = !had;
		if (had) this.#mark(false);
		let gone = false;
		let poll: ReturnType<typeof setTimeout> | undefined;
		const moment = had ? undefined : setTimeout(() => this.#mark(true), MOMENT);
		const ask = () =>
			this.source.ask(req).then(
				(res) => {
					if (gone) return;
					clearTimeout(moment);
					this.#show(res);
					this.asking = false;
					this.#mark(false);
					if (!this.source.options.changes && !this.source.final(res)) poll = setTimeout(ask, POLL);
				},
				(e) => {
					if (gone) return;
					clearTimeout(moment);
					this.#fail(e);
					this.asking = false;
					this.#mark(false);
				}
			);
		ask();
		return () => {
			gone = true;
			clearTimeout(moment);
			clearTimeout(poll);
		};
	}
}

// A page's question to a source, asked again whenever `request` changes
// (undefined: nothing to ask). `about` says what the answer is about (a lot,
// a set): a request about something else does not keep the last answer.
// Made while a component starts, which it belongs to.
export function query<Req, Res>(
	source: Source<Req, Res>,
	request: () => Req | undefined,
	about?: (req: Req) => unknown
): Query<Req, Res> {
	return new Query(source, request, about);
}
