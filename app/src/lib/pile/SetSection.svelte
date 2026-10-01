<!--
	One section of a lot's sets: a flush panel titled with its count, one
	sentence on what is in it, and its grid. When the server sent only the
	first of them, a link at the bottom goes to every set.
-->
<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import type { Section } from '$lib/gen/pile/v1/pile_pb';
	import Panel from '$lib/components/Panel.svelte';
	import Button from '$lib/components/Button.svelte';
	import { count } from '$lib/format';
	import SetGrid from './SetGrid.svelte';

	let {
		title,
		description,
		section,
		lot,
		numbered = false,
		small = false
	}: {
		title: string;
		description: string;
		section: Section | undefined;
		lot: string;
		numbered?: boolean;
		small?: boolean;
	} = $props();
</script>

{#if section && section.total > 0}
	<Panel title="{title} · {count(section.total)}" {description} flush>
		<SetGrid sets={section.sets} {lot} {numbered} {small} />
		{#if section.total > section.sets.length}
			<div class="border-t border-line px-(--pad-panel) py-3">
				<Button href="/lots/{lot}/sets" size="sm" icon={ArrowRight}
					>Showing {count(section.sets.length)} of {count(section.total)}: every set</Button
				>
			</div>
		{/if}
	</Panel>
{/if}
