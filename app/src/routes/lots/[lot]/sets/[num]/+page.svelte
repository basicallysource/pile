<!--
	One set or custom model against the lot, through the view: how complete
	it is, and every line with what was found for it (in an any-color view,
	how many of those are another color). In the sort-out queue it is counted
	against what the sets before it left; otherwise against what the whole
	queue leaves. Its button puts it in or takes it out of the queue.
-->
<script lang="ts">
	import { page } from '$app/state';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Plus from '@lucide/svelte/icons/plus';
	import X from '@lucide/svelte/icons/x';
	import { Counting, Kind, type GetSetResponse, type SetLine } from '$lib/gen/pile/v1/pile_pb';
	import { pile } from '$lib/api';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import Stat from '$lib/components/Stat.svelte';
	import PartTile from '$lib/components/PartTile.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import SegmentedControl from '$lib/components/SegmentedControl.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import Button from '$lib/components/Button.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import { count, percent } from '$lib/format';
	import { lotId, view } from '$lib/view.svelte';

	type Show = 'all' | 'missing' | 'found';
	let detail = $state<GetSetResponse | null>(null);
	let error = $state<string | null>(null);
	let show = $state<Show>('all');
	const lot = $derived(lotId());
	const num = $derived(page.params.num ?? '');

	$effect(() => {
		const v = view.forLot(lot);
		const setNum = num;
		let stale = false;
		pile.getSet({ view: v, setNum }).then(
			(d) => !stale && (detail = d),
			(e) => (error = String(e))
		);
		return () => (stale = true);
	});

	const counted = $derived(detail?.lines.filter((l) => l.counting === Counting.COUNTED) ?? []);
	const uncounted = $derived(detail?.lines.filter((l) => l.counting !== Counting.COUNTED) ?? []);
	const lines = $derived(
		counted
			.filter((l) => (show === 'all' ? true : show === 'missing' ? l.found < l.need : l.found > 0))
			.sort((a, b) => a.found / a.need - b.found / b.need)
	);
	const grid = 'grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-x-3 gap-y-4 p-(--pad-panel)';
	const tile = (l: SetLine) => ({
		name: l.name,
		partNum: l.partNum,
		imgUrl: l.imageUrl,
		color: l.color ? { name: l.color.name, rgb: l.color.rgb } : null
	});
</script>

<div>
	<Button href="/lots/{lot}" variant="ghost" size="sm" icon={ArrowLeft}>Overview</Button>
</div>

{#if error}
	<Alert tone="danger" title="This set did not load">{error}</Alert>
{:else if !detail?.set}
	<div aria-busy="true" class="flex flex-col gap-3">
		<Skeleton class="h-10 w-80" /><Skeleton class="h-24 w-full" /><Skeleton class="h-96 w-full" />
	</div>
{:else}
	{@const set = detail.set}
	{@const custom = set.kind === Kind.CUSTOM}
	<div class="flex flex-wrap items-start gap-4">
		<PartImage src={set.imageUrl} class="size-28 shrink-0" />
		<div class="min-w-0 flex-1">
			<PageHeader
				title={set.name}
				description={custom ? `${set.setNum} · custom model by ${set.designer}` : `${set.setNum} · ${set.year} · ${set.theme}`}
			>
				{#snippet actions()}
					{#if set.url}<Button href={set.url} variant="ghost" icon={ExternalLink} target="_blank" rel="noreferrer"
							>Rebrickable</Button
						>{/if}
					<Button
						variant={detail?.sortOutPlace ? 'secondary' : 'primary'}
						icon={detail?.sortOutPlace ? X : Plus}
						onclick={() => view.toggle(lot, set.setNum)}
						>{detail?.sortOutPlace ? `Sorting out ${detail.sortOutPlace}: take out` : 'Sort this out'}</Button
					>
				{/snippet}
				<div class="flex flex-wrap gap-1.5">
					{#if custom}<Badge tone="info">custom model</Badge>{/if}
					{#if set.pick}<Badge tone="primary">{set.pick} in the likely order</Badge>{/if}
					{#if set.sameContents.length}<Badge>also sold as {set.sameContents.join(', ')}</Badge>{/if}
				</div>
			</PageHeader>
		</div>
	</div>

	<Panel flush>
		<div class="grid grid-cols-2 divide-line md:grid-cols-4 md:divide-x">
			<Stat
				label="Complete"
				value={percent(set.have / Math.max(1, set.need))}
				hint="{count(set.have)} of {count(set.need)} pieces"
			/>
			<Stat
				label={view.anyColor ? 'In another color' : 'Of its rarer pieces'}
				value={view.anyColor ? count(set.wrongColor) : percent(set.weightedCompleteness)}
				hint={view.anyColor ? 'found, to recolor' : undefined}
			/>
			<Stat label="Printed parts and stickers" value={set.printedParts} hint="not counted" />
			<Stat
				label="Minifigures"
				value={set.minifigures}
				hint={set.minifigureParts ? `and ${set.minifigureParts} figure parts, not counted` : 'not counted'}
			/>
		</div>
	</Panel>

	<Panel title="Inventory" flush>
		{#snippet actions()}
			<SegmentedControl
				label="Show"
				size="sm"
				bind:value={show}
				options={[
					{ value: 'all', label: 'All' },
					{ value: 'missing', label: 'Missing' },
					{ value: 'found', label: 'Found' }
				]}
			/>
		{/snippet}
		<div class={grid}>
			{#each lines as l (l.partNum + ':' + l.color?.id)}
				<PartTile layout="tile" {...tile(l)} quantity={l.need} found={l.found}>
					{#if l.wrongColor}<span class="text-xs text-warning-ink">{l.wrongColor} in another color</span>{/if}
				</PartTile>
			{:else}
				<p class="col-span-full text-sm text-ink-muted">No lines to show.</p>
			{/each}
		</div>
	</Panel>

	{#if uncounted.length || detail.minifigures.length}
		<Panel
			title="Not counted"
			description="Printed parts, stickers, minifigures and their parts: a set is complete without them."
			flush
		>
			<div class={grid}>
				{#each detail.minifigures as f (f.figNum)}
					<PartTile layout="tile" name={f.name} partNum={f.figNum} imgUrl={f.imageUrl} quantity={f.quantity} />
				{/each}
				{#each uncounted as l (l.partNum + ':' + l.color?.id)}
					<PartTile layout="tile" {...tile(l)} quantity={l.need} />
				{/each}
			</div>
		</Panel>
	{/if}
{/if}
