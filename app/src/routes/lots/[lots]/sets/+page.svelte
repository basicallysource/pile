<!--
	Every set and custom model with a piece in the lots, through the view, to
	browse: listed as the filter says (the same as the overview's), in the
	chosen order, and narrowed by words. Tiles are drawn a page at a time.
	Each stack's list is kept as it was left (its order, its words, how far
	it was drawn), and its answers last the visit, so coming back to it
	lands where it was scrolled (docs/loading-data.md).
-->
<script lang="ts" module>
	type OrderName = 'interesting' | 'biggest' | 'complex' | 'complete' | 'found';
	type Left = { order: OrderName; query: string; drawn: number };
	const PAGE = 200;
	// Each stack's list as it was left, for the visit.
	const left = new Map<string, Left>();
	const fresh: Left = { order: 'interesting', query: '', drawn: PAGE };
</script>

<script lang="ts">
	import { Order } from '$lib/gen/pile/v1/pile_pb';
	import { sources } from '$lib/api';
	import { query as ask } from '$lib/data/query.svelte';
	import Loaded from '$lib/data/Loaded.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import Select from '$lib/components/Select.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import SetGrid from '$lib/pile/SetGrid.svelte';
	import FilterBar from '$lib/pile/FilterBar.svelte';
	import { count } from '$lib/format';
	import { routeStack, view } from '$lib/view.svelte';
	import { untrack } from 'svelte';

	const stack = $derived(routeStack());
	const was = left.get(routeStack()) ?? fresh;
	let order = $state<OrderName>(was.order);
	let query = $state(was.query);
	let drawn = $state(was.drawn);

	const orders: Record<OrderName, Order> = {
		interesting: Order.INTERESTING,
		biggest: Order.BIGGEST,
		complex: Order.COMPLEX,
		complete: Order.COMPLETE,
		found: Order.FOUND
	};
	const answer = ask(sources.sets, () => ({ view: view.of(stack), order: orders[order] }));
	const themes = $derived(answer.data?.themes ?? []);

	const shown = $derived.by(() => {
		const sets = answer.data?.sets ?? [];
		const q = query.trim().toLowerCase();
		if (!q) return sets;
		return sets.filter((s) =>
			[s.name, s.setNum, s.theme, s.designer, String(s.year)].some((v) => v.toLowerCase().includes(q))
		);
	});

	// A new filter starts the list from its first page; coming back keeps it.
	const filters = $derived(JSON.stringify([order, query, view.filter]));
	let drawnFor = untrack(() => filters);
	$effect(() => {
		const now = filters;
		if (now !== drawnFor) {
			drawnFor = now;
			drawn = PAGE;
		}
	});

	$effect(() => {
		left.set(stack, { order, query, drawn });
	});
</script>

<PageHeader
	title="Every set"
	description="Every set and custom model with a piece in these lots that the filter lists, each against what the sort-out queue leaves."
/>

<Panel flush>
	<div class="flex flex-col gap-3 border-b border-line px-(--pad-panel) py-3">
		<FilterBar {themes} />
		<div class="flex flex-wrap items-center gap-3">
			<Select
				label="Order"
				class="w-52"
				bind:value={order}
				options={[
					{ value: 'interesting', label: 'Most interesting' },
					{ value: 'biggest', label: 'Biggest' },
					{ value: 'complex', label: 'Most different parts' },
					{ value: 'complete', label: 'Most complete' },
					{ value: 'found', label: 'Most pieces found' }
				]}
			/>
			<Input
				type="search"
				placeholder="Narrow by name, number, theme, designer or year"
				aria-label="Narrow the sets"
				bind:value={query}
				class="min-w-48 flex-1"
			/>
		</div>
	</div>
	<Loaded query={answer} what="The sets">
		{#snippet waiting()}
			<div class="grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-2 p-2">
				{#each { length: 16 } as _}<Skeleton class="aspect-square w-full" />{/each}
			</div>
		{/snippet}
		{#snippet children()}
			{#if shown.length === 0}
				<div class="p-(--pad-panel)">
					<EmptyState title={query ? 'Nothing matches those words.' : 'The filter lists nothing here.'} />
				</div>
			{:else}
				<p class="px-(--pad-panel) pt-3 text-sm text-ink-muted">{count(shown.length)} with a piece here.</p>
				<SetGrid sets={shown.slice(0, drawn)} />
				{#if shown.length > drawn}
					<div class="border-t border-line px-(--pad-panel) py-3">
						<Button onclick={() => (drawn += PAGE)}
							>Show {Math.min(PAGE, shown.length - drawn)} more of {count(shown.length - drawn)}</Button
						>
					</div>
				{/if}
			{/if}
		{/snippet}
	</Loaded>
</Panel>
