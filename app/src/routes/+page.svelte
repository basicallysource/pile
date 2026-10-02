<!--
	The collection: every lot of sorted pieces (one box, one sort), each
	opening to what it holds and what it builds.
-->
<script lang="ts">
	import type { ListLotsResponse } from '$lib/gen/pile/v1/pile_pb';
	import { pile } from '$lib/api';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import LotCard from '$lib/pile/LotCard.svelte';
	import { count, dayOf } from '$lib/format';

	let data = $state<ListLotsResponse | null>(null);
	let error = $state<string | null>(null);

	// Unlisted lots open only from a lot's menu.
	const listed = $derived(data?.lots.filter((l) => !l.unlisted) ?? []);

	$effect(() => {
		pile.listLots({}).then((r) => (data = r), (e) => (error = String(e)));
	});
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

{#if error}
	<Alert tone="danger" title="The collection did not load">{error}</Alert>
{:else if !data}
	<div aria-busy="true" class="grid gap-(--gap-panels) md:grid-cols-2"><Skeleton class="h-40" /><Skeleton class="h-40" /></div>
{:else if listed.length === 0}
	<EmptyState title="No lots yet." />
{:else}
	<div class="grid gap-(--gap-panels) md:grid-cols-2">
		{#each listed as lot (lot.id)}<LotCard {lot} />{/each}
	</div>
{/if}
