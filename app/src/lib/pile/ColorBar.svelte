<!--
	A lot's colors as one bar, each color's segment as wide as its share of
	the pieces (the rest grouped as "other"), drawn in the color itself.
-->
<script lang="ts">
	import type { ColorShare } from '$lib/gen/pile/v1/pile_pb';
	import { swatchColor } from '$lib/components/ColorChip.svelte';

	let {
		colors,
		shown = 24,
		height = 'h-3'
	}: { colors: ColorShare[]; shown?: number; height?: string } = $props();

	const total = $derived(colors.reduce((n, c) => n + c.pieces, 0));
	const rest = $derived(colors.slice(shown).reduce((n, c) => n + c.pieces, 0));
</script>

<div
	class="flex w-full overflow-hidden rounded-badge ring-1 ring-line {height}"
	role="img"
	aria-label="Colors by share of pieces"
>
	{#each colors.slice(0, shown) as c (c.color?.id)}
		<div
			class="h-full"
			style="width: {(c.pieces / Math.max(1, total)) * 100}%; background: {swatchColor(c.color?.rgb) ??
				'var(--track)'}"
			title="{c.color?.name}: {c.pieces}"
		></div>
	{/each}
	{#if rest}
		<div class="h-full bg-track" style="width: {(rest / Math.max(1, total)) * 100}%" title="Other colors: {rest}"></div>
	{/if}
</div>
