<!--
	One set or custom model against the lot, as its sheet's body: how complete
	it is, and every line with what was found for it (in an any-color view,
	how many of those are another color). In the sort-out queue it is counted
	against what the sets before it left; otherwise against what the whole
	queue leaves.
-->
<script lang="ts">
	import { Counting, Kind, Place, type GetSetResponse, type SetLine } from '$lib/gen/pile/v1/pile_pb';
	import Stat from '$lib/components/Stat.svelte';
	import PartTile from '$lib/components/PartTile.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import SegmentedControl from '$lib/components/SegmentedControl.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import { count, percent } from '$lib/format';
	import { view } from '$lib/view.svelte';

	let { detail }: { detail: GetSetResponse } = $props();

	type Show = 'all' | 'missing' | 'found';
	let show = $state<Show>('all');

	const set = $derived(detail.set!);
	const custom = $derived(set.kind === Kind.CUSTOM);
	const counted = $derived(detail.lines.filter((l) => l.counting === Counting.COUNTED));
	const uncounted = $derived(detail.lines.filter((l) => l.counting !== Counting.COUNTED));
	const lines = $derived(
		counted
			.filter((l) => (show === 'all' ? true : show === 'missing' ? l.found < l.need : l.found > 0))
			.sort((a, b) => a.found / a.need - b.found / b.need)
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
		[Place.BULK_HIDDEN]: ''
	};
	const share = (have: number, need: number) => percent(have / Math.max(1, need));
	// Where the overview shows it, and when it does not, why: in a few words.
	const where = $derived.by(() => {
		const mode = view.anyColor ? 'any color' : 'exact colors';
		const otherMode = view.anyColor ? 'exact colors' : 'any color';
		if (detail.place === Place.BULK_HIDDEN)
			return 'A set of loose bricks: the overview lists it only with Bulk sets on.';
		if (detail.place !== Place.NONE) return `On the overview under ${sectionName[detail.place]}.`;
		let s = `Not on the overview: in ${mode}, ${count(set.have)} of its ${count(set.need)} pieces are here (${share(set.have, set.need)}).`;
		const o = detail.otherColors;
		if (o && detail.otherPlace !== Place.NONE && detail.otherPlace !== Place.BULK_HIDDEN)
			s += ` In ${otherMode}, ${count(o.have)} are (${share(o.have, o.need)})${o.wrongColor ? `, ${count(o.wrongColor)} of them in another color` : ''}, and it shows under ${sectionName[detail.otherPlace]} with ${otherMode === 'any color' ? 'Any color' : 'Exact colors'} on.`;
		return s;
	});
	const grid = 'grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-x-3 gap-y-4';
	const tile = (l: SetLine) => ({
		name: l.name,
		partNum: l.partNum,
		imgUrl: l.imageUrl,
		color: l.color ? { name: l.color.name, rgb: l.color.rgb } : null
	});
</script>

<section class="flex gap-4 border-b border-line p-5">
	<PartImage src={set.imageUrl} class="size-28 shrink-0" />
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
		hint="{count(set.have)} of {count(set.need)} pieces"
	/>
	<Stat
		label={view.anyColor ? 'In another color' : 'Of its rarer pieces'}
		value={view.anyColor ? count(set.wrongColor) : percent(set.weightedCompleteness)}
		hint={view.anyColor ? 'found, to recolor' : undefined}
	/>
	<Stat label="Printed parts and stickers" value={set.printedParts} hint="not counted" />
	<Stat
		label="Minifigures"
		value={set.minifigures}
		hint={set.minifigureParts ? `and ${set.minifigureParts} figure parts, not counted` : 'not counted'}
	/>
</section>

<section class="flex flex-col gap-3 border-b border-line p-5">
	<div class="flex items-center justify-between gap-4">
		<h3 class="text-sm font-semibold">Inventory</h3>
		<SegmentedControl
			label="Show"
			size="sm"
			bind:value={show}
			options={[
				{ value: 'all', label: 'All' },
				{ value: 'missing', label: 'Missing' },
				{ value: 'found', label: 'Found' }
			]}
		/>
	</div>
	<div class={grid}>
		{#each lines as l (l.partNum + ':' + l.color?.id)}
			<PartTile layout="tile" {...tile(l)} quantity={l.need} found={l.found}>
				{#if l.wrongColor}<span class="text-xs text-warning-ink">{l.wrongColor} in another color</span>{/if}
			</PartTile>
		{:else}
			<p class="col-span-full text-sm text-ink-muted">No lines to show.</p>
		{/each}
	</div>
</section>

{#if uncounted.length || detail.minifigures.length}
	<section class="flex flex-col gap-3 p-5">
		<div>
			<h3 class="text-sm font-semibold">Not counted</h3>
			<p class="text-sm text-ink-muted">
				Printed parts, stickers, minifigures and their parts: a set is complete without them.
			</p>
		</div>
		<div class={grid}>
			{#each detail.minifigures as f (f.figNum)}
				<PartTile layout="tile" name={f.name} partNum={f.figNum} imgUrl={f.imageUrl} quantity={f.quantity} />
			{/each}
			{#each uncounted as l (l.partNum + ':' + l.color?.id)}
				<PartTile layout="tile" {...tile(l)} quantity={l.need} />
			{/each}
		</div>
	</section>
{/if}
