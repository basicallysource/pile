// Keeping the reader's place (docs/loading-data.md#keeping-the-place). When what
// is on screen changes under the reader (a newer answer lands, a sheet opens
// or closes beside the page), the page must not move: what they were looking
// at stays where it was on the screen.
//
// Before the change, each scrolling area notes a place in it: the smallest
// place under the pointer, or when the pointer is elsewhere the one at the
// top of the area, with the places nearest it to fall back on, and how far
// each is from the area's top. Once the change is drawn, the area scrolls so
// the first of them still on the page is back where it was.
//
// When the change leaves an area too short to scroll back to the place
// (what was below it is gone), the area gets room at its end, which gives
// way as the reader scrolls back up, and goes when the page changes.
//
// A place is an element marked `data-place`, and each element directly
// inside one marked `data-place="blocks"` (a Loaded region: its panels and
// grids). A place stays put while its insides change; it is never an item of
// a list that can reorder, since the browser keeps such an element across a
// reorder and following it would carry the page away.
import { tick } from 'svelte';

let pointer: { x: number; y: number } | null = null;
if (typeof window !== 'undefined') {
	const at = (event: PointerEvent) => (pointer = { x: event.clientX, y: event.clientY });
	addEventListener('pointermove', at, { capture: true, passive: true });
	addEventListener('pointerdown', at, { capture: true, passive: true });
	document.documentElement.addEventListener('pointerleave', () => (pointer = null));
}

type Area = { scroller: Element; top: number; bottom: number };
type Mark = { el: Element; top: number };

function scrollerOf(el: Element): Element {
	for (let p = el.parentElement; p; p = p.parentElement) {
		const y = getComputedStyle(p).overflowY;
		if ((y === 'auto' || y === 'scroll') && p.scrollHeight > p.clientHeight) return p;
	}
	return document.scrollingElement ?? document.documentElement;
}

function areaOf(scroller: Element): Area {
	if (scroller === (document.scrollingElement ?? document.documentElement))
		return { scroller, top: 0, bottom: innerHeight };
	const r = scroller.getBoundingClientRect();
	return { scroller, top: r.top, bottom: r.bottom };
}

function places(): Element[] {
	const out = new Set<Element>();
	for (const el of document.querySelectorAll('[data-place]')) {
		out.add(el);
		if (el.getAttribute('data-place') === 'blocks') for (const child of el.children) out.add(child);
	}
	return [...out];
}

// The area's places on screen, best first: the one given, if on screen; the
// smallest across the focus line; the rest by how near they are to it; then
// the larger ones across it.
function rank(area: Area, els: Element[], at: Element | undefined): Mark[] {
	const focus =
		pointer &&
		pointer.y >= area.top &&
		pointer.y < area.bottom &&
		scrollerOf(document.elementFromPoint(pointer.x, pointer.y) ?? document.body) === area.scroller
			? pointer.y
			: area.top;
	const across: { el: Element; top: number; size: number }[] = [];
	const near: { el: Element; top: number; distance: number }[] = [];
	for (const el of els) {
		const r = el.getBoundingClientRect();
		if (r.height === 0 || r.bottom <= area.top || r.top >= area.bottom) continue;
		const top = r.top - area.top;
		if (r.top <= focus && r.bottom > focus) across.push({ el, top, size: r.height });
		else near.push({ el, top, distance: Math.min(Math.abs(r.top - focus), Math.abs(r.bottom - focus)) });
	}
	across.sort((a, b) => a.size - b.size);
	near.sort((a, b) => a.distance - b.distance);
	const marks = [...across.slice(0, 1), ...near, ...across.slice(1)].map(({ el, top }) => ({ el, top }));
	if (at && at.isConnected) {
		const r = at.getBoundingClientRect();
		if (r.height > 0 && r.bottom > area.top && r.top < area.bottom) marks.unshift({ el: at, top: r.top - area.top });
	}
	return marks;
}

export type Held = { restore: () => void };

// Notes the place in every scrolling area, to restore once a change is
// drawn. `at` is held first in its area when it is on screen: the item
// clicked, when the change is about it and it keeps its spot in the DOM.
export function holdPlace(at?: Element): Held {
	const byArea = new Map<Element, Element[]>();
	const candidates = places();
	if (at) candidates.push(at);
	for (const el of candidates) {
		const s = scrollerOf(el);
		const list = byArea.get(s) ?? [];
		list.push(el);
		byArea.set(s, list);
	}
	const held = [...byArea].map(([scroller, els]) => {
		const area = areaOf(scroller);
		return { scroller, marks: rank(area, els.filter((e) => e !== at), at && els.includes(at) ? at : undefined) };
	});
	return {
		restore() {
			for (const { scroller, marks } of held) {
				const mark = marks.find((m) => m.el.isConnected);
				if (!mark) continue;
				const top = () => mark.el.getBoundingClientRect().top - areaOf(scroller).top;
				if (Math.abs(top() - mark.top) >= 1) scroller.scrollTop += top() - mark.top;
				const short = top() - mark.top;
				if (short >= 1) {
					makeRoom(scroller, short);
					scroller.scrollTop += short;
				}
			}
		}
	};
}

// Room at the end of a scrolling area, by the area.
const rooms = new Map<Element, { spacer: HTMLElement; off: () => void }>();

function makeRoom(scroller: Element, px: number) {
	let room = rooms.get(scroller);
	if (!room) {
		const root = document.scrollingElement ?? document.documentElement;
		const spacer = document.createElement('div');
		spacer.setAttribute('aria-hidden', 'true');
		spacer.style.height = '0px';
		(scroller === root ? document.body : scroller).append(spacer);
		const target: EventTarget = scroller === root ? window : scroller;
		let last = scroller.scrollTop;
		const onscroll = () => {
			const up = last - scroller.scrollTop;
			last = scroller.scrollTop;
			if (up <= 0) return;
			const h = parseFloat(spacer.style.height) - up;
			if (h > 0) spacer.style.height = `${h}px`;
			else off();
		};
		const off = () => {
			target.removeEventListener('scroll', onscroll);
			spacer.remove();
			rooms.delete(scroller);
		};
		target.addEventListener('scroll', onscroll, { passive: true });
		room = { spacer, off };
		rooms.set(scroller, room);
	}
	room.spacer.style.height = `${parseFloat(room.spacer.style.height) + px}px`;
}

// Takes away the room made at the end of every area: another page.
export function clearRoom() {
	for (const room of [...rooms.values()]) room.off();
}

let batch: Held | null = null;

// Makes a change (new state for the page to draw) without moving the
// reader: the place is held before it and restored once it is drawn.
// Changes made together share one hold.
export function keepPlace(change: () => void) {
	if (batch) return change();
	const held = holdPlace();
	batch = held;
	change();
	tick().then(() => {
		batch = null;
		held.restore();
	});
}

// Coming back to a page (Back, or a reload), the browser scrolls to where it
// was left, but until its answers are in (after a reload, from a slow
// server) the page may be too short to get there. Until it is tall enough,
// or the reader scrolls, it is scrolled there again each time it grows. A
// hold handed over with `returnWith` (closing a sheet with its own place)
// is restored instead.
let resuming: (() => void) | null = null;
let handed: { held: Held; until: number } | null = null;

export function returnWith(held: Held) {
	handed = { held, until: Date.now() + 3000 };
}

export function resumeAt(y: number) {
	resuming?.();
	const h = handed;
	handed = null;
	if (h && Date.now() < h.until) return h.held.restore();
	const root = document.scrollingElement ?? document.documentElement;
	const go = () => {
		root.scrollTop = y;
		if (Math.abs(root.scrollTop - y) < 1) stop();
	};
	const grown = new ResizeObserver(go);
	const events = ['wheel', 'touchstart', 'keydown', 'pointerdown'] as const;
	const timer = setTimeout(() => stop(), 20000);
	function stop() {
		grown.disconnect();
		clearTimeout(timer);
		for (const e of events) removeEventListener(e, stop, true);
		if (resuming === stop) resuming = null;
	}
	resuming = stop;
	for (const e of events) addEventListener(e, stop, { capture: true, passive: true });
	grown.observe(document.body);
	go();
}
