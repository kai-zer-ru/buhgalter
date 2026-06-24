<script lang="ts">
	import { toastStore } from '$lib/toast';
</script>

<div
	class="pointer-events-none fixed inset-x-0 bottom-4 z-[80] flex flex-col items-center gap-2 px-4 sm:items-end sm:px-6"
	aria-live="polite"
>
	{#each $toastStore as item (item.id)}
		<div
			class="pointer-events-auto max-w-sm rounded-xl border px-4 py-3 text-sm shadow-lg"
			style:background-color="var(--bg-elevated)"
			style:border-color="var(--border)"
			style:color={item.type === 'error' ? 'var(--danger)' : 'var(--text)'}
			role="status"
		>
			{item.message}
		</div>
	{/each}
</div>
