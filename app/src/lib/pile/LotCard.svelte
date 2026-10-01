<!--
	A lot in the collection: its name and where it came from, its size, when
	it was sorted, and its colors as a bar. The whole card opens the lot.
-->
<script lang="ts">
	import type { Lot } from '$lib/gen/pile/v1/pile_pb';
	import { count, dayOf } from '$lib/format';
	import ColorBar from './ColorBar.svelte';

	let { lot }: { lot: Lot } = $props();
</script>

<a
	href="/lots/{lot.id}"
	class="flex flex-col gap-4 rounded-panel bg-surface p-(--pad-panel) transition-colors hover:bg-raised"
>
	<div>
		<div class="text-lg font-medium text-ink">{lot.name}</div>
		<p class="text-sm text-ink-muted">{lot.description}</p>
	</div>
	<div class="num flex flex-wrap gap-x-6 gap-y-1 text-sm">
		<span><span class="font-medium text-ink">{count(lot.pieces)}</span> <span class="text-ink-muted">pieces</span></span>
		<span><span class="font-medium text-ink">{count(lot.partColors)}</span> <span class="text-ink-muted">parts in a color</span></span>
		<span class="text-ink-muted"
			>sorted {dayOf(lot.firstSeenUnix)} to {dayOf(lot.lastSeenUnix)}{lot.machineName
				? ` on ${lot.machineName}`
				: ''}</span
		>
	</div>
	<ColorBar colors={lot.colors} height="h-2" />
</a>
