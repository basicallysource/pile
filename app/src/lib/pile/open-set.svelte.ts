// The set open in a lot's sheet, kept in the URL (`?set=10252-1`) so a
// reload or a link opens it again and Back closes it. Opening the first set
// adds a history entry; opening another while one is open replaces it, so
// one Back always returns to the page under it. The page narrows beside the
// sheet and widens again when it closes; either way the reader's place
// holds (docs/loading-data.md#keeping-the-place): the item clicked, or the open
// one, stays where it was on the screen.
import { tick } from 'svelte';
import { goto } from '$app/navigation';
import { page } from '$app/state';
import { holdPlace, returnWith } from '$lib/data/place';

const PARAM = 'set';

// The open set's number, or null.
export function openSet(): string | null {
	return page.url.searchParams.get(PARAM);
}

function withSet(num: string | null): string {
	const url = new URL(page.url);
	if (num) url.searchParams.set(PARAM, num);
	else url.searchParams.delete(PARAM);
	return url.pathname + url.search + url.hash;
}

// The link to a set: this page with the set open over it.
export function setHref(num: string): string {
	return withSet(num);
}

// A link's click handler that opens the set over this page. A click with a
// modifier key (a new tab, a new window) is left to the browser.
export function opensSet(num: string) {
	return (event: MouseEvent) => {
		if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
		event.preventDefault();
		if (openSet() === num) return;
		const opened = openSet() !== null;
		const place = holdPlace(event.currentTarget instanceof Element ? event.currentTarget : undefined);
		goto(withSet(num), {
			noScroll: true,
			keepFocus: true,
			replaceState: opened,
			state: { sheet: opened ? !!page.state.sheet : true }
		})
			.then(tick)
			.then(place.restore);
	};
}

export function closeSet() {
	// The open set's tile on the page, if it is on screen.
	const tile = document.querySelector('main [aria-current="true"]') ?? undefined;
	const place = holdPlace(tile);
	if (page.state.sheet) {
		// Back restores the page's own scroll; this place is restored instead.
		returnWith(place);
		history.back();
	} else
		goto(withSet(null), { noScroll: true, keepFocus: true, replaceState: true })
			.then(tick)
			.then(place.restore);
}
