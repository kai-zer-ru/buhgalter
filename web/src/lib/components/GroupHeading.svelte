<script lang="ts">
	let { title, hint = '' }: { title: string; hint?: string } = $props();
</script>

<div class="space-y-1 px-1">
	<h2 class="text-base font-semibold">{title}</h2>
	{#if hint}
		<p class="text-sm" style:color="var(--text-muted)">{hint}</p>
	{/if}
</div>
