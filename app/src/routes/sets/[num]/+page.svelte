<!--
	One set against the pile: how complete it is, and every line of its
	inventory with how many of that part and color are there. "Whole pile"
	counts everything sorted; "As explained" counts only the pieces this set
	was given when the pile was explained, so pieces another set took are not.
-->
<script lang="ts">
	import { page } from '$app/state';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import type { GetSetResponse, SetLine } from '$lib/gen/pile/v1/pile_pb';
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

	type Basis = 'pile' | 'explained';
	type Show = 'missing' | 'all' | 'found';
	let detail = $state<GetSetResponse | null>(null);
	let error = $state<string | null>(null);
	let basis = $state<Basis>('pile');
	let show = $state<Show>('all');

	$effect(() => {
		const setNum = page.params.num ?? '';
		detail = null;
		pile.getSet({ setNum }).then(
			(d) => {
				detail = d;
				basis = d.explained ? 'explained' : 'pile';
			},
			(e) => (error = String(e))
		);
	});

	const found = (l: SetLine) => (basis === 'pile' ? l.inPile : l.explained);
	const counted = $derived(detail?.lines.filter((l) => !l.printed) ?? []);
	const printed = $derived(detail?.lines.filter((l) => l.printed) ?? []);
	const lines = $derived(
		counted
			.filter((l) =>
				show === 'all' ? true : show === 'missing' ? found(l) < l.need : found(l) > 0
			)
			.sort((a, b) => found(a) / a.need - found(b) / b.need)
	);
	const match = $derived(basis === 'explained' && detail?.explained ? detail.explained : detail?.alone);
</script>

<div>
	<Button href="/" variant="ghost" size="sm" icon={ArrowLeft}>Sets</Button>
</div>

{#if error}
	<Alert tone="danger" title="This set did not load">{error}</Alert>
{:else if !detail || !match}
	<div aria-busy="true" class="flex flex-col gap-3">
		<Skeleton class="h-10 w-80" />
		<Skeleton class="h-24 w-full" />
		<Skeleton class="h-96 w-full" />
	</div>
{:else}
	{@const set = detail.alone!}
	<div class="flex flex-wrap items-start gap-4">
		<PartImage src={set.imageUrl} class="size-28 shrink-0" />
		<div class="min-w-0 flex-1">
			<PageHeader title={set.name} description="{set.setNum} · {set.year} · {set.theme}">
				<div class="flex flex-wrap gap-1.5">
					{#if detail.explained}
						<Badge tone="primary">picked {detail.explained.pick} explaining the pile</Badge>
					{:else}
						<Badge>not picked explaining the pile</Badge>
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
				value={percent(match.have / Math.max(1, match.need))}
				hint="{count(match.have)} of {count(match.need)} pieces"
			/>
			<Stat label="Of its rarer pieces" value={percent(match.weightedCompleteness)} />
			<Stat label="Printed parts and stickers" value={match.printedParts} hint="not counted" />
			<Stat label="Minifigures" value={match.minifigures} hint="not counted" />
		</div>
	</Panel>

	<Panel title="Inventory" flush>
		{#snippet actions()}
			<div class="flex flex-wrap gap-2">
				{#if detail?.explained}
					<SegmentedControl
						label="Count from"
						size="sm"
						bind:value={basis}
						options={[
							{ value: 'explained', label: 'As explained' },
							{ value: 'pile', label: 'Whole pile' }
						]}
					/>
				{/if}
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
		<div class="divide-y divide-line">
			{#each lines as l (l.partNum + ':' + l.color?.id)}
				<PartTile
					name={l.name}
					partNum={l.partNum}
					imgUrl={l.imageUrl}
					color={l.color ? { name: l.color.name, rgb: l.color.rgb } : null}
					quantity={l.need}
					found={found(l)}
				/>
			{:else}
				<p class="px-(--pad-panel) py-4 text-sm text-ink-muted">No lines to show.</p>
			{/each}
		</div>
	</Panel>

	{#if printed.length || detail.minifigures.length}
		<Panel
			title="Not counted"
			description="Printed parts, stickers and minifigures: the sorter is not expected to find them."
			flush
		>
			<div class="divide-y divide-line">
				{#each detail.minifigures as f (f.figNum)}
					<PartTile name={f.name} partNum={f.figNum} imgUrl={f.imageUrl} quantity={f.quantity} />
				{/each}
				{#each printed as l (l.partNum + ':' + l.color?.id)}
					<PartTile
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
