<!--
	One set against the pile: how complete it is, and every line of its
	inventory with how many of that part and color the pile holds.
-->
<script lang="ts">
	import { page } from '$app/state';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import { Counting, type GetSetResponse, type SetLine } from '$lib/gen/pile/v1/pile_pb';
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

	type Show = 'missing' | 'all' | 'found';
	let detail = $state<GetSetResponse | null>(null);
	let error = $state<string | null>(null);
	let show = $state<Show>('all');

	$effect(() => {
		const setNum = page.params.num ?? '';
		detail = null;
		pile.getSet({ setNum }).then(
			(d) => (detail = d),
			(e) => (error = String(e))
		);
	});

	const found = (l: SetLine) => l.inPile;
	const counted = $derived(detail?.lines.filter((l) => l.counting === Counting.COUNTED) ?? []);
	const printed = $derived(detail?.lines.filter((l) => l.counting !== Counting.COUNTED) ?? []);
	const lines = $derived(
		counted
			.filter((l) =>
				show === 'all' ? true : show === 'missing' ? found(l) < l.need : found(l) > 0
			)
			.sort((a, b) => found(a) / a.need - found(b) / b.need)
	);
</script>

<div>
	<Button href="/" variant="ghost" size="sm" icon={ArrowLeft}>Sets</Button>
</div>

{#if error}
	<Alert tone="danger" title="This set did not load">{error}</Alert>
{:else if !detail?.set}
	<div aria-busy="true" class="flex flex-col gap-3">
		<Skeleton class="h-10 w-80" />
		<Skeleton class="h-24 w-full" />
		<Skeleton class="h-96 w-full" />
	</div>
{:else}
	{@const set = detail.set}
	<div class="flex flex-wrap items-start gap-4">
		<PartImage src={set.imageUrl} class="size-28 shrink-0" />
		<div class="min-w-0 flex-1">
			<PageHeader title={set.name} description="{set.setNum} · {set.year} · {set.theme}">
				<div class="flex flex-wrap gap-1.5">
					{#if set.pick}
						<Badge tone="primary">{set.pick} in the likely order</Badge>
					{:else}
						<Badge>not among the likely sets</Badge>
					{/if}
					{#if set.sameContents.length}<Badge tone="info"
							>also sold as {set.sameContents.join(', ')}</Badge
						>{/if}
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
			<Stat label="Of its rarer pieces" value={percent(set.weightedCompleteness)} />
			<Stat label="Printed parts and stickers" value={set.printedParts} hint="not counted" />
			<Stat
				label="Minifigures"
				value={set.minifigures}
				hint={set.minifigureParts
					? `and ${set.minifigureParts} figure parts, not counted`
					: 'not counted'}
			/>
		</div>
	</Panel>

	<Panel title="Inventory" flush>
		{#snippet actions()}
			<div class="flex flex-wrap gap-2">
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
			</div>
		{/snippet}
		<div class="grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-x-3 gap-y-4 p-(--pad-panel)">
			{#each lines as l (l.partNum + ':' + l.color?.id)}
				<PartTile
					layout="tile"
					name={l.name}
					partNum={l.partNum}
					imgUrl={l.imageUrl}
					color={l.color ? { name: l.color.name, rgb: l.color.rgb } : null}
					quantity={l.need}
					found={found(l)}
				/>
			{:else}
				<p class="col-span-full text-sm text-ink-muted">No lines to show.</p>
			{/each}
		</div>
	</Panel>

	{#if printed.length || detail.minifigures.length}
		<Panel
			title="Not counted"
			description="Printed parts, stickers, minifigures and their parts: a set is complete without them."
			flush
		>
			<div class="grid grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))] gap-x-3 gap-y-4 p-(--pad-panel)">
				{#each detail.minifigures as f (f.figNum)}
					<PartTile
						layout="tile"
						name={f.name}
						partNum={f.figNum}
						imgUrl={f.imageUrl}
						quantity={f.quantity}
					/>
				{/each}
				{#each printed as l (l.partNum + ':' + l.color?.id)}
					<PartTile
						layout="tile"
						name={l.name}
						partNum={l.partNum}
						imgUrl={l.imageUrl}
						color={l.color ? { name: l.color.name, rgb: l.color.rgb } : null}
						quantity={l.need}
					/>
				{/each}
			</div>
		</Panel>
	{/if}
{/if}
