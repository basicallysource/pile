<!--
	Every set and custom model with a piece in the lot, through the view, to
	browse: filter by words, choose the order and the kind. Tiles are drawn a
	page at a time.
-->
<script lang="ts">
	import { Kind, Order, type SetMatch } from '$lib/gen/pile/v1/pile_pb';
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
	import SetGrid from '$lib/pile/SetGrid.svelte';
	import { count } from '$lib/format';
	import { lotId, view } from '$lib/view.svelte';

	const PAGE = 200;
	type OrderName = 'complete' | 'found' | 'evidence';
	type KindName = 'all' | 'sets' | 'custom';
	let order = $state<OrderName>('complete');
	let kind = $state<KindName>('all');
	let query = $state('');
	let drawn = $state(PAGE);
	let sets = $state<SetMatch[] | null>(null);
	let error = $state<string | null>(null);
	const lot = $derived(lotId());

	$effect(() => {
		const v = view.forLot(lot);
		const o = { complete: Order.COMPLETE, found: Order.FOUND, evidence: Order.EVIDENCE }[order];
		const k = { all: Kind.UNSPECIFIED, sets: Kind.SET, custom: Kind.CUSTOM }[kind];
		let stale = false;
		pile.listSets({ view: v, order: o, kind: k }).then(
			(r) => !stale && (sets = r.sets),
			(e) => (error = String(e))
		);
		return () => (stale = true);
	});

	const shown = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!sets || !q) return sets;
		return sets.filter((s) =>
			[s.name, s.setNum, s.theme, s.designer, String(s.year)].some((v) => v.toLowerCase().includes(q))
		);
	});

	$effect(() => {
		void [order, kind, query];
		drawn = PAGE;
	});
</script>

<PageHeader
	title="Every set"
	description="Every set and custom model with a piece in this lot, each against what the sort-out queue leaves."
/>

{#if error}
	<Alert tone="danger" title="The sets did not load">{error}</Alert>
{/if}

<Panel flush>
	<div class="flex flex-wrap items-center gap-3 border-b border-line px-(--pad-panel) py-3">
		<SegmentedControl
			label="Kind"
			bind:value={kind}
			options={[
				{ value: 'all', label: 'All' },
				{ value: 'sets', label: 'Sets' },
				{ value: 'custom', label: 'Custom models' }
			]}
		/>
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
		<Input
			type="search"
			placeholder="Filter by name, number, theme, designer or year"
			aria-label="Filter sets"
			bind:value={query}
			class="min-w-48 flex-1"
		/>
	</div>
	{#if shown === null}
		<div aria-busy="true" class="grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-2 p-2">
			{#each { length: 16 } as _}<Skeleton class="aspect-square w-full" />{/each}
		</div>
	{:else if shown.length === 0}
		<div class="p-(--pad-panel)">
			<EmptyState title={query ? 'Nothing matches that filter.' : 'Nothing matches the pile.'} />
		</div>
	{:else}
		<p class="px-(--pad-panel) pt-3 text-sm text-ink-muted">{count(shown.length)} with a piece here.</p>
		<SetGrid sets={shown.slice(0, drawn)} {lot} />
		{#if shown.length > drawn}
			<div class="border-t border-line px-(--pad-panel) py-3">
				<Button onclick={() => (drawn += PAGE)}
					>Show {Math.min(PAGE, shown.length - drawn)} more of {count(shown.length - drawn)}</Button
				>
			</div>
		{/if}
	{/if}
</Panel>
