// The set open in a lot's sheet, kept in the URL (`?set=10252-1`) so a
// reload or a link opens it again and Back closes it. Opening the first set
// adds a history entry; opening another while one is open replaces it, so
// one Back always returns to the page under it, scrolled where it was. The
// page narrows beside the sheet, so the item clicked is brought back into
// view when the first one opens.
import { tick } from 'svelte';
import { goto } from '$app/navigation';
import { page } from '$app/state';

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
		const item = event.currentTarget;
		goto(withSet(num), {
			noScroll: true,
			keepFocus: true,
			replaceState: opened,
			state: { sheet: opened ? !!page.state.sheet : true }
		})
			.then(tick)
			.then(() => {
				if (!opened && item instanceof Element) item.scrollIntoView({ block: 'center' });
			});
	};
}

export function closeSet() {
	if (page.state.sheet) history.back();
	else goto(withSet(null), { noScroll: true, keepFocus: true, replaceState: true });
}
