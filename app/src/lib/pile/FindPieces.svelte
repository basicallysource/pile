<!--
	Find pieces: what the bins hold, found as you type. A search field and
	filters (color, type of part, bin, lot) over the list of each part in a
	color found: its picture, name, color and number, how many there are, and
	a badge for each bin they are in with how many are there. Each filter
	offers only what the search and the other filters leave, with its pieces.

	Each keystroke asks again; the last answer stays shown until the new one
	is in, and an answer had this visit shows at once while it is asked for
	again (the bins can change while pile runs: docs/loading-data.md).

	It fills the screen on a phone and floats over the page on a wider one.
	Scrolling the list folds the filters away, leaving the field and a line
	saying what is chosen; back at the top they unfold. Escape and the close
	button close it; Enter in the field puts a phone's keyboard away.
-->
<script lang="ts">
	import X from '@lucide/svelte/icons/x';
	import type { FilterOption } from '$lib/gen/pile/v1/pile_pb';
	import { sources } from '$lib/api';
	import { query as ask } from '$lib/data/query.svelte';
	import Loaded from '$lib/data/Loaded.svelte';
	import { count } from '$lib/format';
	import Input from '$lib/components/Input.svelte';
	import Select from '$lib/components/Select.svelte';
	import Button from '$lib/components/Button.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import ColorChip from '$lib/components/ColorChip.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	let dialog: HTMLDialogElement;
	let field = $state<HTMLInputElement>();
	let query = $state('');
	let color = $state('');
	let type = $state('');
	let bin = $state('');
	let lot = $state('');
	let folded = $state(false);

	$effect(() => {
		if (open && !dialog.open) {
			dialog.showModal();
			field?.focus();
		}
		if (!open && dialog.open) dialog.close();
	});

	const answer = ask(sources.pieces, () =>
		open ? { query, color, type, bin, lotId: lot } : undefined
	);
	const found = $derived(answer.data);

	// A filter's choices: any, then what is left with its pieces, and the
	// chosen one even when nothing is left of it.
	function choices(list: FilterOption[] | undefined, chosen: string, any: string) {
		const out = [
			{ value: '', label: any },
			...(list ?? []).map((o) => ({
				value: o.value,
				label: o.label || o.value,
				hint: count(o.pieces)
			}))
		];
		if (chosen && !out.some((o) => o.value === chosen))
			out.push({ value: chosen, label: chosen, hint: '0' });
		return out;
	}

	const chosen = $derived(
		[color, type, bin, lot && (found?.lots.find((o) => o.value === lot)?.label ?? lot)].filter(
			Boolean
		)
	);

	function onscroll(event: Event) {
		const top = (event.currentTarget as HTMLElement).scrollTop;
		if (top > 24) folded = true;
		else if (top < 4) folded = false;
	}

	function clear() {
		query = color = type = bin = lot = '';
		field?.focus();
	}
</script>

<dialog
	bind:this={dialog}
	aria-label="Find pieces"
	onclose={() => (open = false)}
	class="m-0 h-dvh max-h-none w-screen max-w-none flex-col overflow-hidden bg-raised p-0 text-ink outline-none open:flex sm:m-auto sm:h-[85dvh] sm:w-[calc(100vw-2rem)] sm:max-w-3xl sm:rounded-panel sm:border sm:border-line"
>
	<header class="flex shrink-0 flex-col gap-3 border-b border-line px-4 pt-3 pb-3 sm:px-5">
		<div class="flex items-center gap-2">
			<Input
				bind:element={field}
				bind:value={query}
				type="search"
				size="lg"
				class="flex-1"
				placeholder="A part, a color, a bin"
				aria-label="Search the bins"
				autocomplete="off"
				spellcheck="false"
				enterkeyhint="search"
				onkeydown={(e: KeyboardEvent) => e.key === 'Enter' && field?.blur()}
			/>
			<Button variant="ghost" icon={X} label="Close" onclick={() => dialog.close()} />
		</div>
		{#if !folded}
			<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
				<Select
					bind:value={color}
					label="Color"
					size="sm"
					options={choices(found?.colors, color, 'Any color')}
				/>
				<Select
					bind:value={type}
					label="Type"
					size="sm"
					options={choices(found?.types, type, 'Any type')}
				/>
				<Select
					bind:value={bin}
					label="Bin"
					size="sm"
					options={choices(found?.bins, bin, 'Any bin')}
				/>
				<Select
					bind:value={lot}
					label="Lot"
					size="sm"
					options={choices(found?.lots, lot, 'Any lot')}
				/>
			</div>
		{/if}
		<div class="flex min-h-5 items-center justify-between gap-3 text-xs text-ink-muted">
			<p class="min-w-0 truncate">
				{#if found}
					<span class="font-medium text-ink tabular-nums">{count(found.totalPieces)}</span> pieces ·
					{count(found.totalParts)} parts in a color{#if folded && chosen.length}&ensp;·&ensp;{chosen.join(
							', '
						)}{/if}
				{/if}
			</p>
			{#if query || chosen.length}
				<button type="button" class="shrink-0 text-primary-ink hover:underline" onclick={clear}
					>Clear</button
				>
			{/if}
		</div>
	</header>

	<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain" {onscroll}>
		<Loaded query={answer} what="The search">
			{#snippet waiting()}
				<div class="flex flex-col">
					{#each { length: 6 } as _}
						<div class="flex gap-3 border-b border-line px-4 py-3 sm:px-5">
							<Skeleton class="size-14 shrink-0" />
							<div class="flex flex-1 flex-col gap-2"><Skeleton class="h-4 w-2/3" /><Skeleton class="h-3 w-1/3" /></div>
						</div>
					{/each}
				</div>
			{/snippet}
			{#snippet children(found)}
				{#if found.storedPieces === 0}
					<EmptyState title="No bins yet.">
						Once bins are set out, each with an address and what it holds, their pieces can be found
						here.
					</EmptyState>
				{:else if found.parts.length === 0}
					<EmptyState title="Nothing in the bins matches." />
				{:else}
					<ul>
						{#each found.parts as p (p.partNum + '/' + (p.color?.id ?? ''))}
							<li class="flex gap-3 border-b border-line px-4 py-3 sm:px-5">
								<PartImage src={p.imageUrl} class="size-14 shrink-0" />
								<div class="min-w-0 flex-1">
									<div class="flex items-start justify-between gap-3">
										<div class="min-w-0">
											<p class="text-sm leading-snug font-medium">{p.name}</p>
											<p class="mt-0.5 flex flex-wrap items-center gap-x-2 text-xs text-ink-muted">
												<ColorChip name={p.color?.name} rgb={p.color?.rgb} />
												<span class="font-mono">{p.bricklinkId || p.partNum}</span>
											</p>
										</div>
										<p class="shrink-0 text-lg leading-none font-semibold tabular-nums">
											{count(p.count)}
										</p>
									</div>
									<ul class="mt-2 flex flex-wrap gap-1.5" aria-label="Bins">
										{#each p.bins as b (b.address)}
											<li title={b.name}>
												<Badge tone="primary"
													><span class="font-mono font-semibold">{b.address}</span><span
														class="tabular-nums">× {count(b.count)}</span
													></Badge
												>
											</li>
										{/each}
									</ul>
								</div>
							</li>
						{/each}
					</ul>
					{#if found.totalParts > found.parts.length}
						<p class="px-4 py-4 text-center text-xs text-ink-faint">
							The first {count(found.parts.length)} of {count(found.totalParts)}. Search or filter to
							narrow them down.
						</p>
					{/if}
				{/if}
			{/snippet}
		</Loaded>
	</div>
</dialog>
