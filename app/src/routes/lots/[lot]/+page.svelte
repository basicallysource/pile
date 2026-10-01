<!--
	A lot: what its pieces are, then what they make, through the view (exact
	or any color, and the sort-out queue): the queue, the sets that are all
	there, the ones nearly there, the ones the pieces most likely came from,
	the custom models it can build, and the minifigure and tiny sets last.
	While a new view loads, the last one stays on screen.
-->
<script lang="ts">
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import type { GetLotResponse } from '$lib/gen/pile/v1/pile_pb';
	import { pile } from '$lib/api';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import Stat from '$lib/components/Stat.svelte';
	import Disclosure from '$lib/components/Disclosure.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import Distribution from '$lib/pile/Distribution.svelte';
	import SetSection from '$lib/pile/SetSection.svelte';
	import SetGrid from '$lib/pile/SetGrid.svelte';
	import SortOutQueue from '$lib/pile/SortOutQueue.svelte';
	import { count, dayOf } from '$lib/format';
	import { lotId, view } from '$lib/view.svelte';

	let data = $state<GetLotResponse | null>(null);
	let loading = $state(false);
	let error = $state<string | null>(null);
	const lot = $derived(lotId());

	$effect(() => {
		const v = view.forLot(lot);
		loading = true;
		let stale = false;
		pile.getLot({ view: v }).then(
			(r) => {
				if (stale) return;
				data = r;
				loading = false;
			},
			(e) => (error = String(e))
		);
		return () => (stale = true);
	});

	const l = $derived(data?.lot);
</script>

<div class="flex items-center gap-1 text-sm text-ink-muted">
	<a href="/" class="hover:text-ink">Collection</a>
	<ChevronRight size={14} />
	<span class="text-ink">{l?.name ?? lot}</span>
</div>

{#if error}
	<Alert tone="danger" title="The lot did not load">{error}</Alert>
{:else if !data || !l}
	<div aria-busy="true" class="flex flex-col gap-3">
		<Skeleton class="h-10 w-80" /><Skeleton class="h-24 w-full" /><Skeleton class="h-96 w-full" />
	</div>
{:else}
	<PageHeader title={l.name} description={l.description}>
		<p class="text-xs text-ink-faint">
			Sorted {dayOf(l.firstSeenUnix)} to {dayOf(l.lastSeenUnix)}{l.machineName
				? ` on ${l.machineName}`
				: ''}. {view.anyColor
				? 'Any color: a piece of the right part in another color counts, and each set says how many would be a recolor.'
				: 'Exact colors: a set counts only pieces of the right part in the right color.'} Printed parts,
			stickers, minifigures and their parts are left out of every count.
		</p>
	</PageHeader>

	<div class="flex flex-col gap-(--gap-panels) transition-opacity {loading ? 'opacity-60' : ''}">
		<Panel flush>
			<div class="grid grid-cols-2 divide-line md:grid-cols-5 md:divide-x">
				<Stat label="Pieces" value={count(l.pieces)} hint="{count(l.partColors)} parts in a color" />
				<Stat label="Complete sets" value={count(data.complete?.total ?? 0)} />
				<Stat label="Almost complete" value={count(data.almost?.total ?? 0)} hint="80% or more" />
				<Stat label="Custom models" value={count(data.custom?.total ?? 0)} hint="60% or more" />
				<Stat label="Not in the catalog" value={count(l.unmatchedPieces)} hint="no Rebrickable match" />
			</div>
		</Panel>

		<Distribution categories={data.categories} colors={l.colors} pieces={l.pieces} />

		{#if data.sortOut.length}
			<SortOutQueue queue={data.sortOut} {lot} pieces={l.pieces} left={data.piecesLeft} />
		{/if}

		<SetSection
			title="Complete"
			description="Every counted piece is here. The ones with rarer pieces first; the last are made of common bricks any big pile has."
			section={data.complete}
			{lot}
		/>
		<SetSection
			title="Almost complete"
			description="80% or more of the pieces are here. The ones with rarer pieces first."
			section={data.almost}
			{lot}
		/>
		<SetSection
			title="Probably in the box"
			description="The sets these pieces most likely came from, judged by their rarer pieces in exact colors, most likely first."
			section={data.likely}
			{lot}
			numbered
		/>
		<SetSection
			title="Custom models"
			description="Free custom models on Rebrickable that are 60% or more here, most complete first."
			section={data.custom}
			{lot}
		/>
		<SetSection
			title="Minifigure sets"
			description="Sets that are basically a minifigure. Only their loose pieces are counted."
			section={data.minifigure}
			{lot}
			small
		/>
		{#if data.tiny?.total}
			<Panel flush>
				<div class="px-(--pad-panel) py-3">
					<Disclosure
						title="{count(data.tiny.total)} tiny sets"
						help="Complete, with under five counted pieces: key chains, gear and the like."
					>
						<SetGrid sets={data.tiny.sets} {lot} small />
					</Disclosure>
				</div>
			</Panel>
		{/if}
	</div>
{/if}
