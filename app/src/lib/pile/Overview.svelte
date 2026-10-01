<!--
	The pile's totals in one row: how many pieces, how many lots, how much of it
	the sets explain, and what no set can claim.
-->
<script lang="ts">
	import type { GetOverviewResponse } from '$lib/gen/pile/v1/pile_pb';
	import Panel from '$lib/components/Panel.svelte';
	import Stat from '$lib/components/Stat.svelte';
	import { count, percent } from '$lib/format';

	let { overview }: { overview: GetOverviewResponse } = $props();
</script>

<Panel flush>
	<div class="grid grid-cols-2 divide-line md:grid-cols-4 md:divide-x">
		<Stat label="Pieces" value={count(overview.pieces)} />
		<Stat label="Lots" value={count(overview.lots)} hint="parts in a color" />
		<Stat
			label="Explained by sets"
			value={percent(overview.explainedPieces / Math.max(1, overview.pieces))}
			hint="{count(overview.explainedPieces)} pieces"
		/>
		<Stat
			label="Not in the catalog"
			value={count(overview.unmatchedPieces)}
			hint="no Rebrickable match"
		/>
	</div>
</Panel>
