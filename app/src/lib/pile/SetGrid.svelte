<!--
	A dense grid of SetTiles, as many columns as fit: three on a phone, up to
	ten on a wide screen.
-->
<script lang="ts">
	import type { SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import SetTile from './SetTile.svelte';

	let { sets, numbered = false }: { sets: SetMatch[]; numbered?: boolean } = $props();
</script>

<div class="grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-1 p-2">
	{#each sets as set (set.setNum)}<SetTile {set} {numbered} />{/each}
</div>
