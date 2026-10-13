<!--
	One line of a set in its panel, small, so many show at once: the part's
	picture and how many of it were found of how many it needs. Its fill says
	which: all found, some, or none; nothing else marks it. Its name, color,
	number and where the pieces are say themselves on hover, and the picture
	opens up close.
-->
<script lang="ts">
	import type { SetLine } from '$lib/gen/pile/v1/pile_pb';
	import PartImage from '$lib/components/PartImage.svelte';
	import { zoomsTo } from './zoom.svelte';

	let {
		line,
		names
	}: {
		line: SetLine;
		// The lots' names, when there are several to tell apart.
		names?: Record<string, string>;
	} = $props();

	const fill = $derived(
		line.found >= line.need ? 'bg-success-soft' : line.found > 0 ? 'bg-warning-soft' : 'bg-danger-soft'
	);
	const caption = $derived(`${line.name}${line.color ? `, ${line.color.name}` : ''}, ${line.partNum}`);
	const title = $derived(
		[
			caption,
			`${line.found} of ${line.need} found`,
			line.wrongColor && `${line.wrongColor} in another color`,
			names && line.lots.length && line.lots.map((c) => `${c.pieces} in ${names[c.lotId] ?? c.lotId}`).join(', ')
		]
			.filter(Boolean)
			.join(' · ')
	);
</script>

<div class="flex min-w-0 flex-col items-center gap-0.5 rounded-control p-1 {fill}" {title}>
	<PartImage src={line.imageUrl} onzoom={zoomsTo(caption)} class="size-11" />
	<span class="num text-xs leading-none text-ink">{line.found}/{line.need}</span>
</div>
