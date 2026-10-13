<!--
	Time to collect: how long one sorter, running nonstop and fed bulk mixed
	like a lot, takes to come across every piece of a set, in exact colors,
	near colors (the old and new grays and browns as one) and any color. The
	mix is one lot's: with several, a switch picks which (the first, until
	another is picked; kept for the visit). The server works it out from the
	lot's data at start, a set at a time, so the page asks again every few
	seconds while anything is on its way (its source is not settled until
	then: docs/loading-data.md). A set added here is timed too and remembered
	in this browser.
-->
<script lang="ts" module>
	// The lot timed, for each stack that picked one.
	const timed = new Map<string, string>();
</script>

<script lang="ts">
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Plus from '@lucide/svelte/icons/plus';
	import type { CollectTime, SetCollectTimes } from '$lib/gen/pile/v1/pile_pb';
	import { sources } from '$lib/api';
	import { query } from '$lib/data/query.svelte';
	import Loaded from '$lib/data/Loaded.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import Panel from '$lib/components/Panel.svelte';
	import Input from '$lib/components/Input.svelte';
	import Button from '$lib/components/Button.svelte';
	import Skeleton from '$lib/components/Skeleton.svelte';
	import Spinner from '$lib/components/Spinner.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import { zoomsTo } from '$lib/pile/zoom.svelte';
	import { count, dayOf, dayTimeOf } from '$lib/format';
	import { span, hoursOf } from '$lib/pile/hours';
	import SegmentedControl from '$lib/components/SegmentedControl.svelte';
	import { lotsOf, routeStack } from '$lib/view.svelte';

	const KEY = 'pile-collect-extra';
	function loadExtra(): string[] {
		try {
			const v = JSON.parse(localStorage.getItem(KEY) ?? '[]');
			return Array.isArray(v) ? v.filter((x) => typeof x === 'string') : [];
		} catch {
			return [];
		}
	}

	const stack = $derived(routeStack());
	const lots = $derived(lotsOf(stack));
	let picked = $state(timed.get(routeStack()));
	const lot = $derived(picked && lots.includes(picked) ? picked : lots[0]);
	$effect(() => {
		if (picked) timed.set(stack, picked);
	});
	const collection = query(sources.lots, () => ({}));
	let extra = $state<string[]>(loadExtra());
	let adding = $state('');
	let open = $state<string | null>(null);

	const times = query(sources.collectTimes, () => ({ lotId: lot, extra: [...extra] }));
	const data = $derived(times.data);
	const working = $derived(!data || !sources.collectTimes.final(data));

	function add(event: SubmitEvent) {
		event.preventDefault();
		let num = adding.trim();
		if (!num) return;
		if (!num.includes('-')) num += '-1';
		if (!extra.includes(num)) {
			extra = [...extra, num];
			try {
				localStorage.setItem(KEY, JSON.stringify(extra));
			} catch {
				// Storage can be off; the set lasts the visit.
			}
		}
		adding = '';
		open = num;
	}

	const modes = [
		{ key: 'exact', label: 'Exact colors' },
		{ key: 'near', label: 'Near colors' },
		{ key: 'any', label: 'Any color' }
	] as const;

	function leftOut(s: SetCollectTimes): string {
		const parts = Object.entries(s.leftOut)
			.filter(([, n]) => n > 0)
			.map(([why, n]) => `${count(n)} ${why}`);
		if (s.minifigures) parts.push(`${s.minifigures} minifigures`);
		return parts.join(', ');
	}
</script>

{#snippet cell(t: CollectTime | undefined)}
	{#if t}
		<td class="num" title="90%: {hoursOf(t.most)}">{span(t.most)}</td>
		<td class="num font-medium" title="Every piece: {hoursOf(t.all)}">{span(t.all)}</td>
	{:else}
		<td class="num text-ink-faint">…</td>
		<td class="num text-ink-faint">…</td>
	{/if}
{/snippet}

{#snippet detail(s: SetCollectTimes)}
	<div class="flex flex-col gap-4 bg-well p-(--pad-panel)">
		<p class="text-sm text-ink-muted">
			{count(s.pieces)} pieces counted{leftOut(s) ? `; left out: ${leftOut(s)}` : ''}.
		</p>
		<div class="grid gap-4 md:grid-cols-3">
			{#each modes as m (m.key)}
				{@const t = s[m.key]}
				{#if t}
					<div class="flex flex-col gap-2 text-sm">
						<div class="label">{m.label}</div>
						<dl class="num grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
							<dt class="text-ink-muted">Half</dt><dd title={hoursOf(t.half)}>{span(t.half)}</dd>
							<dt class="text-ink-muted">90%</dt>
							<dd title={hoursOf(t.most)}>{span(t.most)} <span class="text-ink-faint">({span(t.mostLow)} to {span(t.mostHigh)})</span></dd>
							<dt class="text-ink-muted">99%</dt><dd title={hoursOf(t.nearly)}>{span(t.nearly)}</dd>
							<dt class="text-ink-muted">Every piece</dt>
							<dd title={hoursOf(t.all)}>{span(t.all)} <span class="text-ink-faint">(one run in ten {span(t.allFast)} or less, one in ten {span(t.allSlow)} or more)</span></dd>
							<dt class="text-ink-muted">Seen parts only</dt><dd title={hoursOf(t.allSeen)}>{span(t.allSeen)}</dd>
							<dt class="text-ink-muted">Never seen</dt><dd>{t.unseenKeys} of {t.keys} part-colors, {count(t.unseenPieces)} pieces</dd>
						</dl>
						<div class="label mt-1">Slowest pieces</div>
						<ul class="flex flex-col gap-1">
							{#each t.slowest.slice(0, 4) as p (p.partNum + (p.color?.id ?? ''))}
								<li class="flex items-center gap-2">
									<PartImage src={p.imageUrl} onzoom={zoomsTo(`${p.name}${p.color ? `, ${p.color.name}` : ''}`)} class="size-8 shrink-0" />
									<span class="min-w-0 flex-1 truncate" title="{p.name}{p.color ? `, ${p.color.name}` : ''}">
										{p.need} × {p.name}{p.color ? `, ${p.color.name}` : ''}
									</span>
									<span class="num shrink-0 text-ink-faint" title="{p.seen} seen; {hoursOf(p.waitHours)}">{span(p.waitHours)}</span>
								</li>
							{/each}
						</ul>
					</div>
				{/if}
			{/each}
		</div>
	</div>
{/snippet}

<PageHeader
	title="Time to collect"
	description="How long one sorter, running nonstop and fed bulk mixed like this lot, takes to come across every piece of a set."
>
	{#if lots.length > 1}
		<div class="mb-2">
			<SegmentedControl
				label="Lot timed"
				size="sm"
				value={lot}
				onchange={(v) => (picked = v)}
				options={lots.map((id) => ({
					value: id,
					label: collection.data?.lots.find((l) => l.id === id)?.name ?? id
				}))}
			/>
		</div>
	{/if}
	{#snippet actions()}
		<form class="flex items-center gap-2" onsubmit={add}>
			<Input bind:value={adding} placeholder="Set number" aria-label="Set number to add" class="w-36" />
			<Button type="submit" icon={Plus}>Add</Button>
		</form>
	{/snippet}
	{#if data?.basis}
		{@const b = data.basis}
		<p class="text-xs text-ink-faint">
			From {count(b.pieces)} pieces sorted {dayOf(b.firstSeenUnix)} to {dayOf(b.lastSeenUnix)}{b.machineName
				? ` on ${b.machineName}`
				: ''}: {count(Math.round(b.sortingHours))} hours of sorting, {count(Math.round(b.ratePerHour))} pieces an
			hour. Worked out {dayTimeOf(data.fittedUnix)}. Printed parts, minifigures, specialty molds and parts too big
			for a sorter are left out; near colors count Light Gray with Light Bluish Gray, Dark Gray with Dark Bluish
			Gray, Brown with Reddish Brown, and Pearl Light Gray with Flat Silver.
		</p>
	{/if}
</PageHeader>

<Loaded query={times} what="The times" class="flex flex-col gap-(--gap-panels)">
	{#snippet waiting()}<Skeleton class="h-96 w-full" />{/snippet}
	{#snippet children(data)}
		{#if data.fittedUnix === 0n}
			<Panel>
				<div class="flex items-center gap-3 text-sm text-ink-muted">
					<Spinner /> Working out this lot's mix of pieces. The times follow in a minute or two.
				</div>
			</Panel>
		{:else}
			<Panel flush>
				<div class="overflow-x-auto">
					<table class="data-table">
						<thead>
							<tr>
								<th rowspan="2">Set</th>
								<th rowspan="2" class="num">Pieces</th>
								{#each modes as m (m.key)}<th colspan="2" class="text-center">{m.label}</th>{/each}
							</tr>
							<tr>
								{#each modes as m (m.key)}<th class="num">90%</th><th class="num">Every piece</th>{/each}
							</tr>
						</thead>
						<tbody>
							{#each data.sets as s (s.setNum)}
								<tr class="is-link" onclick={() => (open = open === s.setNum ? null : s.setNum)}>
									<td>
										<div class="flex items-center gap-3">
											<ChevronRight size={14} class="shrink-0 text-ink-faint transition-transform {open === s.setNum ? 'rotate-90' : ''}" />
											<PartImage src={s.imageUrl} onzoom={zoomsTo(`${s.name}, ${s.setNum}`)} class="size-10 shrink-0" />
											<div class="min-w-0">
												<div class="truncate">{s.name || s.setNum}</div>
												<div class="text-xs text-ink-faint">{s.setNum}{s.year ? ` · ${s.year}` : ''}{s.theme ? ` · ${s.theme}` : ''}</div>
											</div>
										</div>
									</td>
									{#if s.unknown}
										<td colspan="7" class="text-ink-muted">No such set with counted pieces in the catalog.</td>
									{:else if s.pending}
										<td class="num text-ink-faint">…</td>
										<td colspan="6" class="text-ink-faint"><span class="inline-flex items-center gap-2"><Spinner size={14} /> On its way</span></td>
									{:else}
										<td class="num">{count(s.pieces)}</td>
										{@render cell(s.exact)}
										{@render cell(s.near)}
										{@render cell(s.any)}
									{/if}
								</tr>
								{#if open === s.setNum && !s.pending && !s.unknown}
									<tr><td colspan="8" class="p-0">{@render detail(s)}</td></tr>
								{/if}
							{/each}
						</tbody>
					</table>
				</div>
			</Panel>
			<p class="text-xs text-ink-faint">
				90% is when nine in ten of the set's pieces are expected to have come through; every piece is the median time to
				the last one, and hangs on its rarest pieces, often ones these records never had, whose share is estimated from
				how much LEGO used them. Open a set for the ranges and the pieces that hold it up.{working ? ' Still working out the rest.' : ''}
			</p>
		{/if}
	{/snippet}
</Loaded>
