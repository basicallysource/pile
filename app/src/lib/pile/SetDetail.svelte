<!--
	One set or custom model against the lots, as its sheet's body. First what
	it is and where the overview shows it (or why not), then how complete it
	is. Then its lines as small tiles, filled by how much of each was found,
	in stacked groups: what is missing first, then what was found, then what
	the view does not count toward completeness (still found, and sorted out
	with it), then its whole minifigures. Each group shows its first rows and
	opens to all of them. In the sort-out queue it is counted against what the
	sets before it left; otherwise against what the whole queue leaves.
-->
<script lang="ts">
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronUp from '@lucide/svelte/icons/chevron-up';
	import { Kind, Place, type GetSetResponse, type SetLine } from '$lib/gen/pile/v1/pile_pb';
	import Stat from '$lib/components/Stat.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import Button from '$lib/components/Button.svelte';
	import { count, percent } from '$lib/format';
	import { view } from '$lib/view.svelte';
	import { zoomsTo } from './zoom.svelte';
	import { hiddenAs } from './rules';
	import { classes } from './counting';
	import LineTile from './LineTile.svelte';

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
	const short = (l: SetLine) => l.need - l.found;
	const counted = $derived(detail.lines.filter((l) => l.counted));
	const missing = $derived(counted.filter((l) => l.found < l.need).sort((a, b) => short(b) - short(a)));
	const found = $derived(counted.filter((l) => l.found >= l.need));
	const uncounted = $derived(
		detail.lines.filter((l) => !l.counted).sort((a, b) => short(b) - short(a))
	);
	const pieces = (ls: SetLine[], f: (l: SetLine) => number) => ls.reduce((n, l) => n + f(l), 0);
	const kindsOut = $derived(
		classes.filter((k) => !view.counting[k.key] && uncounted.some((l) => l.class === k.class))
	);

	const sectionName: Record<Place, string> = {
		[Place.NONE]: '',
		[Place.COMPLETE]: 'Complete',
		[Place.ALMOST]: 'Almost complete',
		[Place.HALF]: 'Half or more here',
		[Place.LIKELY]: 'Probably in the box',
		[Place.CUSTOM]: 'Custom models',
		[Place.MINIFIGURE]: 'Minifigure sets',
		[Place.TINY]: 'Tiny sets',
		[Place.SORT_OUT]: 'Sorting out',
		[Place.HIDDEN]: ''
	};
	const share = (have: number, need: number) => percent(have / Math.max(1, need));
	// Where the overview shows it, and when it does not, why: in a few words.
	const where = $derived.by(() => {
		const mode = view.anyColor ? 'any color' : 'exact colors';
		const otherMode = view.anyColor ? 'exact colors' : 'any color';
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
					<LineTile line={l} {names} />
				{/each}
			</div>
		</section>
	{/if}
{/snippet}

<section class="flex gap-4 border-b border-line p-5">
	<PartImage src={set.imageUrl} onzoom={zoomsTo(`${set.name}, ${set.setNum}`)} class="size-28 shrink-0" />
	<div class="flex min-w-0 flex-col gap-2">
		<p class="text-sm text-ink-muted">
			{custom ? `Custom model by ${set.designer}` : `${set.year} · ${set.theme}`}
		</p>
		<div class="flex flex-wrap gap-1.5">
			{#if custom}<Badge tone="info">custom model</Badge>{/if}
			{#if set.pick}<Badge tone="primary">{set.pick} in the likely order</Badge>{/if}
			{#if set.sameContents.length}<Badge>also sold as {set.sameContents.join(', ')}</Badge>{/if}
		</div>
		<p class="text-sm text-ink">{where}</p>
	</div>
</section>

<section class="grid grid-cols-2 divide-line border-b border-line sm:grid-cols-4 sm:divide-x">
	<Stat
		label="Complete"
		value={percent(set.have / Math.max(1, set.need))}
		hint="{count(set.have)} of {count(set.need)} counted pieces"
	/>
	<Stat
		label={view.anyColor ? 'In another color' : 'Of its rarer pieces'}
		value={view.anyColor ? count(set.wrongColor) : percent(set.weightedCompleteness)}
		hint={view.anyColor ? 'found, to recolor' : undefined}
	/>
	<Stat
		label="Not counted"
		value={count(pieces(uncounted, (l) => l.need))}
		hint="{count(pieces(uncounted, (l) => l.found))} of them found"
	/>
	<Stat label="Minifigures" value={set.minifigures} hint="whole, apart" />
</section>

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
