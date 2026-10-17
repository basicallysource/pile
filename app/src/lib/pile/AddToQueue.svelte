<!--
	Add a set or build to the sort-out queue: a search field that finds sets,
	custom models and builds within sets (a building, a vehicle) by name or
	number as one types, a build by its set's name or number too, and lists
	them under it, the number first, then whole words, sets before builds
	before custom models. A pick goes to the end of the queue, to be dragged
	to its place; Enter takes the first. One already in the queue says where.
-->
<script lang="ts">
	import { Kind, type FoundSet } from '$lib/gen/pile/v1/pile_pb';
	import { sources } from '$lib/api';
	import { query } from '$lib/data/query.svelte';
	import Input from '$lib/components/Input.svelte';
	import PartImage from '$lib/components/PartImage.svelte';
	import { count } from '$lib/format';
	import { view } from '$lib/view.svelte';

	let search = $state('');
	const found = query(sources.findSets, () => (search.trim() ? { query: search } : undefined));
	const sets = $derived(search.trim() ? (found.data?.sets ?? []) : []);

	const what = (f: FoundSet) =>
		[
			f.kind === Kind.BUILD
				? `a build in ${f.ofName}, ${f.ofSetNum}`
				: f.kind === Kind.CUSTOM
					? `${f.collection || 'custom model'}${f.designer ? ` by ${f.designer}` : ''}`
					: [f.setNum, f.year, f.theme].filter(Boolean).join(' · '),
			`${count(f.pieces)} pieces`
		].join(' · ');

	function add(num: string) {
		view.add(num);
		search = '';
	}

	function onkeydown(e: KeyboardEvent) {
		const first = sets.find((f) => !view.placeOf(f.setNum));
		if (e.key === 'Enter' && first) {
			e.preventDefault();
			add(first.setNum);
		}
	}
</script>

<div class="flex flex-col gap-1">
	<Input
		bind:value={search}
		type="search"
		placeholder="Add a set or build: its name or number"
		aria-label="Add a set or build to sort out"
		autocomplete="off"
		spellcheck="false"
		enterkeyhint="done"
		{onkeydown}
	/>
	{#if search.trim() && found.data}
		<ul class="flex max-h-80 flex-col overflow-y-auto" aria-label="Sets found">
			{#each sets as f (f.setNum)}
				{@const place = view.placeOf(f.setNum)}
				<li>
					<button
						type="button"
						disabled={place > 0}
						onclick={() => add(f.setNum)}
						class="flex w-full items-center gap-3 rounded-item px-2 py-1.5 text-left transition-colors hover:bg-hover active:bg-pressed disabled:pointer-events-none"
					>
						<PartImage src={f.imageUrl} class="size-10 shrink-0" />
						<span class="flex min-w-0 flex-1 flex-col">
							<span class="truncate text-sm text-ink">{f.name}</span>
							<span class="truncate text-xs text-ink-muted">{what(f)}</span>
						</span>
						<span class="shrink-0 text-xs {place ? 'text-ink-faint' : 'text-primary-ink'}"
							>{place ? `Sorting out ${place}` : 'Add'}</span
						>
					</button>
				</li>
			{:else}
				<li class="px-2 py-1.5 text-sm text-ink-muted">
					No set, build or custom model is called that.
				</li>
			{/each}
		</ul>
	{/if}
</div>
