<!--
	The set open beside a page of lots (`?set=` in the URL), in a Sheet: its name
	and number in the head with its Rebrickable link and the button that puts
	it in or takes it out of the sort-out queue, and SetDetail under them.
	The page beside it narrows and keeps its place; another set opened swaps it,
	and the same set in a new view keeps showing until its new numbers are in
	(docs/loading-data.md).
-->
<script lang="ts">
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Plus from '@lucide/svelte/icons/plus';
	import X from '@lucide/svelte/icons/x';
	import { Kind } from '$lib/gen/pile/v1/pile_pb';
	import { sources } from '$lib/api';
	import { query } from '$lib/data/query.svelte';
	import Loaded from '$lib/data/Loaded.svelte';
	import Sheet from '$lib/components/Sheet.svelte';
	import Button from '$lib/components/Button.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import { lotsOf, routeStack, view } from '$lib/view.svelte';
	import { closeSet, openSet } from './open-set.svelte';
	import SetDetail from './SetDetail.svelte';

	const stack = $derived(routeStack());
	const num = $derived(openSet());
	const answer = query(
		sources.set,
		() => (num ? { view: view.of(stack), setNum: num } : undefined),
		(r) => r.setNum
	);
	// The lots' names, when there are several to tell apart.
	const collection = query(sources.lots, () => ({}));
	const names = $derived(
		lotsOf(stack).length > 1
			? Object.fromEntries((collection.data?.lots ?? []).map((l) => [l.id, l.name]))
			: undefined
	);
	const detail = $derived(answer.data);
	const set = $derived(detail?.set);
	// Its place in the queue, from the view itself: the button changes on
	// the click, before the new numbers are in.
	const place = $derived(set ? view.queue(stack).indexOf(set.setNum) + 1 : 0);
</script>

{#if num}
<Sheet
	width="40rem"
	title={set?.name ?? num ?? ''}
	description={set ? (set.kind === Kind.CUSTOM ? `${set.setNum} · custom model` : set.setNum) : undefined}
	onclose={closeSet}
>
	{#snippet actions()}
		{#if set}
			{#if set.url}<Button href={set.url} variant="ghost" size="sm" icon={ExternalLink} target="_blank" rel="noreferrer"
					>Rebrickable</Button
				>{/if}
			<Button
				size="sm"
				variant={place ? 'secondary' : 'primary'}
				icon={place ? X : Plus}
				onclick={() => view.toggle(stack, set.setNum)}
				>{place ? `Sorting out ${place}: take out` : 'Sort this out'}</Button
			>
		{/if}
	{/snippet}
	<Loaded query={answer} what="This set">
		{#snippet waiting()}
			<div class="flex flex-col gap-3 p-5">
				<Skeleton class="h-28 w-full" /><Skeleton class="h-20 w-full" /><Skeleton class="h-96 w-full" />
			</div>
		{/snippet}
		{#snippet children(detail)}<SetDetail {detail} {names} />{/snippet}
	</Loaded>
</Sheet>
{/if}
