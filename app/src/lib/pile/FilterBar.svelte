<!--
	The filter: which sets and custom models the overview and Every set list
	(docs/views.md#the-filter), the same on both and kept in this browser. A
	preset first (Interesting, the default, hides loose bricks and basic
	sets; Models hides only loose bricks; Everything hides nothing; none is
	marked when the rules are the reader's own), then the rules it sets,
	each its own switch with what it hides, then a theme, a
	size, the kind and licensed only, and whether sets are grouped by theme.
	It sits in a flush Panel's head, as a row that wraps.
-->
<script lang="ts">
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import { Kind, type ThemeCount } from '$lib/gen/pile/v1/pile_pb';
	import Button from '$lib/components/Button.svelte';
	import Popover from '$lib/components/Popover.svelte';
	import SegmentedControl from '$lib/components/SegmentedControl.svelte';
	import Select from '$lib/components/Select.svelte';
	import Switch from '$lib/components/Switch.svelte';
	import { count } from '$lib/format';
	import { presets, view, type Filter, type Preset } from '$lib/view.svelte';
	import { rules } from './rules';

	let { themes }: { themes: ThemeCount[] } = $props();

	const f = $derived(view.filter);
	const hiding = $derived(rules.filter((r) => f[r.key]).length);
	const kinds = { all: Kind.UNSPECIFIED, sets: Kind.SET, custom: Kind.CUSTOM } as const;
	const kind = $derived(
		(Object.keys(kinds) as (keyof typeof kinds)[]).find((k) => kinds[k] === f.kind) ?? 'all'
	);
</script>

<div class="flex flex-wrap items-center gap-3">
	<SegmentedControl
		label="Show"
		size="sm"
		value={view.preset() ?? ''}
		onchange={(p) => view.setFilter(presets[p as Preset])}
		options={[
			{ value: 'interesting', label: 'Interesting' },
			{ value: 'models', label: 'Models' },
			{ value: 'everything', label: 'Everything' }
		]}
	/>
	<Popover label="What to hide" width="24rem">
		{#snippet trigger(props)}
			<Button {...props} size="sm" variant="ghost" icon={SlidersHorizontal}
				>Hiding {hiding} of {rules.length}</Button
			>
		{/snippet}
		<ul class="flex flex-col gap-3">
			{#each rules as r (r.key)}
				<li class="flex items-start justify-between gap-4">
					<div class="min-w-0">
						<p class="text-sm font-medium text-ink">{r.name}</p>
						<p class="text-xs text-ink-muted">{r.hides}</p>
					</div>
					<Switch
						label="Hide {r.name.toLowerCase()}"
						checked={f[r.key]}
						onchange={(v) => view.setFilter({ [r.key]: v } as Partial<Filter>)}
					/>
				</li>
			{/each}
		</ul>
	</Popover>
	<Select
		label="Theme"
		size="sm"
		class="w-52"
		value={f.themeGroup}
		onchange={(v) => view.setFilter({ themeGroup: v })}
		options={[
			{ value: '', label: 'Every theme' },
			...themes.map((t) => ({ value: t.name, label: t.name || 'No theme', hint: count(t.sets) })),
			...(f.themeGroup && !themes.some((t) => t.name === f.themeGroup)
				? [{ value: f.themeGroup, label: f.themeGroup, hint: '0' }]
				: [])
		]}
	/>
	<Select
		label="Size"
		size="sm"
		class="w-36"
		value={String(f.minPieces)}
		onchange={(v) => view.setFilter({ minPieces: +v })}
		options={[
			{ value: '0', label: 'Any size' },
			{ value: '50', label: '50+ pieces' },
			{ value: '100', label: '100+ pieces' },
			{ value: '250', label: '250+ pieces' },
			{ value: '500', label: '500+ pieces' },
			{ value: '1000', label: '1,000+ pieces' }
		]}
	/>
	<SegmentedControl
		label="Kind"
		size="sm"
		value={kind}
		onchange={(k) => view.setFilter({ kind: kinds[k] })}
		options={[
			{ value: 'all', label: 'All' },
			{ value: 'sets', label: 'Sets' },
			{ value: 'custom', label: 'Custom models' }
		]}
	/>
	<label class="flex items-center gap-2 text-sm whitespace-nowrap text-ink">
		<Switch checked={f.licensedOnly} onchange={(v) => view.setFilter({ licensedOnly: v })} />
		Licensed only
	</label>
	<label class="flex items-center gap-2 text-sm whitespace-nowrap text-ink">
		<Switch checked={view.byTheme} onchange={(v) => view.setByTheme(v)} />
		Group by theme
	</label>
</div>
