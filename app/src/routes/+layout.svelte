<!--
	The shell: the top bar and the page under it. In a lot the bar holds the
	lot's pages, the color mode and whether sets of loose bricks show, which
	apply on all of them; outside one
	it holds the collection and the design page. In a lot, the set open in
	the URL (`?set=`) shows in a sheet beside whichever page it is on, which
	narrows to make room. A
	picture seen up close shows in the Lightbox over everything.
-->
<script lang="ts">
	import '../app.css';
	import '@fontsource-variable/geist';
	import '@fontsource-variable/geist-mono';
	// Applies the saved mode and primary color, and keeps them applied.
	import '$lib/theme.svelte';
	import { page } from '$app/state';
	import TopBar from '$lib/components/TopBar.svelte';
	import Wordmark from '$lib/components/Wordmark.svelte';
	import SegmentedControl from '$lib/components/SegmentedControl.svelte';
	import Switch from '$lib/components/Switch.svelte';
	import SetSheet from '$lib/pile/SetSheet.svelte';
	import Lightbox from '$lib/components/Lightbox.svelte';
	import { unzoom, zoomed } from '$lib/pile/zoom.svelte';
	import { view } from '$lib/view.svelte';

	let { children } = $props();
	const lot = $derived(page.params.lot);
	const items = $derived(
		lot
			? [
					{ href: `/lots/${lot}`, label: 'Overview' },
					{ href: `/lots/${lot}/sets`, label: 'Every set' },
					{ href: `/lots/${lot}/parts`, label: 'Parts' },
					{ href: `/lots/${lot}/collect`, label: 'Time to collect' }
				]
			: [
					{ href: '/', label: 'Collection' },
					{ href: '/design', label: 'Design' }
				]
	);
</script>

<div class="flex min-h-dvh flex-col">
	<TopBar {items} collapse="sm">
		{#snippet brand()}<Wordmark name="Pile" href="/" />{/snippet}
		{#snippet end()}
			{#if lot}
				<label class="flex items-center gap-2 text-sm whitespace-nowrap text-ink-muted">
					<Switch checked={view.showBulk} onchange={(v) => view.setShowBulk(v)} />
					Bulk sets
				</label>
				<SegmentedControl
					label="Colors"
					size="sm"
					value={view.anyColor ? 'any' : 'exact'}
					onchange={(v) => view.setAnyColor(v === 'any')}
					options={[
						{ value: 'exact', label: 'Exact colors' },
						{ value: 'any', label: 'Any color' }
					]}
				/>
			{/if}
		{/snippet}
	</TopBar>
	<div class="flex flex-1 items-start">
		<main class="mx-auto flex w-full max-w-7xl min-w-0 flex-1 flex-col gap-(--gap-panels) p-4 md:p-6">
			{@render children()}
		</main>
		{#if lot}<SetSheet />{/if}
	</div>
</div>
<Lightbox src={zoomed()?.src} alt={zoomed()?.alt} onclose={unzoom} />
