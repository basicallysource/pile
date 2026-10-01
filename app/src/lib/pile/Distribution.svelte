<!--
	What a lot's pieces are: its part categories as bars, most first, and its
	colors as one bar with the leading colors named under it.
-->
<script lang="ts">
	import type { CategoryShare, ColorShare } from '$lib/gen/pile/v1/pile_pb';
	import Panel from '$lib/components/Panel.svelte';
	import ColorChip from '$lib/components/ColorChip.svelte';
	import { count, percent } from '$lib/format';
	import ColorBar from './ColorBar.svelte';

	let {
		categories,
		colors,
		pieces
	}: { categories: CategoryShare[]; colors: ColorShare[]; pieces: number } = $props();

	const top = $derived(categories.slice(0, 10));
	const most = $derived(Math.max(1, ...top.map((c) => c.pieces)));
</script>

<Panel title="What's in it" description="The pieces by kind of part, and by color.">
	<div class="grid gap-8 lg:grid-cols-2">
		<div class="flex flex-col gap-1.5">
			{#each top as c (c.name)}
				<div class="grid grid-cols-[9rem_1fr_4rem] items-center gap-3 text-sm">
					<span class="truncate text-ink" title={c.name}>{c.name}</span>
					<div class="h-2 overflow-hidden rounded-badge bg-track">
						<div class="h-full bg-primary" style="width: {(c.pieces / most) * 100}%"></div>
					</div>
					<span class="num text-right text-ink-muted">{count(c.pieces)}</span>
				</div>
			{/each}
			{#if categories.length > top.length}
				<p class="text-xs text-ink-faint">
					and {categories.length - top.length} more kinds of part
				</p>
			{/if}
		</div>
		<div class="flex flex-col gap-3">
			<ColorBar {colors} />
			<div class="grid grid-cols-2 gap-x-4 gap-y-1 sm:grid-cols-3">
				{#each colors.slice(0, 15) as c (c.color?.id)}
					<div class="flex min-w-0 items-center justify-between gap-2 text-sm">
						<span class="min-w-0 truncate text-ink"
							><ColorChip name={c.color?.name ?? ''} rgb={c.color?.rgb} /></span
						>
						<span class="num shrink-0 text-ink-muted">{percent(c.pieces / Math.max(1, pieces))}</span>
					</div>
				{/each}
			</div>
			<p class="text-xs text-ink-faint">{colors.length} colors in all</p>
		</div>
	</div>
</Panel>
