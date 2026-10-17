<!--
	One line of a set in its panel, small, so many show at once: the part's
	picture and how many of it were found of how many it needs. Its fill says
	which: all found, some, or none. Its name, color, number, where the
	pieces are and what stands in say themselves on hover.

	The whole tile is a button: a click, or a right click, opens its menu
	(docs/changes.md), which changes how the set will be built, kept with
	the set:
	- Color: its own, one other color the lots have it in (with how many),
	  several colors to fill it from in any mix (a dialog of checks), or any
	  color.
	- Make it from other parts (a dialog: what could stand in, and a search).
	- Count it toward this set, or leave it out, whatever the view counts.
	- See the picture up close.
	A mark beside the count says which color choice is in force: a dot of
	the color it takes, a dot split into the colors it mixes, a ring for any
	color; and a mark when other parts stand in.
-->
<script lang="ts">
	import Blocks from '@lucide/svelte/icons/blocks';
	import type { ColorTake, SetLine } from '$lib/gen/pile/v1/pile_pb';
	import PartImage from '$lib/components/PartImage.svelte';
	import Menu from '$lib/components/Menu.svelte';
	import { swatchColor } from '$lib/components/ColorChip.svelte';
	import { count } from '$lib/format';
	import { view } from '$lib/view.svelte';
	import { zoomsTo } from './zoom.svelte';
	import { changes } from './changes.svelte';
	import { countsByView } from './counting';

	let {
		line,
		names,
		setNum,
		onmix,
		onswap
	}: {
		line: SetLine;
		// The lots' names, when there are several to tell apart.
		names?: Record<string, string>;
		// The set it is a line of, whose changes it makes.
		setNum: string;
		// Open the dialogs for several colors, and for other parts.
		onmix: (line: SetLine) => void;
		onswap: (line: SetLine) => void;
	} = $props();

	let menu = $state<ReturnType<typeof Menu>>();

	const fill = $derived(
		line.found >= line.need
			? 'bg-success-soft'
			: line.found > 0
				? 'bg-warning-soft'
				: 'bg-danger-soft'
	);
	const caption = $derived(
		`${line.name}${line.color ? `, ${line.color.name}` : ''}, ${line.partNum}`
	);
	const madeFrom = $derived(
		line.madeFrom.map((p) => `${p.count} × ${p.partNum} ${p.color?.name ?? ''}`).join(', ')
	);
	const title = $derived(
		[
			caption,
			`${line.found} of ${line.need} found`,
			line.useColors.length && `taking ${line.useColors.map((c) => c.name).join(', ')}`,
			line.anyColor && 'taking any color',
			line.wrongColor && `${line.wrongColor} in another color`,
			line.made && `${line.made} made from ${madeFrom}`,
			names &&
				line.lots.length &&
				line.lots.map((c) => `${c.pieces} in ${names[c.lotId] ?? c.lotId}`).join(', ')
		]
			.filter(Boolean)
			.join(' · ')
	);

	// The set's changes for the line, from the kept changes so a choice
	// shows at once.
	const own = $derived(line.color?.id ?? 0);
	const forLine = $derived(changes.find(setNum, line.partNum, own, 'colors'));
	const forColor = $derived(changes.find(setNum, '', own, 'colors'));
	const take = $derived((forLine ?? forColor)?.what.value as ColorTake | undefined);
	const several = $derived(!!take && take.colors.length > 1);
	const one = $derived(
		take && take.colors.length === 1 && take.colors[0] !== own ? take.colors[0] : undefined
	);
	const isOwn = $derived(!take || (!take.any && !several && one === undefined));
	const swap = $derived(changes.find(setNum, line.partNum, own, 'swap'));
	const counting = $derived(changes.find(setNum, line.partNum, own, 'counted'));
	const counted = $derived(
		counting?.what.case === 'counted'
			? counting.what.value
			: countsByView(view.counting, line.class)
	);

	const choose = (colors: number[], anyColor = false) =>
		changes.recolor(setNum, line.partNum, own, colors, anyColor);

	function toggleCounted() {
		const next = !counted;
		if (next === countsByView(view.counting, line.class)) {
			if (counting) changes.undo(setNum, counting);
		} else
			changes.make(setNum, {
				partNum: line.partNum,
				color: own,
				what: { case: 'counted', value: next }
			});
	}

	const items = $derived([
		{
			group: 'Color',
			items: [
				{
					label: `${line.color?.name ?? 'Its color'}, its own`,
					checked: isOwn,
					onselect: () => choose([])
				},
				...line.partColors
					.filter((c) => c.color!.id !== own)
					.map((c) => ({
						label: c.color!.name,
						hint: count(c.pieces),
						checked: one === c.color!.id,
						onselect: () => choose([c.color!.id])
					})),
				{
					label: 'Several colors…',
					hint: several ? `${take!.colors.length} colors` : undefined,
					checked: several,
					onselect: () => onmix(line)
				},
				{ label: 'Any color', checked: !!take?.any, onselect: () => choose([], true) }
			]
		},
		'separator' as const,
		{
			label: 'Make it from other parts…',
			hint: swap?.what.case === 'swap' ? `${swap.what.value.standIns.length} chosen` : undefined,
			checked: !!swap,
			onselect: () => onswap(line)
		},
		{ label: 'Count it toward this set', checked: counted, onselect: toggleCounted },
		'separator' as const,
		{
			label: 'See the picture up close',
			disabled: !line.imageUrl,
			onselect: () => zoomsTo(caption)(line.imageUrl)
		}
	]);

	// A mark for several colors: a dot split into them.
	const mixed = $derived(
		line.useColors.length > 1
			? `conic-gradient(${line.useColors
					.map(
						(c, i, all) =>
							`${swatchColor(c.rgb) ?? 'transparent'} ${(i / all.length) * 360}deg ${((i + 1) / all.length) * 360}deg`
					)
					.join(', ')})`
			: ''
	);
</script>

<Menu
	bind:this={menu}
	label="{line.name}, {line.color?.name}"
	{items}
	placement="bottom-start"
	width="17rem"
>
	{#snippet trigger(props)}
		<button
			{...props}
			type="button"
			{title}
			aria-label="{caption}: {line.found} of {line.need} found"
			oncontextmenu={(e) => {
				e.preventDefault();
				// From the keyboard there is no pointer: open it from the tile.
				if (e.clientX || e.clientY) menu?.openAt(e);
				else e.currentTarget.click();
			}}
			class="group relative flex w-full min-w-0 flex-col items-center gap-0.5 rounded-control p-1 {fill}"
		>
			<!-- State is a fill over the tile's own. -->
			<span
				aria-hidden="true"
				class="pointer-events-none absolute inset-0 rounded-control transition-colors group-hover:bg-hover group-active:bg-pressed group-aria-expanded:bg-pressed"
			></span>
			<PartImage src={line.imageUrl} class="relative size-11" />
			<span class="num relative flex items-center gap-1 px-1 text-xs leading-none text-ink">
				{#if mixed}
					<span class="size-2 shrink-0 rounded-full ring-1 ring-line" style:background={mixed}
					></span>
				{:else if line.useColors.length}
					<span
						class="size-2 shrink-0 rounded-full ring-1 ring-line"
						style:background={swatchColor(line.useColors[0].rgb)}
					></span>
				{:else if line.anyColor}
					<span class="size-2 shrink-0 rounded-full ring-1 ring-ink-muted"></span>
				{/if}
				{#if line.made || swap}<Blocks size={10} class="shrink-0 text-ink-muted" />{/if}
				{line.found}/{line.need}
			</span>
		</button>
	{/snippet}
</Menu>
