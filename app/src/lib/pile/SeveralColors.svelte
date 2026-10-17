<!--
	Several colors for one line of a set (docs/changes.md): a check for each
	color the lots have its part in, with how many, and its own color; its
	pieces are then filled from any mix of the colors checked, the first in
	the list counting as its color and the rest as another color. It opens
	on what the line takes now, and keeps the choice with the set.
-->
<script lang="ts">
	import { untrack } from 'svelte';
	import type { ColorTake, SetLine } from '$lib/gen/pile/v1/pile_pb';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import Checkbox from '$lib/components/Checkbox.svelte';
	import ColorChip from '$lib/components/ColorChip.svelte';
	import { count } from '$lib/format';
	import { changes } from './changes.svelte';

	let {
		line,
		setNum,
		open = $bindable(false)
	}: {
		line: SetLine | null;
		setNum: string;
		open?: boolean;
	} = $props();

	const own = $derived(line?.color?.id ?? 0);
	// Its own color first, then the lots' colors of the part, the most
	// pieces first, then any it takes that the lots lack.
	const options = $derived.by(() => {
		if (!line) return [];
		const out = [
			{
				id: own,
				name: line.color?.name ?? '',
				rgb: line.color?.rgb ?? '',
				pieces: line.partColors.find((c) => c.color?.id === own)?.pieces ?? 0
			},
			...line.partColors
				.filter((c) => c.color!.id !== own)
				.map((c) => ({ id: c.color!.id, name: c.color!.name, rgb: c.color!.rgb, pieces: c.pieces }))
		];
		for (const c of line.useColors)
			if (!out.some((o) => o.id === c.id))
				out.push({ id: c.id, name: c.name, rgb: c.rgb, pieces: 0 });
		return out;
	});

	let checked = $state<number[]>([]);

	// Opened: what the line takes now, else its own color.
	$effect(() => {
		if (!open || !line) return;
		const l = line;
		untrack(() => {
			const take = (
				changes.find(setNum, l.partNum, own, 'colors') ?? changes.find(setNum, '', own, 'colors')
			)?.what.value as ColorTake | undefined;
			checked = take?.colors.length ? [...take.colors] : [own];
		});
	});

	function toggle(id: number, on: boolean) {
		checked = on ? [...checked, id] : checked.filter((c) => c !== id);
	}

	function save() {
		if (line)
			changes.recolor(
				setNum,
				line.partNum,
				own,
				options.filter((o) => checked.includes(o.id)).map((o) => o.id)
			);
		open = false;
	}
</script>

<Modal bind:open title="{line?.name ?? 'A part'}: several colors" size="sm">
	<p class="text-ink-muted">
		Its {line?.need ?? ''} pieces are filled from any mix of the colors checked, its own first. The count
		is of the pieces the lots have.
	</p>
	<div class="mt-4 flex flex-col gap-3">
		{#each options as o (o.id)}
			<Checkbox
				checked={checked.includes(o.id)}
				onchange={(e) => toggle(o.id, e.currentTarget.checked)}
			>
				<span class="flex items-center gap-2">
					<ColorChip name={o.name} rgb={o.rgb} />
					{#if o.id === own}<span class="text-ink-muted">its own</span>{/if}
					<span class="num text-xs text-ink-faint">{count(o.pieces)}</span>
				</span>
			</Checkbox>
		{/each}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
		<Button variant="primary" disabled={!checked.length} onclick={save}>Take these</Button>
	{/snippet}
</Modal>
