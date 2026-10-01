<!--
	The shell: the top bar with the app's pages, and the page under it. Pages
	scroll as one column; the top bar stays.
-->
<script lang="ts">
	import '../app.css';
	import '@fontsource-variable/geist';
	import '@fontsource-variable/geist-mono';
	// Applies the saved mode and primary color, and keeps them applied.
	import '$lib/theme.svelte';
	import TopBar from '$lib/components/TopBar.svelte';
	import Wordmark from '$lib/components/Wordmark.svelte';

	let { children } = $props();
</script>

<div class="flex min-h-dvh flex-col">
	<TopBar
		items={[
			{ href: '/', label: 'Sets' },
			{ href: '/sets', label: 'Every set' },
			{ href: '/pile', label: 'Pile' },
			{ href: '/design', label: 'Design' }
		]}
	>
		{#snippet brand()}<Wordmark name="Pile" href="/" />{/snippet}
	</TopBar>
	<main class="mx-auto flex w-full max-w-6xl flex-col gap-(--gap-panels) p-4 md:p-6">
		{@render children()}
	</main>
</div>
