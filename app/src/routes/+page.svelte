<!--
	The answer: the sets the pile holds whole, then the sets it most likely
	came from that are still missing pieces (the explained picks, in the
	order picked). Sets of under five counted pieces (a key chain, gear)
	are complete in any pile, so they fold away. Sets that are basically a
	minifigure go last, smaller, complete or likely. Every set is browsable at
	/sets.
-->
<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import { Ranking, type GetOverviewResponse, type SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import { pile } from '$lib/api';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import Disclosure from '$lib/components/Disclosure.svelte';
	import Button from '$lib/components/Button.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import Overview from '$lib/pile/Overview.svelte';
	import SetGrid from '$lib/pile/SetGrid.svelte';
	import { count, dayOf, dayTimeOf } from '$lib/format';

	// Fewer counted pieces than this and a set is complete in any pile.
	const TINY = 5;

	let overview = $state<GetOverviewResponse | null>(null);
	let complete = $state<SetMatch[] | null>(null);
	let picks = $state<SetMatch[] | null>(null);
	let error = $state<string | null>(null);

	$effect(() => {
		const fail = (e: unknown) => (error = String(e));
		pile.getOverview({}).then((o) => (overview = o), fail);
		pile.listSets({ ranking: Ranking.COMPLETE }).then((r) => (complete = r.sets), fail);
		pile.listSets({ ranking: Ranking.EXPLAINED }).then((r) => (picks = r.sets), fail);
	});

	const builds = $derived(complete?.filter((s) => !s.minifigureSet && s.need >= TINY) ?? []);
	const tiny = $derived(complete?.filter((s) => !s.minifigureSet && s.need < TINY) ?? []);
	// The picks the pile does not hold whole (those are listed as complete).
	const likely = $derived(picks?.filter((s) => !s.minifigureSet && s.have < s.need) ?? null);
	const figures = $derived.by(() => {
		if (!complete || !picks) return null;
		const seen = new Set<string>();
		return [...complete, ...picks].filter(
			(s) => s.minifigureSet && !seen.has(s.setNum) && seen.add(s.setNum)
		);
	});
</script>

<PageHeader
	title="Sets"
	description={overview
		? `What the ${overview.machineName || 'sorter'}'s pieces from ${dayOf(overview.firstSeenUnix)} to ${dayOf(overview.lastSeenUnix)} make up: the sets that are all there, then the ones still missing pieces.`
		: 'The sets the sorted pieces make up.'}
>
	{#if overview}
		<p class="text-xs text-ink-faint">
			Records copied {dayTimeOf(overview.snapshotUnix)} · Rebrickable catalog of {dayOf(
				overview.catalogUnix
			)} · every one of {count(overview.setsRanked)} sets checked. Printed parts, stickers and minifigures
			are left out of every count.
		</p>
	{/if}
</PageHeader>

{#if error}
	<Alert tone="danger" title="The pile did not load">{error}</Alert>
{/if}

{#if overview}<Overview {overview} />{/if}

{#snippet loading()}
	<div aria-busy="true" class="flex flex-col gap-3 p-(--pad-panel)">
		{#each { length: 4 } as _}<Skeleton class="h-16 w-full" />{/each}
	</div>
{/snippet}

<Panel
	title={complete ? `Complete · ${count(builds.length)}` : 'Complete'}
	description="Every counted piece is in the pile. The ones with rarer pieces come first; the last are made of common bricks any big pile has."
	flush
>
	{#if complete === null}
		{@render loading()}
	{:else if builds.length === 0}
		<div class="p-(--pad-panel)"><EmptyState title="No set is all there." /></div>
	{:else}
		<SetGrid sets={builds} />
	{/if}
	{#if tiny.length}
		<div class="border-t border-line px-(--pad-panel) py-3">
			<Disclosure
				title="{count(tiny.length)} tiny sets"
				help="Under {TINY} counted pieces: key chains, gear, and figures whose printed parts are not counted."
			>
				<SetGrid sets={tiny} />
			</Disclosure>
		</div>
	{/if}
</Panel>

<Panel
	title={likely ? `Likely in the box, still missing pieces · ${count(likely.length)}` : 'Likely in the box, still missing pieces'}
	description="The sets the pile most likely came from, most likely first, with what the pile holds of each."
	flush
>
	{#if likely === null}
		{@render loading()}
	{:else if likely.length === 0}
		<div class="p-(--pad-panel)"><EmptyState title="No other set stands out." /></div>
	{:else}
		<SetGrid sets={likely} numbered />
	{/if}
</Panel>

{#if figures?.length}
	<Panel
		title="Minifigure sets · {count(figures.length)}"
		description="Sets that are basically a minifigure, complete or likely. Only their loose pieces are counted."
		flush
	>
		<SetGrid sets={figures} small />
	</Panel>
{/if}

<div>
	<Button href="/sets" icon={ArrowRight}>Browse every set</Button>
</div>
