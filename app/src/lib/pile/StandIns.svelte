<!--
	Make one line of a set from other parts (docs/changes.md): what could
	stand in for one of its part, each with how many the lots could make of
	what the sets before it in the queue and the set itself leave, checked
	to use it. For a plain plate, tile or brick, any smaller ones of its kind
	that fit (matching picks from what is there) and each way to cover its
	footprint; for any part, the part it is a print of, its prints, and the
	parts it is assembled from. Under them, a search to pick any parts, and
	how many of each make one. The line takes its own part first, then the
	ones checked, in the order they were checked; Save keeps them with the
	set.
-->
<script lang="ts">
	import { untrack } from 'svelte';
	import Plus from '@lucide/svelte/icons/plus';
	import X from '@lucide/svelte/icons/x';
	import { StandInSource, type SetLine, type StandInOption } from '$lib/gen/pile/v1/pile_pb';
	import { sources } from '$lib/api';
	import { query } from '$lib/data/query.svelte';
	import Loaded from '$lib/data/Loaded.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import Checkbox from '$lib/components/Checkbox.svelte';
	import Input from '$lib/components/Input.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import { count } from '$lib/format';
	import { lotsOf, view } from '$lib/view.svelte';
	import { changes } from './changes.svelte';

	let {
		line,
		setNum,
		stack,
		open = $bindable(false)
	}: {
		line: SetLine | null;
		setNum: string;
		// The route's lots, whose view counts them.
		stack: string;
		open?: boolean;
	} = $props();

	const own = $derived(line?.color?.id ?? 0);

	// A part of a stand-in as shown: its picture, name and how many.
	type Shown = {
		partNum: string;
		name: string;
		imageUrl: string;
		quantity: number;
		pieces?: number;
	};
	// The stand-ins checked, in order, as the set would keep them.
	type Chosen = { fits: boolean; parts: Shown[] };
	let chosen = $state<Chosen[]>([]);
	// Parts picked by hand for one more stand-in, and the search for them.
	let draft = $state<Shown[]>([]);
	let search = $state('');

	const keyOf = (s: { fits: boolean; parts: { partNum: string; quantity: number }[] }) =>
		s.fits
			? 'fits'
			: s.parts
					.map((p) => `${p.quantity} x ${p.partNum}`)
					.sort()
					.join(' + ');
	const shownOf = (o: StandInOption): Chosen => ({
		fits: !!o.standIn?.fits,
		parts: o.parts.map((p) => ({
			partNum: p.partNum,
			name: p.name,
			imageUrl: p.imageUrl,
			quantity: p.quantity,
			pieces: p.pieces
		}))
	});

	const answer = query(
		sources.standIns,
		() =>
			open && line
				? { view: view.of(stack), setNum, partNum: line.partNum, color: own }
				: undefined,
		(r) => `${r.setNum} ${r.partNum} ${r.color}`
	);

	// Opened: the stand-ins the line has now.
	$effect(() => {
		if (!open || !line) return;
		const l = line;
		untrack(() => {
			const swap = changes.find(setNum, l.partNum, own, 'swap');
			chosen =
				swap?.what.case === 'swap'
					? swap.what.value.standIns.map((si) => ({
							fits: si.fits,
							parts: si.parts.map((p) => ({
								partNum: p.partNum,
								name: p.partNum,
								imageUrl: '',
								quantity: p.quantity
							}))
						}))
					: [];
			draft = [];
			search = '';
		});
	});

	const keys = $derived(new Set(chosen.map(keyOf)));
	function toggle(s: Chosen, on: boolean) {
		chosen = on ? [...chosen, s] : chosen.filter((c) => keyOf(c) !== keyOf(s));
	}
	// Chosen here and not among the answer's ways (picked by hand just now).
	const extra = $derived(
		chosen.filter(
			(c) => !c.fits && !answer.data?.options.some((o) => keyOf(shownOf(o)) === keyOf(c))
		)
	);

	// The colors the line takes, to count the parts found in: none for any.
	const colors = $derived(
		!line || line.anyColor || view.colors === 'any'
			? []
			: line.useColors.length
				? line.useColors.map((c) => c.id)
				: [own]
	);
	const found = query(sources.findParts, () =>
		open && search.trim() ? { query: search, lots: lotsOf(stack), colors } : undefined
	);

	function pick(p: { partNum: string; name: string; imageUrl: string }) {
		const at = draft.findIndex((d) => d.partNum === p.partNum);
		draft =
			at < 0
				? [...draft, { ...p, quantity: 1 }]
				: draft.map((d, i) => (i === at ? { ...d, quantity: d.quantity + 1 } : d));
	}

	function addDraft() {
		const s = {
			fits: false,
			parts: draft.map((d) => ({ ...d, quantity: Math.max(1, Math.round(d.quantity) || 1) }))
		};
		if (!keys.has(keyOf(s))) chosen = [...chosen, s];
		draft = [];
		search = '';
	}

	// Keeps the stand-ins checked with the set; none takes its swap back.
	function save() {
		if (line) {
			const standIns = chosen.map((c) => ({
				fits: c.fits,
				parts: c.parts.map((p) => ({ partNum: p.partNum, quantity: p.quantity }))
			}));
			const swap = changes.find(setNum, line.partNum, own, 'swap');
			if (standIns.length)
				changes.make(setNum, {
					partNum: line.partNum,
					color: own,
					what: { case: 'swap', value: { standIns } }
				});
			else if (swap) changes.undo(setNum, swap);
		}
		open = false;
	}

	const from: Record<StandInSource, string> = {
		[StandInSource.UNSPECIFIED]: '',
		[StandInSource.FOOTPRINT]: 'covers its footprint',
		[StandInSource.UNPRINTED]: 'the part it is a print of',
		[StandInSource.PRINTED]: 'a print of it',
		[StandInSource.SUB_PARTS]: 'the parts it is assembled from',
		[StandInSource.PICKED]: 'picked by hand'
	};
	const recipe = (ps: { partNum: string; quantity: number }[]) =>
		ps.map((p) => `${p.quantity} × ${p.partNum}`).join(' + ');
</script>

{#snippet parts(ps: Shown[])}
	<span class="flex shrink-0 items-center gap-1">
		{#each ps as p (p.partNum)}
			<span
				class="relative"
				title="{p.quantity} × {p.name}, {p.partNum}{p.pieces !== undefined
					? `: ${p.pieces} in the lots`
					: ''}"
			>
				<PartImage src={p.imageUrl} class="size-9" />
				<span class="num absolute right-0 bottom-0 text-xs leading-none text-ink">{p.quantity}</span
				>
			</span>
		{/each}
	</span>
{/snippet}

{#snippet way(s: Chosen, source: string, makes: number | undefined)}
	<Checkbox checked={keys.has(keyOf(s))} onchange={(e) => toggle(s, e.currentTarget.checked)}>
		<span class="flex min-w-0 items-center gap-3">
			{@render parts(s.parts)}
			<span class="flex min-w-0 flex-col">
				<span class="truncate">{s.fits ? 'Any that fit' : recipe(s.parts)}</span>
				<span class="truncate text-xs text-ink-muted">
					{source}{#if makes !== undefined}{source ? ' · ' : ''}makes {count(makes)} of {line?.need}{/if}
					{#each s.parts.filter((p) => p.pieces !== undefined) as p (p.partNum)}
						· {count(p.pieces ?? 0)} × {p.partNum} in the lots
					{/each}
				</span>
			</span>
		</span>
	</Checkbox>
{/snippet}

<Modal bind:open title="{line?.name ?? 'A part'}: make it from other parts" size="lg">
	<p class="text-ink-muted">
		One of {line?.partNum} can be made of these instead, in the colors it takes: its own part is used
		first, then the ones checked, in the order checked. How many each makes is of what the sets before
		it in the queue and the rest of the set leave. Kept with the set, for building it later.
	</p>
	<Loaded query={answer} what="The parts to make it from">
		{#snippet waiting()}
			<div class="mt-4 flex flex-col gap-3">
				<Skeleton class="h-10 w-full" /><Skeleton class="h-10 w-full" />
			</div>
		{/snippet}
		{#snippet children(a)}
			<div class="mt-4 flex flex-col divide-y divide-line">
				{#if a.fits}
					<div class="py-2.5">
						{@render way(
							{ fits: true, parts: [] },
							'smaller ones of its kind and height, however they hold together; matching picks from what is there',
							a.fits.makes
						)}
					</div>
				{/if}
				{#each a.options as o (keyOf(shownOf(o)))}
					<div class="py-2.5">{@render way(shownOf(o), from[o.source], o.makes)}</div>
				{/each}
				{#each extra as s (keyOf(s))}
					<div class="py-2.5">{@render way(s, from[StandInSource.PICKED], undefined)}</div>
				{/each}
				{#if !a.fits && !a.options.length && !extra.length}
					<p class="py-2.5 text-ink-muted">
						Nothing the catalog relates to it is in the lots: pick parts below.
					</p>
				{/if}
			</div>
		{/snippet}
	</Loaded>

	<section class="mt-5 flex flex-col gap-3 border-t border-line pt-4">
		<div>
			<h3 class="font-semibold">Pick parts</h3>
			<p class="text-ink-muted">
				Find any parts, and say how many of each make one of {line?.partNum}.
			</p>
		</div>
		<Input
			bind:value={search}
			type="search"
			placeholder="A part's name or number"
			aria-label="Find parts"
			autocomplete="off"
			spellcheck="false"
		/>
		{#if search.trim() && found.data}
			<ul class="flex max-h-64 flex-col divide-y divide-line overflow-y-auto">
				{#each found.data.parts as p (p.partNum)}
					<li class="flex items-center gap-3 py-1.5">
						<PartImage src={p.imageUrl} class="size-9 shrink-0" />
						<span class="flex min-w-0 flex-1 flex-col">
							<span class="truncate">{p.name}</span>
							<span class="num text-xs text-ink-muted"
								>{p.partNum} · {count(p.count)} in the lots</span
							>
						</span>
						<Button size="sm" variant="secondary" icon={Plus} onclick={() => pick(p)}>Add</Button>
					</li>
				{:else}
					<li class="py-1.5 text-ink-muted">No part in the catalog is called that.</li>
				{/each}
			</ul>
		{/if}
		{#if draft.length}
			<div class="flex flex-col gap-2 rounded-control bg-well p-3">
				<span class="label">One of {line?.partNum} is made of</span>
				{#each draft as d, i (d.partNum)}
					<div class="flex items-center gap-3">
						<PartImage src={d.imageUrl} class="size-9 shrink-0" />
						<span class="min-w-0 flex-1 truncate"
							>{d.name} <span class="num text-xs text-ink-muted">{d.partNum}</span></span
						>
						<Input
							type="number"
							size="sm"
							class="w-24"
							unit="×"
							min="1"
							max="99"
							aria-label="How many of {d.partNum}"
							value={d.quantity}
							oninput={(e: Event & { currentTarget: HTMLInputElement }) =>
								(draft = draft.map((x, j) =>
									j === i ? { ...x, quantity: e.currentTarget.valueAsNumber } : x
								))}
						/>
						<Button
							size="sm"
							variant="ghost"
							icon={X}
							label="Take {d.partNum} out"
							onclick={() => (draft = draft.filter((x) => x.partNum !== d.partNum))}
						/>
					</div>
				{/each}
				<div>
					<Button size="sm" variant="secondary" icon={Plus} onclick={addDraft}
						>Use these as one</Button
					>
				</div>
			</div>
		{/if}
	</section>

	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
		<Button variant="primary" onclick={save}>Save</Button>
	{/snippet}
</Modal>
