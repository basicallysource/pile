<!--
	The sort-out queue: the sets, custom models and builds within sets to
	pull from the pile, most wanted first, each counted against what the ones
	before it left, as a grid of cards, under a search that adds one
	(AddToQueue). Everything below on the page is matched against what
	is left after all of them. With several lots, each card says how many of
	its pieces come from which lot (the first lot that has them, then the
	next). It is the view's, so it stays as lots are added, taken out or
	moved, and is counted again against them; a set they cannot help keeps
	its place and shows what they have of it.

	A card is dragged into its place: picked up by the pointer anywhere on
	it, it floats under the pointer as the raised plane
	while the others slide aside, leaving the gap where it will land, and it
	settles into that gap when let go (Escape puts it back). Its grip moves it
	with the arrow keys too. The order is the view's at once: the cards stand
	in it from the moment they are let go, and their numbers, which depend on
	the order, fade until the new count is in and then change in place, so
	nothing jumps back. A card just put in the queue holds its place in the
	shape of a card until its numbers come.
-->
<script lang="ts">
	import { flip } from 'svelte/animate';
	import { cubicOut } from 'svelte/easing';
	import GripVertical from '@lucide/svelte/icons/grip-vertical';
	import X from '@lucide/svelte/icons/x';
	import type { QueuedSet } from '$lib/gen/pile/v1/pile_pb';
	import Panel from '$lib/components/Panel.svelte';
	import Button from '$lib/components/Button.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import { count, percent } from '$lib/format';
	import { view } from '$lib/view.svelte';
	import { opensSet, setHref } from './open-set.svelte';
	import AddToQueue from './AddToQueue.svelte';

	let {
		queue,
		names,
		pieces,
		left
	}: {
		// The server's last count of the queue, in the order it was asked in.
		queue: QueuedSet[];
		// The lots' names.
		names: Record<string, string>;
		pieces: number;
		left: number;
	} = $props();

	const several = $derived(Object.keys(names).length > 1);
	const counted = $derived(new Map(queue.map((q) => [q.set!.setNum, q])));
	// Counted in another order than the one shown: the numbers are on their way.
	const recounting = $derived(queue.map((q) => q.set!.setNum).join() !== view.queue.join());

	// A pointer moves this far before a press becomes a drag.
	const PICK_UP = 5;
	const SLIDE_MS = 180;
	const still = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
	const slide = still ? 0 : SLIDE_MS;

	let grid = $state<HTMLElement>();
	const cards = new Map<string, HTMLElement>();
	// While a card is dragged: the order shown, the card, and where it floats.
	let order = $state<string[] | null>(null);
	let held = $state<{ num: string; x: number; y: number; w: number; settling: boolean } | null>(null);
	const shown = $derived(order ?? view.queue);

	function card(node: HTMLElement, num: string) {
		cards.set(num, node);
		return {
			update(next: string) {
				cards.delete(num);
				num = next;
				cards.set(num, node);
			},
			destroy() {
				if (cards.get(num) === node) cards.delete(num);
			}
		};
	}

	// Where a card is laid out, on the screen: from its place in the grid, not
	// where a slide has it now, so the gaps never chase a moving card.
	function slot(num: string): DOMRect | undefined {
		const el = cards.get(num);
		if (!el || !grid) return;
		const g = grid.getBoundingClientRect();
		return new DOMRect(g.left + el.offsetLeft, g.top + el.offsetTop, el.offsetWidth, el.offsetHeight);
	}

	let press: { num: string; id: number; x: number; y: number; dx: number; dy: number } | null = null;
	let dragged = false;

	function down(e: PointerEvent, num: string) {
		// Anywhere on the card, its buttons too: a press that does not move is
		// still their click.
		if (e.button !== 0 || held) return;
		const r = slot(num);
		if (!r) return;
		press = { num, id: e.pointerId, x: e.clientX, y: e.clientY, dx: e.clientX - r.left, dy: e.clientY - r.top };
		dragged = false;
		window.addEventListener('pointermove', move);
		window.addEventListener('pointerup', up);
		window.addEventListener('pointercancel', cancel);
		window.addEventListener('keydown', escape);
	}

	function move(e: PointerEvent) {
		if (!press || e.pointerId !== press.id) return;
		if (!held) {
			if (Math.hypot(e.clientX - press.x, e.clientY - press.y) < PICK_UP) return;
			const r = slot(press.num)!;
			order = [...view.queue];
			held = { num: press.num, x: r.left, y: r.top, w: r.width, settling: false };
			dragged = true;
			document.body.style.userSelect = 'none';
			document.body.style.cursor = 'grabbing';
		}
		held.x = e.clientX - press.dx;
		held.y = e.clientY - press.dy;
		scrollNear(e.clientY);
		// Over another card's place: the held one takes it, and the rest slide.
		for (const num of order!) {
			if (num === held.num) continue;
			const r = slot(num);
			if (r && e.clientX >= r.left && e.clientX <= r.right && e.clientY >= r.top && e.clientY <= r.bottom) {
				const next = order!.filter((n) => n !== held!.num);
				next.splice(order!.indexOf(num), 0, held.num);
				order = next;
				break;
			}
		}
	}

	// Near the top or bottom of the window, the page scrolls under the card.
	let scrolling = 0;
	function scrollNear(y: number) {
		const edge = 56;
		const speed = y < edge ? -(edge - y) / 4 : y > innerHeight - edge ? (y - (innerHeight - edge)) / 4 : 0;
		cancelAnimationFrame(scrolling);
		if (!speed || !held) return;
		const step = () => {
			if (!held || held.settling) return;
			scrollBy(0, speed);
			scrolling = requestAnimationFrame(step);
		};
		scrolling = requestAnimationFrame(step);
	}

	function up(e: PointerEvent) {
		if (!press || e.pointerId !== press.id) return;
		finish(true);
	}

	function cancel() {
		finish(false);
	}

	function escape(e: KeyboardEvent) {
		if (e.key === 'Escape' && held) {
			e.preventDefault();
			finish(false);
		}
	}

	// Let go: the order becomes the view's (or stays as it was), and the card
	// glides into its place before it is drawn there.
	function finish(keep: boolean) {
		window.removeEventListener('pointermove', move);
		window.removeEventListener('pointerup', up);
		window.removeEventListener('pointercancel', cancel);
		window.removeEventListener('keydown', escape);
		cancelAnimationFrame(scrolling);
		press = null;
		document.body.style.userSelect = '';
		document.body.style.cursor = '';
		if (!held) return;
		const h = held;
		if (keep && order) view.reorder(order);
		order = null;
		// The click that ends a drag opens nothing.
		window.addEventListener('click', swallow, { capture: true, once: true });
		setTimeout(() => window.removeEventListener('click', swallow, { capture: true }), 0);
		requestAnimationFrame(() => {
			const r = slot(h.num);
			if (!r || !slide) {
				held = null;
				return;
			}
			held = { ...h, x: r.left, y: r.top, settling: true };
			setTimeout(() => {
				if (held?.settling) held = null;
			}, slide);
		});
	}

	function swallow(e: MouseEvent) {
		if (dragged) {
			e.preventDefault();
			e.stopPropagation();
		}
	}

	// The grip moves its card with the arrow keys.
	function key(e: KeyboardEvent, num: string) {
		const step = { ArrowLeft: -1, ArrowUp: -1, ArrowRight: 1, ArrowDown: 1 }[e.key];
		if (!step) return;
		e.preventDefault();
		view.move(num, step as -1 | 1);
	}
</script>

{#snippet body(num: string, i: number)}
	{@const q = counted.get(num)}
	{@const set = q?.set}
	<div class="flex items-center justify-between">
		<span class="num text-xs font-medium text-primary-ink">{i + 1}</span>
		<div class="flex">
			<Button
				size="sm"
				variant="ghost"
				icon={GripVertical}
				label="Move {set?.name ?? num}: drag the card, or the arrow keys"
				class="cursor-grab"
				onkeydown={(e: KeyboardEvent) => key(e, num)}
			/>
			<Button size="sm" variant="ghost" icon={X} label="Take out of the queue" onclick={() => view.toggle(num)} />
		</div>
	</div>
	{#if set}
		{@const done = set.have >= set.need}
		<a href={setHref(num)} onclick={opensSet(num)} draggable="false" class="flex flex-col gap-1.5">
			<PartImage src={set.imageUrl} class="pointer-events-none aspect-square w-full" />
			<div class="truncate text-xs font-medium text-ink" title={set.name}>{set.name}</div>
		</a>
		<div class="flex flex-col gap-1.5 transition-opacity duration-200 {recounting ? 'opacity-50' : ''}">
			<ProgressBar
				value={set.have}
				max={set.need}
				tone={done ? 'success' : 'primary'}
				label="{set.name}: {percent(set.have / Math.max(1, set.need))} complete"
			/>
			<div class="num text-xs {done ? 'text-success-ink' : 'text-ink-muted'}">
				{set.have}/{set.need}{#if set.wrongColor}<span class="text-warning-ink">
						· {set.wrongColor} recolor</span
					>{/if}
			</div>
			{#if several}
				<ul class="num flex flex-col text-xs text-ink-muted" aria-label="Pulled from">
					{#each q.lots as l (l.lotId)}
						<li class="truncate" title="{l.pieces} from {names[l.lotId]}">
							{l.pieces} from {names[l.lotId] ?? l.lotId}
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	{:else}
		<Skeleton class="aspect-square w-full" />
		<Skeleton class="h-4 w-3/4" />
		<Skeleton class="h-1.5 w-full" />
		<Skeleton class="h-4 w-1/2" />
	{/if}
{/snippet}

<Panel
	title="Sorting out · {view.queue.length}"
	description={view.queue.length
		? `Most wanted first: drag a card to its place. Each takes its pieces from what the ones before it left; everything below is matched against the ${count(left)} of ${count(pieces)} pieces still in the pile.`
		: 'Sets and builds to pull from the pile, most wanted first: add one by its name or number, or with Sort this out on its panel.'}
	flush
>
	{#snippet actions()}
		{#if view.queue.length}
			<Button size="sm" variant="ghost" icon={X} onclick={() => view.clear()}>Clear</Button>
		{/if}
	{/snippet}
	<div class="px-2 pt-2 {shown.length ? '' : 'pb-2'}"><AddToQueue /></div>
	<div
		bind:this={grid}
		aria-busy={recounting}
		role="list"
		class="relative grid grid-cols-[repeat(auto-fill,minmax(9rem,1fr))] gap-2 {shown.length ? 'p-2' : ''}"
	>
		{#each shown as num, i (num)}
			<div
				use:card={num}
				animate:flip={{ duration: slide, easing: cubicOut }}
				onpointerdown={(e) => down(e, num)}
				ondragstart={(e) => e.preventDefault()}
				role="listitem"
				class="flex min-w-0 touch-pan-y flex-col gap-1.5 rounded-control p-2 select-none {held?.num === num
					? 'bg-hover'
					: 'cursor-grab bg-well'}"
			>
				<div class="flex flex-col gap-1.5 {held?.num === num ? 'invisible' : ''}">
					{@render body(num, i)}
				</div>
			</div>
		{/each}
	</div>
</Panel>

{#if held}
	{@const i = shown.indexOf(held.num)}
	<div
		aria-hidden="true"
		inert
		class="pointer-events-none fixed top-0 left-0 z-50 flex flex-col gap-1.5 rounded-control border border-line bg-raised p-2"
		style="width: {held.w}px; transform: translate({held.x}px, {held.y}px); {held.settling
			? `transition: transform ${slide}ms cubic-bezier(0.33, 1, 0.68, 1);`
			: ''}"
	>
		{@render body(held.num, i < 0 ? 0 : i)}
	</div>
{/if}
