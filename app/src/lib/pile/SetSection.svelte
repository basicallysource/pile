<!--
	One section of the overview's sets: a flush panel titled with its count, one
	sentence on what is in it, and its grid, drawn a page at a time with a
	button for more. When the server sent only the first of them, the last
	button goes to every set. How far each section was drawn is kept for the
	visit, so coming back to the page lands where it was scrolled. The button
	in its head folds it to its title and sentence, and that is kept in this
	browser (folded.svelte.ts); `startFolded` folds it until then.
-->
<script lang="ts" module>
	const PAGE = 150;
	const drawnOf = new Map<string, number>();
</script>

<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronUp from '@lucide/svelte/icons/chevron-up';
	import { untrack } from 'svelte';
	import type { Section } from '$lib/gen/pile/v1/pile_pb';
	import Panel from '$lib/components/Panel.svelte';
	import Button from '$lib/components/Button.svelte';
	import { count } from '$lib/format';
	import SetGrid from './SetGrid.svelte';
	import { folded } from './folded.svelte';

	let {
		title,
		description,
		section,
		stack,
		numbered = false,
		small = false,
		startFolded = false
	}: {
		title: string;
		description: string;
		section: Section | undefined;
		// The lots, as the route has them.
		stack: string;
		numbered?: boolean;
		small?: boolean;
		startFolded?: boolean;
	} = $props();

	const shut = $derived(folded.is(title, startFolded));

	const key = $derived(`${stack}:${title}`);
	let drawn = $state(untrack(() => drawnOf.get(key) ?? PAGE));
	$effect(() => {
		drawnOf.set(key, drawn);
	});
	const sets = $derived(section?.sets ?? []);
</script>

{#if section && section.total > 0}
	<Panel title="{title} · {count(section.total)}" {description} flush>
		{#snippet actions()}
			<Button
				size="sm"
				variant="ghost"
				icon={shut ? ChevronDown : ChevronUp}
				label={shut ? `Unfold ${title}` : `Fold ${title}`}
				aria-expanded={!shut}
				onclick={() => folded.set(title, !shut)}
			/>
		{/snippet}
		{#if !shut}
			<SetGrid sets={sets.slice(0, drawn)} {numbered} {small} />
			{#if sets.length > drawn}
				<div class="border-t border-line px-(--pad-panel) py-3">
					<Button size="sm" icon={ChevronDown} onclick={() => (drawn += PAGE)}
						>Show {count(Math.min(PAGE, sets.length - drawn))} more of {count(section.total - drawn)}</Button
					>
				</div>
			{:else if section.total > sets.length}
				<div class="border-t border-line px-(--pad-panel) py-3">
					<Button href="/lots/{stack}/sets" size="sm" icon={ArrowRight}
						>Showing {count(sets.length)} of {count(section.total)}: every set</Button
					>
				</div>
			{/if}
		{/if}
	</Panel>
{/if}
