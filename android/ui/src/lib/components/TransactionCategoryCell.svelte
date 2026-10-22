<script lang="ts">
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';

	let {
		categoryName,
		categoryIcon,
		subcategoryName,
		subcategoryIcon,
		typeFallback = ''
	}: {
		categoryName?: string | null;
		categoryIcon?: string | null;
		subcategoryName?: string | null;
		subcategoryIcon?: string | null;
		typeFallback?: string;
	} = $props();

	const label = $derived(categoryName ?? typeFallback);
	const subIcon = $derived(subcategoryIcon ?? categoryIcon);
</script>

{#if subcategoryName}
	<span class="inline-flex flex-wrap items-center gap-1 align-middle">
		{#if categoryIcon}
			<span class="inline-flex items-center gap-1">
				<CategoryIcon icon={categoryIcon} size={24} />
				<span class="leading-none">{label}</span>
			</span>
		{:else}
			<span class="leading-none">{label}</span>
		{/if}
		<span class="leading-none" style:color="var(--text-muted)">→</span>
		{#if subIcon}
			<span class="inline-flex items-center gap-1">
				<CategoryIcon icon={subIcon} size={24} />
				<span class="leading-none">{subcategoryName}</span>
			</span>
		{:else}
			<span class="leading-none">{subcategoryName}</span>
		{/if}
	</span>
{:else if categoryIcon}
	<span class="inline-flex items-center gap-1 align-middle">
		<CategoryIcon icon={categoryIcon} size={24} />
		<span class="leading-none">{label}</span>
	</span>
{:else}
	{label}
{/if}
