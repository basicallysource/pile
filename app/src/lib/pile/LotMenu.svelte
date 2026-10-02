<!--
	The lot's name in the trail, opening a menu of every lot to switch to,
	the unlisted ones included: they open only from here, never from the
	collection page. The current lot has the check.
-->
<script lang="ts">
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import type { Lot } from '$lib/gen/pile/v1/pile_pb';
	import { pile } from '$lib/api';
	import Menu from '$lib/components/Menu.svelte';

	let { lot, name }: { lot: string; name: string } = $props();

	let lots = $state<Lot[]>([]);
	$effect(() => {
		pile.listLots({}).then(
			(r) => (lots = r.lots),
			() => (lots = [])
		);
	});
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
