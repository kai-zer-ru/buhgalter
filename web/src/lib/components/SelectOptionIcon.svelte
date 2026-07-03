<script lang="ts">
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import type { SelectOptionIcon } from '$lib/select-options';

	let { icon, size = 20 }: { icon: SelectOptionIcon; size?: number } = $props();
</script>

{#if icon.type === 'account'}
	<span class="shrink-0" aria-hidden="true">
		<AccountIcon type={icon.accountType} bankIcon={icon.bankIcon} {size} />
	</span>
{:else}
	<span class="shrink-0" aria-hidden="true">
		<CategoryIcon icon={icon.icon} {size} />
	</span>
{/if}
