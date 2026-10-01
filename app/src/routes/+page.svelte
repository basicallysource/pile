<!--
	The sets the pile looks like. "Explains the pile" picks sets one at a time,
	each taking its pieces before the next is picked, so a bucket of basic
	bricks is picked once and the sets under it show; "Each on its own" scores
	every set against the whole pile.
-->
<script lang="ts">
	import { Ranking, type GetOverviewResponse, type SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import { pile } from '$lib/api';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import SegmentedControl from '$lib/components/SegmentedControl.svelte';
	import Input from '$lib/components/Input.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import Overview from '$lib/pile/Overview.svelte';
	import SetRow from '$lib/pile/SetRow.svelte';
	import { dayOf, dayTimeOf } from '$lib/format';

	type Mode = 'explained' | 'alone';
	let mode = $state<Mode>('explained');
	let query = $state('');
	let overview = $state<GetOverviewResponse | null>(null);
	let sets = $state<SetMatch[] | null>(null);
	let error = $state<string | null>(null);

	$effect(() => {
		pile.getOverview({}).then((o) => (overview = o), (e) => (error = String(e)));
	});

	$effect(() => {
		const ranking = mode === 'explained' ? Ranking.EXPLAINED : Ranking.ALONE;
		sets = null;
		pile.listSets({ ranking, limit: mode === 'alone' ? 300 : 0 }).then(
			(r) => (sets = r.sets),
			(e) => (error = String(e))
		);
	});

	const shown = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!sets || !q) return sets;
		return sets.filter((s) =>
			[s.name, s.setNum, s.theme, String(s.year)].some((v) => v.toLowerCase().includes(q))
		);
	});
</script>

<PageHeader
	title="Sets"
	description={overview
		? `Which sets the ${overview.machineName || 'sorter'}'s pieces from ${dayOf(overview.firstSeenUnix)} to ${dayOf(overview.lastSeenUnix)} make up, and how complete each is.`
		: 'Which sets the sorted pieces make up, and how complete each is.'}
>
	{#if overview}
		<p class="text-xs text-ink-faint">
			Records copied {dayTimeOf(overview.snapshotUnix)} · Rebrickable catalog of {dayOf(
				overview.catalogUnix
			)} · {overview.setsRanked.toLocaleString()} sets ranked. Printed parts, stickers and minifigures
			are left out of every count.
		</p>
	{/if}
</PageHeader>

{#if error}
	<Alert tone="danger" title="The pile did not load">{error}</Alert>
{/if}

{#if overview}<Overview {overview} />{/if}

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
			Every set against the whole pile, ranked by how much of its rarer pieces are there.
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
			{#each shown as set (set.setNum + set.pick)}<SetRow {set} />{/each}
		</div>
	{/if}
</Panel>
