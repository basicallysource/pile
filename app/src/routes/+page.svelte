<!--
	The collection: every lot of sorted pieces (one box, one sort), each
	opening to what it holds and what it builds.
-->
<script lang="ts">
	import { sources } from '$lib/api';
	import { query } from '$lib/data/query.svelte';
	import Loaded from '$lib/data/Loaded.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import LotCard from '$lib/pile/LotCard.svelte';
	import { count, dayOf } from '$lib/format';

	const lots = query(sources.lots, () => ({}));
	const data = $derived(lots.data);
</script>

<PageHeader
	title="Collection"
	description="Every lot of sorted pieces. Open one to see what is in it, the sets it completes or nearly completes, the ones it most likely came from, and the custom models it can build."
>
	{#if data}
		<p class="text-xs text-ink-faint">
			Matched against {count(data.setsChecked)} sets and {count(data.customModelsChecked)} custom models
			· Rebrickable catalog of {dayOf(data.catalogUnix)}
		</p>
	{/if}
</PageHeader>

<Loaded query={lots} what="The collection">
	{#snippet waiting()}
		<div class="grid gap-(--gap-panels) md:grid-cols-2"><Skeleton class="h-40" /><Skeleton class="h-40" /></div>
	{/snippet}
	{#snippet children(data)}
		<!-- Unlisted lots open only from a lot's menu. -->
		{@const listed = data.lots.filter((l) => !l.unlisted)}
		{#if listed.length === 0}
			<EmptyState title="No lots yet." />
		{:else}
			<div class="grid gap-(--gap-panels) md:grid-cols-2">
				{#each listed as lot (lot.id)}<LotCard {lot} />{/each}
			</div>
		{/if}
	{/snippet}
</Loaded>
