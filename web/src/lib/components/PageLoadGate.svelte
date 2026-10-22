<script lang="ts">
	import type { Snippet } from 'svelte';
	import { _ } from 'svelte-i18n';
	import EmptyStateCard from './EmptyStateCard.svelte';

	let {
		loading = false,
		error = null as string | null,
		onretry,
		inline = false,
		children
	}: {
		loading?: boolean;
		error?: string | null;
		onretry?: () => void;
		/** Use a plain paragraph for loading instead of a card. */
		inline?: boolean;
		children?: Snippet;
	} = $props();
</script>

{#if loading}
	{#if inline}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else}
		<EmptyStateCard message={$_('common.loading')} ariaBusy />
	{/if}
{:else if error}
	<EmptyStateCard message={error}>
		{#if onretry}
			<button type="button" class="btn-primary" onclick={() => onretry?.()}>
				{$_('common.retry')}
			</button>
		{/if}
	</EmptyStateCard>
{:else}
	{@render children?.()}
{/if}
