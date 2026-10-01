<!--
	One set in a ranking: its picture, name and number, what is left out of its
	count, and how complete the pile makes it. The whole row opens the set.
-->
<script lang="ts">
	import type { SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import PartImage from '$lib/components/PartImage.svelte';
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import { count, percent } from '$lib/format';

	// numbered: show its place in the explained order.
	let { set, numbered = false }: { set: SetMatch; numbered?: boolean } = $props();
	const share = $derived(set.have / Math.max(1, set.need));
</script>

<a
	href="/sets/{set.setNum}"
	class="flex flex-wrap items-center gap-x-4 gap-y-2 px-(--pad-panel) py-3 hover:bg-hover"
>
	{#if numbered}<span class="num w-6 shrink-0 text-sm text-ink-faint">{set.pick}</span>{/if}
	<PartImage src={set.imageUrl} class="size-16 shrink-0" />
	<div class="min-w-0 flex-1 basis-56">
		<div class="truncate text-sm font-medium text-ink" title={set.name}>{set.name}</div>
		<div class="truncate text-sm text-ink-muted">
			<span class="num">{set.setNum}</span> · {set.year} · {set.theme}
		</div>
		{#if set.printedParts || set.minifigures || set.sameContents.length}
			<div class="mt-1 flex flex-wrap gap-1.5">
				{#if set.printedParts}<Badge>{set.printedParts} printed not counted</Badge>{/if}
				{#if set.minifigures}<Badge
						>{set.minifigures} minifigure{set.minifigures === 1 ? '' : 's'}</Badge
					>{/if}
				{#if set.sameContents.length}<Badge tone="info"
						>also sold as {set.sameContents.join(', ')}</Badge
					>{/if}
			</div>
		{/if}
	</div>
	<div class="flex w-full shrink-0 flex-col gap-1.5 sm:w-56">
		<div class="flex items-baseline justify-between gap-2 text-sm">
			<span class="num font-medium text-ink">{percent(share)}</span>
			<span class="num text-ink-muted">{count(set.have)} / {count(set.need)}</span>
		</div>
		<ProgressBar value={set.have} max={set.need} label="{set.name}: {percent(share)} complete" />
		<div class="num text-xs text-ink-faint">
			{#if set.have === set.need}
				complete
			{:else}
				{count(set.need - set.have)} missing · {percent(set.weightedCompleteness)} of its rarer pieces
			{/if}
		</div>
	</div>
</a>
