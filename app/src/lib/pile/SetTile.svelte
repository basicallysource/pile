<!--
	One set in a grid: its picture, its name and number in small type, and a
	thin bar for how much of it the pile holds, with the count. Green when
	complete. The whole tile opens the set. `numbered` shows its place in the
	likely order. Grids of them use the `set-grid` columns in SetGrid.svelte.
-->
<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import type { SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import PartImage from '$lib/components/PartImage.svelte';
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import { percent } from '$lib/format';

	let { set, numbered = false }: { set: SetMatch; numbered?: boolean } = $props();
	const done = $derived(set.have >= set.need);
	const share = $derived(set.have / Math.max(1, set.need));
	const notes = $derived(
		[
			set.printedParts && `${set.printedParts} printed not counted`,
			set.minifigures && `${set.minifigures} minifigure${set.minifigures === 1 ? '' : 's'}`,
			set.minifigureParts && `${set.minifigureParts} figure parts not counted`,
			set.sameContents.length && `also sold as ${set.sameContents.join(', ')}`
		]
			.filter(Boolean)
			.join(' · ')
	);
</script>

<a
	href="/sets/{set.setNum}"
	title="{set.name}, {set.setNum}, {set.year}, {set.theme}{notes ? ` (${notes})` : ''}"
	class="flex min-w-0 flex-col gap-1.5 rounded-control p-1.5 hover:bg-hover active:bg-pressed"
>
	<PartImage src={set.imageUrl} class="aspect-square w-full" />
	<div class="min-w-0">
		<div class="truncate text-xs font-medium text-ink">{set.name}</div>
		<div class="num truncate text-xs text-ink-muted">
			{#if numbered}<span class="text-ink-faint">{set.pick} · </span>{/if}{set.setNum} · {set.year}
		</div>
	</div>
	<ProgressBar
		value={set.have}
		max={set.need}
		tone={done ? 'success' : 'primary'}
		label="{set.name}: {percent(share)} complete"
	/>
	<div class="num flex items-center gap-1 text-xs {done ? 'text-success-ink' : 'text-ink-muted'}">
		{#if done}<Check size={12} class="shrink-0" />{/if}
		{set.have}/{set.need}
		{#if !done}<span class="text-ink-faint">· {percent(share)}</span>{/if}
	</div>
</a>
