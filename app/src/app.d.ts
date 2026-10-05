// See https://svelte.dev/docs/kit/types#app.d.ts
declare global {
	namespace App {
		interface PageState {
			// The open set's sheet was opened from the page under it, so
			// closing it goes back rather than adding a history entry.
			sheet?: boolean;
			// The picture open in the Lightbox, and what it is.
			zoom?: { src: string; alt: string };
		}
	}
}

export {};
