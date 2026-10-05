// The picture seen up close, in the Lightbox over every page. It is a
// history entry of its own (shallow, the page under it stays), so Back
// closes it as well as Escape and the close button.
import { pushState } from '$app/navigation';
import { page } from '$app/state';

export function zoomed(): { src: string; alt: string } | null {
	return page.state.zoom ?? null;
}

// A handler for PartImage's `onzoom`: what the picture is, said under it.
export function zoomsTo(alt: string) {
	return (src: string) => pushState('', { ...page.state, zoom: { src, alt } });
}

export function unzoom() {
	if (page.state.zoom) history.back();
}
