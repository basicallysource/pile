<!--
	The overview of the route's lots taken together (docs/views.md): what
	their pieces are, then what they make, through the view (exact or any
	color, and the sort-out queue): the queue, the sets that are all there,
	the ones nearly there, the ones half there, the ones the pieces most
	likely came from, the custom models they can build, and the minifigure
	and tiny sets last; the filter above the sets says which are listed. While a new view (or other lots) loads, the last one
	stays on screen and the reader's place holds when it lands; one already
	seen this visit draws at once (docs/loading-data.md).
-->
<script lang="ts">
	import { sources } from '$lib/api';
	import { query } from '$lib/data/query.svelte';
	import Loaded from '$lib/data/Loaded.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import Stat from '$lib/components/Stat.svelte';
	import Disclosure from '$lib/components/Disclosure.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import Distribution from '$lib/pile/Distribution.svelte';
	import SetSection from '$lib/pile/SetSection.svelte';
	import SetGrid from '$lib/pile/SetGrid.svelte';
	import SortOutQueue from '$lib/pile/SortOutQueue.svelte';
	import FilterBar from '$lib/pile/FilterBar.svelte';
	import { count, dayOf } from '$lib/format';
	import { routeStack, view } from '$lib/view.svelte';

	const stack = $derived(routeStack());
	const answer = query(sources.overview, () => ({ view: view.of(stack) }));
	const names = $derived(Object.fromEntries((answer.data?.lots ?? []).map((l) => [l.id, l.name])));
</script>

<Loaded query={answer} what="The overview" class="flex flex-col gap-(--gap-panels)">
	{#snippet waiting()}
		<div class="flex flex-col gap-3">
			<Skeleton class="h-10 w-80" /><Skeleton class="h-24 w-full" /><Skeleton class="h-96 w-full" />
		</div>
	{/snippet}
	{#snippet children(data)}
		{@const t = data.totals!}
		{@const one = data.lots.length === 1}
		<PageHeader
			title={data.lots.map((l) => l.name).join(' + ')}
			description={one
				? data.lots[0].description
				: `${data.lots.length} lots taken together. A set's pieces are pulled from the first lot that has them, then the next.`}
		>
			<p class="text-xs text-ink-faint">
				Sorted {dayOf(t.firstSeenUnix)} to {dayOf(t.lastSeenUnix)}{t.machineName
					? ` on ${t.machineName}`
					: ''}. {view.anyColor
					? 'Any color: a piece of the right part in another color counts, and each set says how many would be a recolor.'
					: 'Exact colors: a set counts only pieces of the right part in the right color.'} Printed parts,
				stickers, minifigures and their parts are left out of every count.
			</p>
		</PageHeader>

		<Panel flush>
			<div class="grid grid-cols-2 divide-line md:grid-cols-5 md:divide-x">
				<Stat label="Pieces" value={count(t.pieces)} hint="{count(t.partColors)} parts in a color" />
				<Stat label="Complete sets" value={count(data.complete?.total ?? 0)} />
				<Stat label="Almost complete" value={count(data.almost?.total ?? 0)} hint="80% or more" />
				<Stat label="Custom models" value={count(data.custom?.total ?? 0)} hint="60% or more" />
				<Stat label="Not in the catalog" value={count(t.unmatchedPieces)} hint="no Rebrickable match" />
			</div>
		</Panel>

		<Distribution categories={data.categories} colors={t.colors} pieces={t.pieces} />

		{#if data.sortOut.length}
			<SortOutQueue queue={data.sortOut} {stack} {names} pieces={t.pieces} left={data.piecesLeft} />
		{/if}

		<Panel flush>
			<div class="px-(--pad-panel) py-3"><FilterBar themes={data.themes} /></div>
		</Panel>

		<SetSection
			title="Complete"
			description="Every counted piece is here. The ones with rarer pieces first; the last are made of common bricks any big pile has."
			section={data.complete}
			{stack}
		/>
		<SetSection
			title="Almost complete"
			description="80% or more of the pieces are here. The ones with rarer pieces first."
			section={data.almost}
			{stack}
		/>
		<SetSection
			title="Half or more here"
			description="Sets of 50 or more pieces with at least half of them here, the most pieces found first."
			section={data.half}
			{stack}
		/>
		<SetSection
			title="Probably in the box"
			description="The sets these pieces most likely came from, judged by their rarer pieces in exact colors, most likely first."
			section={data.likely}
			{stack}
			numbered
		/>
		<SetSection
			title="Custom models"
			description="Free custom models on Rebrickable that are 60% or more here, most complete first."
			section={data.custom}
			{stack}
		/>
		<SetSection
			title="Minifigure sets"
			description="Sets that are basically a minifigure. Only their loose pieces are counted."
			section={data.minifigure}
			{stack}
			small
		/>
		{#if data.tiny?.total}
			<Panel flush>
				<div class="px-(--pad-panel) py-3">
					<Disclosure
						title="{count(data.tiny.total)} tiny sets"
						help="Complete, with under five counted pieces: key chains, gear and the like."
					>
						<SetGrid sets={data.tiny.sets} {stack} small />
					</Disclosure>
				</div>
			</Panel>
		{/if}
	{/snippet}
</Loaded>
