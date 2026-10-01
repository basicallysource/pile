<!--
	The design system, shown with the components this app uses, on sample
	data. The system is the Sorter's (docs/design-system/README.md): its
	tokens are src/app.css and its components src/lib/components/, copied
	unchanged from the Sorter design system.
-->
<script lang="ts">
	import Download from '@lucide/svelte/icons/download';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import Stat from '$lib/components/Stat.svelte';
	import PartTile from '$lib/components/PartTile.svelte';
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import SegmentedControl from '$lib/components/SegmentedControl.svelte';
	import Button from '$lib/components/Button.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import Input from '$lib/components/Input.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { theme } from '$lib/theme.svelte';
	import { create } from '@bufbuild/protobuf';
	import { SetMatchSchema } from '$lib/gen/pile/v1/pile_pb';
	import SetGrid from '$lib/pile/SetGrid.svelte';

	let mode = $state<'explained' | 'alone'>('explained');
	const red = { name: 'Red', rgb: 'C91A09' };
	const img = 'https://cdn.rebrickable.com/media/parts/elements/300121.jpg';
	const sample = (setNum: string, have: number, need: number, pick = 0) =>
		create(SetMatchSchema, {
			setNum,
			name: 'Speedboat',
			year: 2006,
			theme: 'Creator',
			imageUrl: 'https://cdn.rebrickable.com/media/sets/7610-1.jpg',
			have,
			need,
			pick
		});
</script>

<PageHeader
	title="Design"
	description="The Sorter's design system, with the components this app is built from."
>
	{#snippet actions()}
		<SegmentedControl
			label="Mode"
			size="sm"
			value={theme.mode}
			onchange={(m) => theme.setMode(m)}
			options={[
				{ value: 'light', label: 'Light' },
				{ value: 'dark', label: 'Dark' }
			]}
		/>
	{/snippet}
</PageHeader>

<Panel title="Numbers" description="A Stat row in a flush panel: the totals at the top of a page." flush>
	<div class="grid grid-cols-2 divide-line md:grid-cols-4 md:divide-x">
		<Stat label="Pieces" value="6,056" />
		<Stat label="Lots" value="1,971" hint="parts in a color" />
		<Stat label="Explained by sets" value="72%" hint="4,376 pieces" />
		<Stat label="Not in the catalog" value="52" hint="no Rebrickable match" />
	</div>
</Panel>

<Panel title="Controls" description="One primary action a page; choices that apply at once are a segmented control.">
	<div class="flex flex-wrap items-center gap-3">
		<Button variant="primary" icon={Download}>BrickLink XML</Button>
		<Button>Secondary</Button>
		<Button variant="ghost">Ghost</Button>
		<SegmentedControl
			label="Ranking"
			bind:value={mode}
			options={[
				{ value: 'explained', label: 'Explains the pile' },
				{ value: 'alone', label: 'Each on its own' }
			]}
		/>
		<Input type="search" placeholder="Filter" aria-label="Filter" class="w-56" />
	</div>
	<div class="mt-4 flex flex-wrap gap-1.5">
		<Badge>12 printed not counted</Badge>
		<Badge>3 minifigures</Badge>
		<Badge tone="info">also sold as 3033-1</Badge>
		<Badge tone="primary">2 in the likely order</Badge>
	</div>
</Panel>

<Panel title="Parts" description="PartTile: a line of a set's inventory (found of needed), and a lot in the pile." flush>
	<div class="divide-y divide-line">
		<PartTile name="Brick 2 x 4" partNum="3001" imgUrl={img} color={red} quantity={6} found={3} />
		<PartTile name="Brick 2 x 4" partNum="3001" imgUrl={img} color={red} quantity={6} found={6} />
	</div>
	<div class="grid grid-cols-2 gap-4 p-(--pad-panel) sm:grid-cols-6">
		<PartTile layout="tile" name="Brick 2 x 4" bricklinkId="3001" imgUrl={img} color={red} quantity={40} />
	</div>
</Panel>

<Panel
	title="Sets"
	description="SetGrid of SetTiles: the picture, a small caption, a thin bar for how much the pile holds, green when complete."
	flush
>
	<SetGrid sets={[sample('7610-1', 18, 18), sample('7610-2', 15, 18), sample('7610-3', 4, 18)]} />
</Panel>

<Panel title="Progress" description="How complete a set is: the bar, the share, and the pieces.">
	<div class="flex max-w-sm flex-col gap-1.5">
		<div class="flex items-baseline justify-between text-sm">
			<span class="num font-medium">69%</span><span class="num text-ink-muted">396 / 574</span>
		</div>
		<ProgressBar value={396} max={574} label="Sample: 69% complete" />
	</div>
</Panel>

<Panel title="States" description="Every list has its loading, empty and error states.">
	<div class="flex flex-col gap-4">
		<Skeleton class="h-16 w-full" />
		<EmptyState title="No set matches that filter." />
		<Alert tone="danger" title="The pile did not load">The server could not be reached.</Alert>
	</div>
</Panel>
