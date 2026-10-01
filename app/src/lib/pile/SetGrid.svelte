<!--
	A dense grid of SetTiles, as many columns as fit: three on a phone, up to
	ten on a wide screen. `small` packs smaller tiles, for what is secondary
	(the minifigure sets).
-->
<script lang="ts">
	import type { SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import SetTile from './SetTile.svelte';

	let {
		sets,
		lot,
		numbered = false,
		small = false
	}: { sets: SetMatch[]; lot: string; numbered?: boolean; small?: boolean } = $props();
</script>

<div
	class="grid gap-1 p-2 {small
		? 'grid-cols-[repeat(auto-fill,minmax(5.5rem,1fr))]'
		: 'grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))]'}"
>
	{#each sets as set (set.setNum)}<SetTile {set} {lot} {numbered} />{/each}
</div>
