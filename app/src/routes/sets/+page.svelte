<!--
	Every set, to browse: the sets the pile looks like, every one of them. "Explains the pile" picks
	sets one at a time, each taking its pieces before the next is picked, so a
	bucket of basic bricks is picked once and the sets under it show; "Each on
	its own" is every set with a piece in the pile, scored against the whole
	pile, in the order chosen. Rows are drawn a page at a time.
-->
<script lang="ts">
	import { Ranking, type GetOverviewResponse, type SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import { pile } from '$lib/api';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import SegmentedControl from '$lib/components/SegmentedControl.svelte';
	import Select from '$lib/components/Select.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import SetRow from '$lib/pile/SetRow.svelte';
	import { count, dayOf, dayTimeOf } from '$lib/format';

	const PAGE = 100;
	type Mode = 'explained' | 'alone';
	type Order = 'evidence' | 'complete' | 'found';
	let mode = $state<Mode>('explained');
	let order = $state<Order>('complete');
	let query = $state('');
	let drawn = $state(PAGE);
	let overview = $state<GetOverviewResponse | null>(null);
	let sets = $state<SetMatch[] | null>(null);
	let error = $state<string | null>(null);

	$effect(() => {
		pile.getOverview({}).then((o) => (overview = o), (e) => (error = String(e)));
	});

	$effect(() => {
		const ranking = mode === 'explained' ? Ranking.EXPLAINED : Ranking.ALONE;
		sets = null;
		pile.listSets({ ranking, limit: 0 }).then(
			(r) => (sets = r.sets),
			(e) => (error = String(e))
		);
	});

	const share = (s: SetMatch) => s.have / Math.max(1, s.need);
	const orders: Record<Order, (a: SetMatch, b: SetMatch) => number> = {
		evidence: (a, b) => b.evidence * b.weightedCompleteness - a.evidence * a.weightedCompleteness,
		complete: (a, b) => share(b) - share(a) || b.need - a.need,
		found: (a, b) => b.have - a.have || share(b) - share(a)
	};

	const shown = $derived.by(() => {
		if (!sets) return null;
		const q = query.trim().toLowerCase();
		const list = q
			? sets.filter((s) =>
					[s.name, s.setNum, s.theme, String(s.year)].some((v) => v.toLowerCase().includes(q))
				)
			: [...sets];
		return mode === 'alone' ? list.sort(orders[order]) : list;
	});

	// A new list starts at its first page.
	$effect(() => {
		void [mode, order, query];
		drawn = PAGE;
	});
</script>

<PageHeader
	title="Every set"
	description={overview
		? `Which sets the ${overview.machineName || 'sorter'}'s pieces from ${dayOf(overview.firstSeenUnix)} to ${dayOf(overview.lastSeenUnix)} make up, and how complete each is.`
		: 'Which sets the sorted pieces make up, and how complete each is.'}
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

<Panel flush>
	<div class="flex flex-wrap items-center gap-3 border-b border-line px-(--pad-panel) py-3">
		<SegmentedControl
			label="Ranking"
			bind:value={mode}
			options={[
				{ value: 'explained', label: 'Explains the pile' },
				{ value: 'alone', label: 'Each on its own' }
			]}
		/>
		{#if mode === 'alone'}
			<Select
				label="Order"
				class="w-48"
				bind:value={order}
				options={[
					{ value: 'complete', label: 'Most complete' },
					{ value: 'found', label: 'Most pieces found' },
					{ value: 'evidence', label: 'Best evidence' }
				]}
			/>
		{/if}
		<Input
			type="search"
			placeholder="Filter by name, number, theme or year"
			aria-label="Filter sets"
			bind:value={query}
			class="min-w-48 flex-1"
		/>
	</div>
	<p class="px-(--pad-panel) pt-3 text-sm text-ink-muted">
		{#if mode === 'explained'}
			Picked in order, best evidence first; each pick's pieces are taken out of the pile before the
			next, so no piece counts twice.
		{:else}
			Every set with a piece in the pile{shown ? ` (${count(shown.length)})` : ''}, each against the
			whole pile, so two sets can count the same piece. Sets made only of common bricks are complete
			in almost any big pile.
		{/if}
	</p>
	{#if shown === null}
		<div aria-busy="true" class="flex flex-col gap-3 p-(--pad-panel)">
			{#each { length: 6 } as _}<Skeleton class="h-16 w-full" />{/each}
		</div>
	{:else if shown.length === 0}
		<div class="p-(--pad-panel)">
			<EmptyState title={query ? 'No set matches that filter.' : 'No set matches the pile.'} />
		</div>
	{:else}
		<div class="divide-y divide-line">
			{#each shown.slice(0, drawn) as set (set.setNum + set.pick)}<SetRow {set} />{/each}
		</div>
		{#if shown.length > drawn}
			<div class="border-t border-line px-(--pad-panel) py-3">
				<Button onclick={() => (drawn += PAGE)}
					>Show {Math.min(PAGE, shown.length - drawn)} more of {count(shown.length - drawn)}</Button
				>
			</div>
		{/if}
	{/if}
</Panel>
