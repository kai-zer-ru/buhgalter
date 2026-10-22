<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		label,
		count = null,
		children
	}: {
		label: string;
		count?: number | null;
		children: Snippet;
	} = $props();
</script>

<details class="account-group-panel">
	<summary class="account-group-summary">
		<span class="text-lg font-medium">
			{label}
			{#if count != null && count > 0}
				<span class="font-normal tabular-nums" style:color="var(--text-muted)">({count})</span>
			{/if}
		</span>
		<svg
			class="account-group-chevron"
			aria-hidden="true"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
		>
			<path d="m6 9 6 6 6-6" />
		</svg>
	</summary>
	<div class="pt-3 sm:pt-4">
		{@render children()}
	</div>
</details>
