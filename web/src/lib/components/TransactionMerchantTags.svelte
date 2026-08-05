<script lang="ts">
	import type { TagRef } from '$lib/api/client';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';

	let {
		merchantName = null,
		merchantIcon = null,
		tags = null
	}: {
		merchantName?: string | null;
		merchantIcon?: string | null;
		tags?: TagRef[] | null;
	} = $props();

	const name = $derived(merchantName?.trim() ?? '');
	const tagList = $derived(tags ?? []);
	const visible = $derived(Boolean(name) || tagList.length > 0);
</script>

{#if visible}
	<span class="inline-flex min-w-0 flex-wrap items-center gap-1.5">
		{#if name}
			<span class="inline-flex min-w-0 items-center gap-1">
				<CategoryIcon icon={merchantIcon || 'default'} size={20} />
				<span class="truncate leading-none">{name}</span>
			</span>
		{/if}
		{#each tagList as t (t.id)}
			<span
				class="rounded-md px-1.5 py-0.5 text-xs leading-none"
				style:background="var(--surface-2)"
			>
				{t.name}
			</span>
		{/each}
	</span>
{/if}
