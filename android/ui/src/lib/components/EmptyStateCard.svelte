<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		message,
		children,
		ariaBusy = false
	}: {
		message: string;
		children?: Snippet;
		ariaBusy?: boolean;
	} = $props();
</script>

<div
	class="card flex min-h-[7rem] flex-col items-center justify-center text-center text-sm"
	style:color="var(--text-muted)"
	aria-busy={ariaBusy || undefined}
>
	<p>{message}</p>
	{#if children}
		<div class="mt-4">{@render children()}</div>
	{/if}
</div>
