<!--
	The lot's sort-out queue: the sets and custom models to pull from the
	pile, most wanted first, each counted against what the ones before it
	left. Arrows reorder it; everything below on the page is matched against
	what is left after all of them.
-->
<script lang="ts">
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import X from '@lucide/svelte/icons/x';
	import type { SetMatch } from '$lib/gen/pile/v1/pile_pb';
	import Panel from '$lib/components/Panel.svelte';
	import Button from '$lib/components/Button.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import { count, percent } from '$lib/format';
	import { view } from '$lib/view.svelte';

	let {
		queue,
		lot,
		pieces,
		left
	}: { queue: SetMatch[]; lot: string; pieces: number; left: number } = $props();
</script>

<Panel
	title="Sorting out · {queue.length}"
	description="Most wanted first. Each takes its pieces from what the ones before it left; everything below is matched against the {count(left)} of {count(pieces)} pieces still in the pile."
	flush
>
	{#snippet actions()}
		<Button size="sm" variant="ghost" icon={X} onclick={() => view.clear(lot)}>Clear</Button>
	{/snippet}
	<div class="flex gap-2 overflow-x-auto p-2">
		{#each queue as set, i (set.setNum)}
			{@const done = set.have >= set.need}
			<div class="flex w-36 shrink-0 flex-col gap-1.5 rounded-control bg-well p-2">
				<div class="flex items-center justify-between">
					<span class="num text-xs font-medium text-primary-ink">{i + 1}</span>
					<div class="flex">
						<Button
							size="sm"
							variant="ghost"
							icon={ArrowLeft}
							label="Sort out sooner"
							disabled={i === 0}
							onclick={() => view.move(lot, set.setNum, -1)}
						/>
						<Button
							size="sm"
							variant="ghost"
							icon={ArrowRight}
							label="Sort out later"
							disabled={i === queue.length - 1}
							onclick={() => view.move(lot, set.setNum, 1)}
						/>
						<Button
							size="sm"
							variant="ghost"
							icon={X}
							label="Take out of the queue"
							onclick={() => view.toggle(lot, set.setNum)}
						/>
					</div>
				</div>
				<a href="/lots/{lot}/sets/{set.setNum}" class="flex flex-col gap-1.5">
					<PartImage src={set.imageUrl} class="aspect-square w-full" />
					<div class="truncate text-xs font-medium text-ink" title={set.name}>{set.name}</div>
				</a>
				<ProgressBar
					value={set.have}
					max={set.need}
					tone={done ? 'success' : 'primary'}
					label="{set.name}: {percent(set.have / Math.max(1, set.need))} complete"
				/>
				<div class="num text-xs {done ? 'text-success-ink' : 'text-ink-muted'}">
					{set.have}/{set.need}{#if set.wrongColor}<span class="text-warning-ink">
							· {set.wrongColor} recolor</span
						>{/if}
				</div>
			</div>
		{/each}
	</div>
</Panel>
