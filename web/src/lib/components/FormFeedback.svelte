<script lang="ts">
	let {
		error = '',
		success = ''
	}: {
		error?: string;
		success?: string;
	} = $props();
</script>

{#if error}
	<p class="text-sm" style:color="var(--danger)" role="alert">{error}</p>
{/if}
{#if success}
	<p class="text-sm" style:color="var(--primary)" role="status">{success}</p>
{/if}
