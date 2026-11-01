<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		actions
	}: {
		title: string;
		actions?: Snippet;
	} = $props();
</script>

<div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-4">
	<h1 class="min-w-0 flex-[1_1_10rem] text-2xl font-semibold tracking-tight text-balance">
		{title}
	</h1>
	{#if actions}
		<div class="flex shrink-0 flex-wrap items-center gap-2">
			{@render actions()}
		</div>
	{/if}
</div>
