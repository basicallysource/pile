<!--
	What a Query shows, in each of its states (docs/loading-data.md):
	- on its way the first time: `waiting`, shaped like what is coming, so
	  nothing moves when it lands;
	- failed with nothing to show: an Alert;
	- shown: `children` with the answer. A newer answer on its way leaves it
	  in place; after a moment it dims (and the top bar says what is
	  updating). A newer one that fails leaves it too, under an Alert.

	The region is a place, and so is each block directly in it: when a newer
	answer lands, whichever the reader was looking at stays where it was on
	the screen (place.ts). So what it draws is blocks (panels, a grid, a
	list), never a list's items directly.

	<Loaded query={sets} what="The sets" class="flex flex-col gap-(--gap-panels)">
		{#snippet waiting()}<Skeleton class="h-96 w-full" />{/snippet}
		{#snippet children(answer)}<SetGrid sets={answer.sets} />{/snippet}
	</Loaded>
-->
<script lang="ts" generics="Res">
	import type { Snippet } from 'svelte';
	import Alert from '$lib/components/Alert.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import type { Query } from './query.svelte';

	let {
		query,
		what,
		waiting,
		children,
		class: className = ''
	}: {
		// Of any request.
		query: Query<any, Res>;
		// What it is, to say it did not load: "The sets".
		what: string;
		waiting?: Snippet;
		children: Snippet<[Res]>;
		class?: string;
	} = $props();
</script>

{#if query.data !== undefined}
	<div
		data-place="blocks"
		aria-busy={query.asking || undefined}
		class="transition-opacity {query.slow ? 'opacity-60' : ''} {className}"
	>
		{#if query.error}
			<Alert tone="danger" title="{what} did not update">
				{query.error}. What shows is from before.
			</Alert>
		{/if}
		{@render children(query.data)}
	</div>
{:else if query.error}
	<Alert tone="danger" title="{what} did not load">{query.error}</Alert>
{:else}
	<div aria-busy="true" class={className}>
		{#if waiting}{@render waiting()}{:else}<Skeleton class="h-96 w-full" />{/if}
	</div>
{/if}
