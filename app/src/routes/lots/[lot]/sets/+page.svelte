<!--
	Every set and custom model with a piece in the lot, through the view, to
	browse: by kind, theme, licensed or not, and size, in the chosen order,
	and filtered by words. Tiles are drawn a page at a time.
-->
<script lang="ts">
	import { Kind, Order, type SetMatch, type ThemeCount } from '$lib/gen/pile/v1/pile_pb';
	import { pile } from '$lib/api';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import SegmentedControl from '$lib/components/SegmentedControl.svelte';
	import Select from '$lib/components/Select.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import Switch from '$lib/components/Switch.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import SetGrid from '$lib/pile/SetGrid.svelte';
	import { count } from '$lib/format';
	import { lotId, view } from '$lib/view.svelte';

	const PAGE = 200;
	type OrderName = 'interesting' | 'biggest' | 'complex' | 'complete' | 'found';
	type KindName = 'all' | 'sets' | 'custom';
	let order = $state<OrderName>('interesting');
	let kind = $state<KindName>('all');
	let theme = $state('');
	let licensedOnly = $state(false);
	let minPieces = $state('0');
	let themes = $state<ThemeCount[]>([]);
	let query = $state('');
	let drawn = $state(PAGE);
	let sets = $state<SetMatch[] | null>(null);
	let error = $state<string | null>(null);
	const lot = $derived(lotId());

	$effect(() => {
		const v = view.forLot(lot);
		const o = {
			interesting: Order.INTERESTING,
			biggest: Order.BIGGEST,
			complex: Order.COMPLEX,
			complete: Order.COMPLETE,
			found: Order.FOUND
		}[order];
		const k = { all: Kind.UNSPECIFIED, sets: Kind.SET, custom: Kind.CUSTOM }[kind];
		const req = { view: v, order: o, kind: k, themeGroup: theme, licensedOnly, minPieces: +minPieces };
		let stale = false;
		pile.listSets(req).then(
			(r) => {
				if (stale) return;
				sets = r.sets;
				themes = r.themes;
			},
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
		void [order, kind, query, theme, licensedOnly, minPieces];
		drawn = PAGE;
	});
</script>

<PageHeader
	title="Every set"
	description="Every set and custom model with a piece in this lot, each against what the sort-out queue leaves. Sets of loose bricks show only with Bulk sets on."
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
		<Select
			label="Theme"
			class="w-56"
			bind:value={theme}
			options={[
				{ value: '', label: 'Every theme' },
				...themes.map((t) => ({ value: t.name, label: t.name, hint: count(t.sets) })),
				...(theme && !themes.some((t) => t.name === theme) ? [{ value: theme, label: theme }] : [])
			]}
		/>
		<Select
			label="Size"
			class="w-40"
			bind:value={minPieces}
			options={[
				{ value: '0', label: 'Any size' },
				{ value: '50', label: '50+ pieces' },
				{ value: '100', label: '100+ pieces' },
				{ value: '250', label: '250+ pieces' },
				{ value: '500', label: '500+ pieces' },
				{ value: '1000', label: '1,000+ pieces' }
			]}
		/>
		<label class="flex items-center gap-2 text-sm whitespace-nowrap text-ink">
			<Switch bind:checked={licensedOnly} />
			Licensed only
		</label>
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
