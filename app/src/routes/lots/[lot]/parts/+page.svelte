<!--
	The lot's bulk catalog: every part in every color, with its count, and the
	pieces the catalog cannot place. Downloads as a BrickLink XML list (the
	sorter's own BrickLink ids and colors).
-->
<script lang="ts">
	import Download from '@lucide/svelte/icons/download';
	import type { ListPartsResponse, PartCount } from '$lib/gen/pile/v1/pile_pb';
	import { pile } from '$lib/api';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import PartTile from '$lib/components/PartTile.svelte';
	import Input from '$lib/components/Input.svelte';
	import Select from '$lib/components/Select.svelte';
	import Button from '$lib/components/Button.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import { bricklinkXML } from '$lib/pile/bricklink';
	import { lotId } from '$lib/view.svelte';

	type Order = 'count' | 'color' | 'category';
	let data = $state<ListPartsResponse | null>(null);
	const lot = $derived(lotId());
	let error = $state<string | null>(null);
	let query = $state('');
	let order = $state<Order>('count');

	$effect(() => {
		pile.listParts({ lotId: lot }).then((r) => (data = r), (e) => (error = String(e)));
	});

	const total = $derived(data?.parts.reduce((n, l) => n + l.count, 0) ?? 0);
	const shown = $derived.by(() => {
		if (!data) return [];
		const q = query.trim().toLowerCase();
		const list = q
			? data.parts.filter((l) =>
					[l.name, l.partNum, l.bricklinkId, l.color?.name ?? '', l.category].some((v) =>
						v.toLowerCase().includes(q)
					)
				)
			: [...data.parts];
		const by: Record<Order, (a: PartCount, b: PartCount) => number> = {
			count: (a, b) => b.count - a.count,
			color: (a, b) => (a.color?.name ?? '').localeCompare(b.color?.name ?? '') || b.count - a.count,
			category: (a, b) => a.category.localeCompare(b.category) || b.count - a.count
		};
		return list.sort(by[order]);
	});

	function download() {
		if (!data) return;
		const blob = new Blob([bricklinkXML(data.parts)], { type: 'application/xml' });
		const a = document.createElement('a');
		a.href = URL.createObjectURL(blob);
		a.download = `${lot}-bricklink.xml`;
		a.click();
		URL.revokeObjectURL(a.href);
	}
</script>

<PageHeader
	title="Parts"
	description={data
		? `${total.toLocaleString()} pieces: ${data.parts.length.toLocaleString()} parts in a color.`
		: 'Every part in every color in this lot.'}
>
	{#snippet actions()}
		<Button variant="primary" icon={Download} onclick={download} disabled={!data}
			>BrickLink XML</Button
		>
	{/snippet}
</PageHeader>

{#if error}
	<Alert tone="danger" title="The pile did not load">{error}</Alert>
{/if}

<Panel flush>
	<div class="flex flex-wrap items-center gap-3 border-b border-line px-(--pad-panel) py-3">
		<Input
			type="search"
			placeholder="Filter by name, number, color or category"
			aria-label="Filter parts"
			bind:value={query}
			class="min-w-48 flex-1"
		/>
		<Select
			label="Order"
			class="w-44"
			bind:value={order}
			options={[
				{ value: 'count', label: 'Most first' },
				{ value: 'color', label: 'By color' },
				{ value: 'category', label: 'By category' }
			]}
		/>
	</div>
	{#if !data}
		<div aria-busy="true" class="grid grid-cols-2 gap-4 p-(--pad-panel) sm:grid-cols-4 lg:grid-cols-6">
			{#each { length: 12 } as _}<Skeleton class="aspect-square w-full" />{/each}
		</div>
	{:else}
		<div class="grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-x-3 gap-y-4 p-(--pad-panel)">
			{#each shown as l (l.partNum + ':' + l.color?.id)}
				<PartTile
					layout="tile"
					name={l.name}
					bricklinkId={l.bricklinkId}
					partNum={l.partNum}
					imgUrl={l.imageUrl}
					color={l.color ? { name: l.color.name, rgb: l.color.rgb } : null}
					quantity={l.count}
				/>
			{/each}
		</div>
	{/if}
</Panel>

{#if data?.unmatched.length}
	<Panel
		title="Not in the catalog"
		description="Pieces whose part or color has no Rebrickable match, as the sorter named them. No set can claim them."
		flush
	>
		<div class="divide-y divide-line">
			{#each data.unmatched as u (u.bricklinkId + u.colorName + u.name)}
				<PartTile
					name={u.name || u.bricklinkId}
					bricklinkId={u.bricklinkId}
					color={u.colorName ? { name: u.colorName } : null}
					quantity={u.count}
				/>
			{/each}
		</div>
	</Panel>
{/if}
