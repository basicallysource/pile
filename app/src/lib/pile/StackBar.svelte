<!--
	The lots a page looks at, at the top of every page of them (docs/views.md):
	a trail back to the collection, then each lot in priority order, numbered,
	with its pieces; a set's pieces are drawn from the first lot that has them.
	With several, each moves earlier or later and comes out; "Add a lot" adds
	any other, the unlisted ones included (they are added only from here).
	A change goes to the same page for the new lots, in place: the page keeps
	what it shows until the new numbers are in. It is as tall with one lot as
	with several, so adding the second moves nothing below it.
-->
<script lang="ts">
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Plus from '@lucide/svelte/icons/plus';
	import X from '@lucide/svelte/icons/x';
	import { goto } from '$app/navigation';
	import { sources } from '$lib/api';
	import { query } from '$lib/data/query.svelte';
	import Button from '$lib/components/Button.svelte';
	import Menu from '$lib/components/Menu.svelte';
	import { count } from '$lib/format';
	import { lotsOf, routeStack, stackHref } from '$lib/view.svelte';

	const collection = query(sources.lots, () => ({}));
	const all = $derived(collection.data?.lots ?? []);
	const lots = $derived(lotsOf(routeStack()));
	const others = $derived(all.filter((l) => !lots.includes(l.id)));
	const named = (id: string) => all.find((l) => l.id === id);

	function change(next: string[]) {
		goto(stackHref(next), { noScroll: true, keepFocus: true });
	}

	function move(i: number, step: -1 | 1) {
		const next = [...lots];
		[next[i], next[i + step]] = [next[i + step], next[i]];
		change(next);
	}
</script>

<nav
	aria-label="Lots"
	class="flex min-h-8 flex-wrap items-center gap-x-1 gap-y-2 text-sm text-ink-muted"
>
	<a href="/" class="hover:text-ink">Collection</a>
	<ChevronRight size={14} />
	{#each lots as id, i (id)}
		{@const l = named(id)}
		<div
			class="flex items-center gap-2 rounded-control {lots.length > 1
				? 'bg-surface py-0.5 pr-0.5 pl-2.5'
				: ''}"
		>
			{#if lots.length > 1}<span class="num text-xs font-medium text-primary-ink">{i + 1}</span>{/if}
			<span class="font-medium text-ink">{l?.name ?? id}</span>
			{#if l?.totals}<span class="num text-xs text-ink-faint">{count(l.totals.pieces)}</span>{/if}
			{#if lots.length > 1}
				<div class="flex">
					<Button
						size="sm"
						variant="ghost"
						icon={ArrowLeft}
						label="Draw from {l?.name ?? id} sooner"
						disabled={i === 0}
						onclick={() => move(i, -1)}
					/>
					<Button
						size="sm"
						variant="ghost"
						icon={ArrowRight}
						label="Draw from {l?.name ?? id} later"
						disabled={i === lots.length - 1}
						onclick={() => move(i, 1)}
					/>
					<Button
						size="sm"
						variant="ghost"
						icon={X}
						label="Take {l?.name ?? id} out"
						onclick={() => change(lots.filter((x) => x !== id))}
					/>
				</div>
			{/if}
		</div>
	{/each}
	{#if others.length}
		<Menu
			label="Add a lot"
			placement="bottom-start"
			width="18rem"
			items={others.map((l) => ({
				label: l.name,
				hint: count(l.totals?.pieces ?? 0),
				onselect: () => change([...lots, l.id])
			}))}
		>
			{#snippet trigger(props)}
				<Button {...props} size="sm" variant="ghost" icon={Plus}>Add a lot</Button>
			{/snippet}
		</Menu>
	{/if}
</nav>
