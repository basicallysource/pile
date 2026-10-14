<!--
	One line of a set in its panel, small, so many show at once: the part's
	picture and how many of it were found of how many it needs. Its fill says
	which: all found, some, or none. Its name, color, number and where the
	pieces are say themselves on hover, and the picture opens up close.
	The count opens the line's colors (docs/views.md#colors): its own, any
	color, or another color of the part the lots have (with how many), which
	the line then takes instead; a line that takes another color shows it
	with a dot of that color, one that takes any with a ring.
-->
<script lang="ts">
	import type { SetLine } from '$lib/gen/pile/v1/pile_pb';
	import PartImage from '$lib/components/PartImage.svelte';
	import Menu from '$lib/components/Menu.svelte';
	import { count } from '$lib/format';
	import { view } from '$lib/view.svelte';
	import { zoomsTo } from './zoom.svelte';

	let {
		line,
		names,
		setNum
	}: {
		line: SetLine;
		// The lots' names, when there are several to tell apart.
		names?: Record<string, string>;
		// The set it is a line of, for the view's color choices.
		setNum: string;
	} = $props();

	const fill = $derived(
		line.found >= line.need ? 'bg-success-soft' : line.found > 0 ? 'bg-warning-soft' : 'bg-danger-soft'
	);
	const caption = $derived(`${line.name}${line.color ? `, ${line.color.name}` : ''}, ${line.partNum}`);
	const title = $derived(
		[
			caption,
			`${line.found} of ${line.need} found`,
			line.useColor && `taking ${line.useColor.name} instead`,
			line.anyColor && 'taking any color',
			line.wrongColor && `${line.wrongColor} in another color`,
			names && line.lots.length && line.lots.map((c) => `${c.pieces} in ${names[c.lotId] ?? c.lotId}`).join(', ')
		]
			.filter(Boolean)
			.join(' · ')
	);
	const own = $derived(line.color?.id ?? 0);
	const items = $derived([
		{
			label: `${line.color?.name ?? 'Its color'}, its own`,
			checked: !line.useColor && !line.anyColor,
			onselect: () => view.choose(setNum, own, line.partNum, null)
		},
		{ label: 'Any color', checked: line.anyColor, onselect: () => view.choose(setNum, own, line.partNum, { any: true }) },
		...(line.partColors.length
			? [
					'separator' as const,
					{
						group: 'In the lots',
						items: line.partColors.map((c) => ({
							label: c.color!.name,
							hint: count(c.pieces),
							checked: c.color!.id === own ? false : line.useColor?.id === c.color!.id,
							disabled: c.color!.id === own,
							onselect: () => view.choose(setNum, own, line.partNum, { use: c.color!.id })
						}))
					}
				]
			: [])
	]);
</script>

<div class="flex min-w-0 flex-col items-center gap-0.5 rounded-control p-1 {fill}" {title}>
	<PartImage src={line.imageUrl} onzoom={zoomsTo(caption)} class="size-11" />
	<Menu label="{line.name}: its color" {items} placement="bottom-start" width="15rem">
		{#snippet trigger(props)}
			<button
				{...props}
				type="button"
				class="num flex items-center gap-1 rounded-item px-1 text-xs leading-none text-ink hover:bg-hover"
			>
				{#if line.useColor}
					<span class="size-2 shrink-0 rounded-full ring-1 ring-line" style="background: #{line.useColor.rgb}"></span>
				{:else if line.anyColor}
					<span class="size-2 shrink-0 rounded-full ring-1 ring-ink-muted"></span>
				{/if}
				{line.found}/{line.need}
			</button>
		{/snippet}
	</Menu>
</div>
