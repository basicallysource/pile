<!--
	A dense grid of SetTiles, as many columns as fit: three on a phone, up to
	ten on a wide screen. `small` packs smaller tiles, for what is secondary
	(the minifigure sets). Grouped by theme (the view's choice), each theme's
	sets follow its name and count, themes in name order, and each keeps the
	order it was given.
-->
<script lang="ts">
	import type { SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import { view } from '$lib/view.svelte';
	import SetTile from './SetTile.svelte';

	let {
		sets,
		numbered = false,
		small = false
	}: { sets: SetMatch[]; numbered?: boolean; small?: boolean } = $props();

	const groups = $derived.by(() => {
		if (!view.byTheme) return [{ theme: '', sets }];
		const by = new Map<string, SetMatch[]>();
		for (const s of sets) {
			const t = s.themeGroup || 'No theme';
			by.set(t, [...(by.get(t) ?? []), s]);
		}
		return [...by]
			.sort(([a], [b]) => a.localeCompare(b))
			.map(([theme, sets]) => ({ theme, sets }));
	});
</script>

<div
	class="grid gap-1 p-2 {small
		? 'grid-cols-[repeat(auto-fill,minmax(5.5rem,1fr))]'
		: 'grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))]'}"
>
	{#each groups as g (g.theme)}
		{#if g.theme}
			<p class="col-span-full px-1.5 pt-2 text-xs font-medium text-ink-muted">
				{g.theme} <span class="num text-ink-faint">· {g.sets.length}</span>
			</p>
		{/if}
		{#each g.sets as set (set.setNum)}<SetTile {set} {numbered} />{/each}
	{/each}
</div>
