<!--
	In the top bar: what is updating, while an answer takes more than a
	moment (docs/loading-data.md). The Spinner and the sentence, or on a narrow bar
	the Spinner alone. It sits at the start of the bar's right side, so
	nothing on the page moves when it comes and goes.
-->
<script lang="ts">
	import Spinner from '$lib/components/Spinner.svelte';
	import { updating } from './query.svelte';

	const doing = $derived(updating());
</script>

{#if doing}
	<span class="flex items-center gap-2 text-sm whitespace-nowrap text-ink-muted" title={doing}>
		<Spinner size={12} /><span class="hidden xl:inline">{doing}</span>
	</span>
{/if}
