<!--
	The lot's name in the trail, opening a menu of every lot to switch to,
	the unlisted ones included: they open only from here, never from the
	collection page. The current lot has the check.
-->
<script lang="ts">
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import { sources } from '$lib/api';
	import { query } from '$lib/data/query.svelte';
	import Menu from '$lib/components/Menu.svelte';

	let { lot, name }: { lot: string; name: string } = $props();

	const collection = query(sources.lots, () => ({}));
	const lots = $derived(collection.data?.lots ?? []);
</script>

{#if lots.length > 1}
	<Menu
		label="Lots"
		placement="bottom-start"
		items={lots.map((l) => ({ label: l.name, href: `/lots/${l.id}`, checked: l.id === lot }))}
	>
		{#snippet trigger(props)}
			<button {...props} type="button" class="inline-flex items-center gap-1 text-ink hover:text-ink-muted">
				{name}<ChevronDown size={14} class="text-ink-faint" />
			</button>
		{/snippet}
	</Menu>
{:else}
	<span class="text-ink">{name}</span>
{/if}
