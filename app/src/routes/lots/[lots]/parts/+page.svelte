<!--
	The lot's bulk catalog: every part in every color, with its count, and the
	pieces the catalog cannot place. Downloads as a BrickLink XML list (the
	sorter's own BrickLink ids and colors).
-->
<script lang="ts">
	import Download from '@lucide/svelte/icons/download';
	import type { PartCount } from '$lib/gen/pile/v1/pile_pb';
	import { sources } from '$lib/api';
	import { query as ask } from '$lib/data/query.svelte';
	import Loaded from '$lib/data/Loaded.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import PartTile from '$lib/components/PartTile.svelte';
	import Input from '$lib/components/Input.svelte';
	import Select from '$lib/components/Select.svelte';
	import Button from '$lib/components/Button.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import { bricklinkXML } from '$lib/pile/bricklink';
	import { lotId } from '$lib/view.svelte';
	import { zoomsTo } from '$lib/pile/zoom.svelte';

	type Order = 'count' | 'color' | 'category';
	const lot = $derived(lotId());
	const answer = ask(sources.parts, () => ({ lotId: lot }), (r) => r.lotId);
	const data = $derived(answer.data);
	let query = $state('');
	let order = $state<Order>('count');

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
		// Freed once the browser has started saving it; freed at once, some
		// browsers save nothing.
		setTimeout(() => URL.revokeObjectURL(a.href), 10000);
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
	<Loaded query={answer} what="The parts">
		{#snippet waiting()}
			<div class="grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-x-3 gap-y-4 p-(--pad-panel)">
				{#each { length: 12 } as _}<Skeleton class="aspect-square w-full" />{/each}
			</div>
		{/snippet}
		{#snippet children()}
			<div class="grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-x-3 gap-y-4 p-(--pad-panel)">
				{#each shown as l (l.partNum + ':' + l.color?.id)}
					<PartTile
						layout="tile"
						name={l.name}
						bricklinkId={l.bricklinkId}
						partNum={l.partNum}
						imgUrl={l.imageUrl}
						onzoom={zoomsTo(`${l.name}${l.color ? `, ${l.color.name}` : ''}, ${l.partNum}`)}
						color={l.color ? { name: l.color.name, rgb: l.color.rgb } : null}
						quantity={l.count}
					/>
				{/each}
			</div>
		{/snippet}
	</Loaded>
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
