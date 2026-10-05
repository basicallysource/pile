<!--
	One set or custom model in a grid: its picture, its name and number (a
	custom model's designer) in small type, and a thin bar for how much of it
	the pieces hold, with the count. Green when complete; in an any-color view
	the pieces in another color are counted after it. The tile opens the set
	in the sheet over the page, and is marked while it is open;
	the button in its corner puts it in (or takes it out of) the lot's
	sort-out queue, and shows its place there; the magnifier in the other
	corner shows its picture up close. `numbered` shows its place in
	the likely order.
-->
<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import Plus from '@lucide/svelte/icons/plus';
	import ZoomIn from '@lucide/svelte/icons/zoom-in';
	import { Kind, type SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import PartImage from '$lib/components/PartImage.svelte';
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import { percent } from '$lib/format';
	import { view } from '$lib/view.svelte';
	import { openSet, opensSet, setHref } from './open-set.svelte';
	import { zoomsTo } from './zoom.svelte';

	let {
		set,
		lot,
		numbered = false
	}: { set: SetMatch; lot: string; numbered?: boolean } = $props();

	const done = $derived(set.have >= set.need);
	const share = $derived(set.have / Math.max(1, set.need));
	const custom = $derived(set.kind === Kind.CUSTOM);
	const place = $derived(view.queue(lot).indexOf(set.setNum) + 1);
	const open = $derived(openSet() === set.setNum);
	const notes = $derived(
		[
			set.wrongColor && `${set.wrongColor} in another color`,
			set.printedParts && `${set.printedParts} printed not counted`,
			set.minifigures && `${set.minifigures} minifigure${set.minifigures === 1 ? '' : 's'}`,
			set.minifigureParts && `${set.minifigureParts} figure parts not counted`,
			set.sameContents.length && `also sold as ${set.sameContents.join(', ')}`
		]
			.filter(Boolean)
			.join(' · ')
	);
</script>

<div class="group relative min-w-0 rounded-control {open ? 'bg-pressed' : 'hover:bg-hover'}">
	<a
		href={setHref(set.setNum)}
		onclick={opensSet(set.setNum)}
		aria-current={open ? 'true' : undefined}
		title="{set.name}, {set.setNum}{custom
			? `, by ${set.designer}`
			: `, ${set.year}, ${set.themeGroup}${set.theme !== set.themeGroup ? ` / ${set.theme}` : ''}`}, {set.distinctParts} different parts{notes
			? ` (${notes})`
			: ''}"
		class="flex min-w-0 flex-col gap-1.5 rounded-control p-1.5 active:bg-pressed"
	>
		<PartImage src={set.imageUrl} class="aspect-square w-full" />
		<div class="min-w-0">
			<div class="truncate text-xs font-medium text-ink">{set.name}</div>
			<div class="num truncate text-xs text-ink-muted">
				{#if numbered && set.pick}<span class="text-ink-faint">{set.pick} · </span>{/if}
				{#if custom}by {set.designer || 'a designer'}{:else}{set.setNum} · {set.year}{/if}
			</div>
		</div>
		<ProgressBar
			value={set.have}
			max={set.need}
			tone={done ? 'success' : 'primary'}
			label="{set.name}: {percent(share)} complete"
		/>
		<div class="num flex min-w-0 items-center gap-1 text-xs {done ? 'text-success-ink' : 'text-ink-muted'}">
			{#if done}<Check size={12} class="shrink-0" />{/if}
			<span class="shrink-0">{set.have}/{set.need}</span>
			{#if !done}<span class="shrink-0 text-ink-faint">· {percent(share)}</span>{/if}
			{#if set.wrongColor}<span class="truncate text-warning-ink">· {set.wrongColor} recolor</span>{/if}
		</div>
	</a>
	{#if set.imageUrl}
		<button
			type="button"
			onclick={() => zoomsTo(`${set.name}, ${set.setNum}`)(set.imageUrl)}
			title="See the picture up close"
			aria-label="See the picture of {set.name} up close"
			class="absolute top-2.5 left-2.5 flex size-7 cursor-zoom-in items-center justify-center rounded-button bg-surface text-ink opacity-0 ring-1 ring-line transition-opacity group-hover:opacity-100 hover:bg-raised focus-visible:opacity-100"
		>
			<ZoomIn size={14} />
		</button>
	{/if}
	<button
		type="button"
		onclick={() => view.toggle(lot, set.setNum)}
		aria-pressed={place > 0}
		title={place ? `Sorting out ${place}: take out of the queue` : 'Sort this out'}
		aria-label={place ? `Take ${set.name} out of the sort-out queue` : `Sort out ${set.name}`}
		class="num absolute top-2.5 right-2.5 flex size-7 items-center justify-center rounded-button text-xs font-medium transition-opacity {place
			? 'bg-primary text-on-primary opacity-100'
			: 'bg-surface text-ink opacity-0 ring-1 ring-line group-hover:opacity-100 hover:bg-raised focus-visible:opacity-100'}"
	>
		{#if place}{place}{:else}<Plus size={14} />{/if}
	</button>
</div>
