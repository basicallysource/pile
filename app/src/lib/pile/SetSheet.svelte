<!--
	The set open over a lot's page (`?set=` in the URL), in a Sheet: its name
	and number in the head with its Rebrickable link and the button that puts
	it in or takes it out of the sort-out queue, and SetDetail under them.
	The page under it keeps its place; another set opened swaps it.
-->
<script lang="ts">
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Plus from '@lucide/svelte/icons/plus';
	import X from '@lucide/svelte/icons/x';
	import { Kind, type GetSetResponse } from '$lib/gen/pile/v1/pile_pb';
	import { setAnswers } from '$lib/api';
	import Sheet from '$lib/components/Sheet.svelte';
	import Button from '$lib/components/Button.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import { lotId, view } from '$lib/view.svelte';
	import { closeSet, openSet } from './open-set.svelte';
	import SetDetail from './SetDetail.svelte';

	const lot = $derived(lotId());
	const num = $derived(openSet());
	let detail = $state<GetSetResponse | null>(null);
	let error = $state<string | null>(null);

	$effect(() => {
		if (!num) return;
		const req = { view: view.forLot(lot), setNum: num };
		const had = setAnswers.peek(req);
		// A new set clears the last; the same set in a new view keeps it on
		// screen until the new one is in.
		if (had) detail = had;
		else if (detail?.set?.setNum !== num) detail = null;
		error = null;
		let stale = false;
		setAnswers.get(req).then(
			(d) => !stale && (detail = d),
			(e) => !stale && (error = String(e))
		);
		return () => (stale = true);
	});

	const set = $derived(detail?.set?.setNum === num ? detail.set : undefined);
</script>

<Sheet
	open={!!num}
	width="46rem"
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
				variant={detail?.sortOutPlace ? 'secondary' : 'primary'}
				icon={detail?.sortOutPlace ? X : Plus}
				onclick={() => view.toggle(lot, set.setNum)}
				>{detail?.sortOutPlace ? `Sorting out ${detail.sortOutPlace}: take out` : 'Sort this out'}</Button
			>
		{/if}
	{/snippet}
	{#if error}
		<div class="p-5"><Alert tone="danger" title="This set did not load">{error}</Alert></div>
	{:else if detail && set}
		<SetDetail {detail} />
	{:else}
		<div aria-busy="true" class="flex flex-col gap-3 p-5">
			<Skeleton class="h-28 w-full" /><Skeleton class="h-20 w-full" /><Skeleton class="h-96 w-full" />
		</div>
	{/if}
</Sheet>
