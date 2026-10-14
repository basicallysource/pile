<!--
	One set, custom model or build against the lots, as its sheet's body.
	First what it is, where its parts were read from when not Rebrickable
	(a 3D model of a set, a designer's digital file), and where the overview
	shows it (or why not), then how complete it is; a set read from a 3D
	model then lists its builds (a building, a vehicle), each opening in the
	sheet. Then its lines as small tiles, filled by how much of each was found,
	in stacked groups: what is missing first, then what was found, then what
	the view does not count toward completeness (still found, and sorted out
	with it), then its whole minifigures. Each group shows its first rows and
	opens to all of them. In the sort-out queue it is counted against what the
	sets before it left; otherwise against what the whole queue leaves.
-->
<script lang="ts">
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronUp from '@lucide/svelte/icons/chevron-up';
	import { Kind, Place, type GetSetResponse, type SetColor, type SetLine } from '$lib/gen/pile/v1/pile_pb';
	import Stat from '$lib/components/Stat.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import Button from '$lib/components/Button.svelte';
	import { count, percent } from '$lib/format';
	import { view } from '$lib/view.svelte';
	import { zoomsTo } from './zoom.svelte';
	import { hiddenAs } from './rules';
	import { classes } from './counting';
	import Palette from '@lucide/svelte/icons/palette';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import Menu from '$lib/components/Menu.svelte';
	import LineTile from './LineTile.svelte';
	import SetGrid from './SetGrid.svelte';

	let {
		detail,
		names
	}: {
		detail: GetSetResponse;
		// The lots' names, when there are several to tell apart.
		names?: Record<string, string>;
	} = $props();

	// How many tiles a group shows before it is opened: about two rows.
	const FOLD = 20;
	let opened = $state<Record<string, boolean>>({});

	const set = $derived(detail.set!);
	const custom = $derived(set.kind === Kind.CUSTOM);
	const build = $derived(set.kind === Kind.BUILD);
	const short = (l: SetLine) => l.need - l.found;
	const counted = $derived(detail.lines.filter((l) => l.counted));
	const missing = $derived(counted.filter((l) => l.found < l.need).sort((a, b) => short(b) - short(a)));
	const found = $derived(counted.filter((l) => l.found >= l.need));
	const uncounted = $derived(
		detail.lines.filter((l) => !l.counted).sort((a, b) => short(b) - short(a))
	);
	const pieces = (ls: SetLine[], f: (l: SetLine) => number) => ls.reduce((n, l) => n + f(l), 0);
	const totalPieces = $derived(pieces(detail.lines, (l) => l.need));
	// Where its parts were read from, in a sentence.
	const source = $derived.by(() => {
		if (!set.sourceAuthor) return '';
		if (build)
			return `Read from a 3D model of ${set.ofName} by ${set.sourceAuthor}: the parts that hold together, apart from the rest of the set.`;
		if (custom)
			return set.sourceAuthor === set.designer
				? `Its parts are from ${set.designer}'s own digital file.`
				: `Its parts are from a digital copy by ${set.sourceAuthor}, not ${set.designer}'s own file.`;
		if (detail.builds.length)
			return `Its builds are read from a 3D model of the set by ${set.sourceAuthor}, which places ${count(detail.sourcePlaced)} of its ${count(totalPieces)} pieces.`;
		return '';
	});
	const kindsOut = $derived(
		classes.filter((k) => !view.counting[k.key] && uncounted.some((l) => l.class === k.class))
	);

	const sectionName: Record<Place, string> = $derived({
		[Place.NONE]: '',
		[Place.COMPLETE]: 'Complete',
		[Place.ALMOST]: 'Almost complete',
		[Place.HALF]: 'Half or more here',
		[Place.LIKELY]: 'Probably in the box',
		[Place.CUSTOM]: 'Custom models',
		[Place.MINIFIGURE]: 'Minifigure sets',
		[Place.TINY]: 'Tiny sets',
		[Place.SORT_OUT]: 'Sorting out',
		[Place.HIDDEN]: '',
		[Place.BUILD]: 'Builds',
		[Place.COLLECTION]: set.collection
	});
	const share = (have: number, need: number) => percent(have / Math.max(1, need));
	// Where the overview shows it, and when it does not, why: in a few words.
	const where = $derived.by(() => {
		const mode = { exact: 'exact colors', similar: 'similar colors', any: 'any color' }[view.colors];
		const otherMode = view.colors === 'any' ? 'exact colors' : 'any color';
		if (detail.place === Place.HIDDEN)
			return `The filter hides it: ${detail.hiddenBy.map((r) => hiddenAs[r]).join(', ')}.`;
		if (detail.place !== Place.NONE) return `On the overview under ${sectionName[detail.place]}.`;
		let s = `Not on the overview: in ${mode}, ${count(set.have)} of its ${count(set.need)} pieces are here (${share(set.have, set.need)}).`;
		const o = detail.otherColors;
		if (o && detail.otherPlace !== Place.NONE && detail.otherPlace !== Place.HIDDEN)
			s += ` In ${otherMode}, ${count(o.have)} are (${share(o.have, o.need)})${o.wrongColor ? `, ${count(o.wrongColor)} of them in another color` : ''}, and it shows under ${sectionName[detail.otherPlace]} with ${otherMode === 'any color' ? 'Any color' : 'Exact colors'} on.`;
		return s;
	});
	const tiles = 'grid grid-cols-[repeat(auto-fill,minmax(3.5rem,1fr))] gap-1';
	// Pieces in other colors count: the view takes them, or a choice does.
	const recolors = $derived(view.colors !== 'exact' || set.wrongColor > 0);
	const changed = $derived(view.choices.some((c) => c.setNum === set.setNum));
	// A color's menu: its own, any, or one of the colors its parts are in.
	const colorItems = (c: SetColor) => {
		const id = c.color!.id;
		const own = !c.useColor && !c.anyColor;
		return [
			{ label: `${c.color!.name}, its own`, checked: own, onselect: () => view.choose(set.setNum, id, '', null) },
			{ label: 'Any color', checked: c.anyColor, onselect: () => view.choose(set.setNum, id, '', { any: true }) },
			...(c.swaps.length
				? [
						'separator' as const,
						{
							group: 'Take instead',
							items: c.swaps.map((s) => ({
								label: s.color!.name,
								hint: `${count(Math.min(s.pieces, c.need))} of ${count(c.need)}`,
								checked: c.useColor?.id === s.color!.id,
								onselect: () => view.choose(set.setNum, id, '', { use: s.color!.id })
							}))
						}
					]
				: [])
		];
	};
</script>

{#snippet group(key: string, title: string, sentence: string, lines: SetLine[])}
	{#if lines.length}
		{@const open = opened[key] ?? false}
		<section class="flex flex-col gap-2 border-b border-line px-5 py-4">
			<div class="flex items-start justify-between gap-4">
				<div class="min-w-0">
					<h3 class="text-sm font-semibold">{title}</h3>
					<p class="text-xs text-ink-muted">{sentence}</p>
				</div>
				{#if lines.length > FOLD}
					<Button
						size="sm"
						variant="ghost"
						icon={open ? ChevronUp : ChevronDown}
						aria-expanded={open}
						onclick={() => (opened = { ...opened, [key]: !open })}
						>{open ? 'Fewer' : `All ${count(lines.length)}`}</Button
					>
				{/if}
			</div>
			<div class={tiles}>
				{#each open ? lines : lines.slice(0, FOLD) as l (l.partNum + ':' + l.color?.id)}
					<LineTile line={l} {names} setNum={set.setNum} />
				{/each}
			</div>
		</section>
	{/if}
{/snippet}

<section class="flex gap-4 border-b border-line p-5">
	<PartImage src={set.imageUrl} onzoom={zoomsTo(`${set.name}, ${set.setNum}`)} class="size-28 shrink-0" />
	<div class="flex min-w-0 flex-col gap-2">
		<p class="text-sm text-ink-muted">
			{#if build}
				A build in {set.ofName}, {set.ofSetNum}{set.copies > 1 ? `, which has ${set.copies}` : ''}
			{:else if custom}
				{set.collection ? `${set.collection}, by ${set.designer}${set.year ? ` · ${set.year}` : ''}` : `Custom model by ${set.designer}`}
			{:else}
				{set.year} · {set.theme}
			{/if}
		</p>
		<div class="flex flex-wrap gap-1.5">
			{#if build}<Badge tone="info">build</Badge>{/if}
			{#if custom}<Badge tone="info">{set.collection || 'custom model'}</Badge>{/if}
			{#if set.pick}<Badge tone="primary">{set.pick} in the likely order</Badge>{/if}
			{#if set.sameContents.length}<Badge>also sold as {set.sameContents.join(', ')}</Badge>{/if}
		</div>
		<p class="text-sm text-ink">{where}</p>
		{#if source}
			<p class="text-xs text-ink-muted">
				{source}
				{#if set.sourceNotes}Its maker's note: “{set.sourceNotes}”.{/if}
				{#if detail.sourceUnplaced.length}<span title={detail.sourceUnplaced.join(', ')}
						>{count(detail.sourceUnplaced.length)} of its parts have no catalog part and are left out.</span
					>{/if}
				{#if set.sourceUrl}<a
						href={set.sourceUrl}
						target="_blank"
						rel="noreferrer"
						class="text-primary-ink underline-offset-2 hover:underline">The model's page</a
					>{/if}
			</p>
		{/if}
	</div>
</section>

{#if detail.builds.length}
	<section class="flex flex-col gap-1 border-b border-line pt-4">
		<div class="px-5">
			<h3 class="text-sm font-semibold">Builds</h3>
			<p class="text-xs text-ink-muted">
				The things it builds, each on its own: what of each is here, against what the view leaves.
			</p>
		</div>
		<SetGrid sets={detail.builds} small />
	</section>
{/if}

<section class="grid grid-cols-2 divide-line border-b border-line sm:grid-cols-4 sm:divide-x">
	<Stat
		label="Complete"
		value={percent(set.have / Math.max(1, set.need))}
		hint="{count(set.have)} of {count(set.need)} counted pieces"
	/>
	<Stat
		label={recolors ? 'In another color' : 'Of its rarer pieces'}
		value={recolors ? count(set.wrongColor) : percent(set.weightedCompleteness)}
		hint={recolors ? 'found, to recolor' : undefined}
	/>
	<Stat
		label="Not counted"
		value={count(pieces(uncounted, (l) => l.need))}
		hint="{count(pieces(uncounted, (l) => l.found))} of them found"
	/>
	<Stat label="Minifigures" value={set.minifigures} hint="whole, apart" />
</section>

{#if detail.colors.length}
	<section class="flex flex-col gap-2 border-b border-line px-5 py-4">
		<div class="flex items-start justify-between gap-4">
			<div class="min-w-0">
				<h3 class="text-sm font-semibold">Colors</h3>
				<p class="text-xs text-ink-muted">
					Swap one of its colors for another (red for dark red), or let any color do. Each says how many of its
					pieces the lots have in the other color. A single part's color is in its menu below.
				</p>
			</div>
			{#if changed}
				<Button size="sm" variant="ghost" icon={RotateCcw} onclick={() => view.clearChoices(set.setNum)}
					>Its own colors</Button
				>
			{/if}
		</div>
		<div class="flex flex-wrap gap-1.5">
			{#each detail.colors as c (c.color?.id)}
				<Menu label="{c.color?.name}: take instead" items={colorItems(c)} placement="bottom-start" width="17rem">
					{#snippet trigger(props)}
						<button
							{...props}
							type="button"
							class="flex h-8 items-center gap-2 rounded-button border px-2.5 text-xs {c.useColor || c.anyColor
								? 'border-primary bg-primary-soft text-primary-ink'
								: 'border-line text-ink hover:bg-hover'}"
						>
							<span class="size-3 shrink-0 rounded-full ring-1 ring-line" style="background: #{c.color?.rgb}"></span>
							<span class="whitespace-nowrap">{c.color?.name}</span>
							<span class="num text-ink-muted">{count(c.found)}/{count(c.need)}</span>
							{#if c.useColor}
								<span class="whitespace-nowrap">→</span>
								<span class="size-3 shrink-0 rounded-full ring-1 ring-line" style="background: #{c.useColor.rgb}"></span>
								<span class="whitespace-nowrap">{c.useColor.name}</span>
							{:else if c.anyColor}
								<span class="whitespace-nowrap">→ any color</span>
							{:else}
								<Palette size={12} class="shrink-0 text-ink-faint" />
							{/if}
						</button>
					{/snippet}
				</Menu>
			{/each}
		</div>
	</section>
{/if}

{@render group(
	'missing',
	'Missing',
	`${count(pieces(missing, short))} pieces in ${count(missing.length)} lines, the most missing first.`,
	missing
)}
{@render group(
	'found',
	'Found',
	`${count(pieces(found, (l) => l.need))} pieces in ${count(found.length)} lines, every piece of each.`,
	found
)}
{@render group(
	'uncounted',
	'Not counted',
	`${kindsOut.length ? kindsOut.map((k) => k.name.toLowerCase()).join(', ').replace(/^./, (c) => c.toUpperCase()) : 'Parts'}: left out of how complete it is (What counts, above the sets), and found or missing like the rest; what was found is sorted out with it.`,
	uncounted
)}

{#if detail.minifigures.length}
	<section class="flex flex-col gap-2 px-5 py-4">
		<div>
			<h3 class="text-sm font-semibold">Minifigures</h3>
			<p class="text-xs text-ink-muted">Whole figures, apart from the count.</p>
		</div>
		<div class={tiles}>
			{#each detail.minifigures as f (f.figNum)}
				<div class="flex min-w-0 flex-col items-center gap-0.5 rounded-control bg-well p-1" title="{f.name}, {f.figNum}">
					<PartImage src={f.imageUrl} onzoom={zoomsTo(f.name)} class="size-11" />
					<span class="num text-xs leading-none text-ink">×{f.quantity}</span>
				</div>
			{/each}
		</div>
	</section>
{/if}
