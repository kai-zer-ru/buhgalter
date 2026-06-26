<script lang="ts">
	import { _ } from 'svelte-i18n';

	type Props = {
		page: number;
		limit: number;
		total: number;
		onchange: (nextPage: number) => void;
	};

	let { page, limit, total, onchange }: Props = $props();
	const totalPages = $derived(Math.max(1, Math.ceil(total / Math.max(1, limit))));
</script>

{#if totalPages > 1}
	<div class="flex flex-wrap items-center justify-between gap-2">
		<p class="text-sm" style:color="var(--text-muted)">
			{$_('transactions.pagination.page', { values: { page, pages: totalPages } })}
		</p>
		<div class="flex items-center gap-2">
			<button
				type="button"
				class="btn-ghost"
				disabled={page <= 1}
				onclick={() => onchange(Math.max(1, page - 1))}
			>
				{$_('transactions.pagination.prev')}
			</button>
			<button
				type="button"
				class="btn-ghost"
				disabled={page >= totalPages}
				onclick={() => onchange(Math.min(totalPages, page + 1))}
			>
				{$_('transactions.pagination.next')}
			</button>
		</div>
	</div>
{/if}
