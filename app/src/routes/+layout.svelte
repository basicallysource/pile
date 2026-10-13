<!--
	The shell: the top bar and the page under it. In lots (`/lots/a,b/...`)
	the bar holds their pages and the color mode, which applies on all of
	them, and the page starts with the lots it looks at (StackBar); outside
	them the bar holds the collection and the design page. Find pieces, at
	the end of the bar on every page (or the / key), searches what the bins
	hold. On a narrow screen the color mode gives way to it. In lots, the set
	open in the URL (`?set=`) shows in a sheet beside whichever page it is
	on, which narrows to make room. A picture seen up close shows in the
	Lightbox over everything. While an answer takes more than a moment, the
	bar says what is updating; coming back to a page (Back, a reload) returns
	to where it was scrolled, even when its answers are slow to come
	(docs/loading-data.md).
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
	import Button from '$lib/components/Button.svelte';
	import Search from '@lucide/svelte/icons/search';
	import FindPieces from '$lib/pile/FindPieces.svelte';
	import SetSheet from '$lib/pile/SetSheet.svelte';
	import StackBar from '$lib/pile/StackBar.svelte';
	import Lightbox from '$lib/components/Lightbox.svelte';
	import { unzoom, zoomed } from '$lib/pile/zoom.svelte';
	import { view } from '$lib/view.svelte';
	import Updating from '$lib/data/Updating.svelte';
	import { clearRoom, resumeAt } from '$lib/data/place';
	import { afterNavigate } from '$app/navigation';
	import type { Snapshot } from '@sveltejs/kit';

	let { children } = $props();

	// Room kept at the end of a page for the reader's place is that page's.
	afterNavigate(({ from, to }) => {
		if (from?.url.pathname !== to?.url.pathname) clearRoom();
	});

	export const snapshot: Snapshot<number> = {
		capture: () => scrollY,
		restore: (y) => resumeAt(y)
	};
	let finding = $state(false);

	// The / key opens Find pieces, unless it is typed into a field.
	function onkeydown(event: KeyboardEvent) {
		const t = event.target as HTMLElement;
		if (event.key !== '/' || finding || t.closest('input, textarea, select, [contenteditable]'))
			return;
		event.preventDefault();
		finding = true;
	}
	const lots = $derived(page.params.lots);
	const items = $derived(
		lots
			? [
					{ href: `/lots/${lots}`, label: 'Overview' },
					{ href: `/lots/${lots}/sets`, label: 'Every set' },
					{ href: `/lots/${lots}/parts`, label: 'Parts' },
					{ href: `/lots/${lots}/collect`, label: 'Time to collect' }
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
			<Updating />
			{#if lots}
				<div class="hidden items-center gap-2 md:flex">
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
				</div>
			{/if}
			<Button size="sm" icon={Search} onclick={() => (finding = true)}>Find pieces</Button>
		{/snippet}
	</TopBar>
	<div class="flex flex-1 items-start">
		<main
			class="mx-auto flex w-full max-w-7xl min-w-0 flex-1 flex-col gap-(--gap-panels) p-4 md:p-6"
		>
			{#if lots}<StackBar />{/if}
			{@render children()}
		</main>
		{#if lots}<SetSheet />{/if}
	</div>
</div>
<Lightbox src={zoomed()?.src} alt={zoomed()?.alt} onclose={unzoom} />
<FindPieces bind:open={finding} />
<svelte:window {onkeydown} />
