<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { getStatsContext, type StatsContext } from '$lib/api/client';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { isOfflineFetchError } from '$lib/offline/ref-cache';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { user } from '$lib/stores/auth';

	type Props = {
		params?: Record<string, string>;
		/** Если задано — число операций из списка (совпадает с пагинацией / спойлерами), не из stats API */
		transactionCount?: number;
	};

	let { params = {}, transactionCount }: Props = $props();
	let summary = $state<StatsContext | null>(null);
	let loading = $state(false);
	let loadError = $state<string | null>(null);

	const currency = $derived(params.currency || summary?.currency || $user?.currency || 'RUB');
	const paramsKey = $derived(JSON.stringify(params));

	$effect(() => {
		void paramsKey;
		void load();
	});

	async function load() {
		loading = true;
		try {
			summary = await getStatsContext(params);
			loadError = null;
		} catch (err) {
			summary = null;
			if (isOfflineFetchError(err)) {
				loadError = null;
			} else {
				const msg = reportPageLoadFailure(err, { hasData: false });
				if (msg) loadError = msg;
			}
		} finally {
			loading = false;
		}
	}
</script>

{#snippet tile(label: string, cents: number, color: string | undefined)}
	<div>
		<p class="stat-label">{label}</p>
		<p class="stat-value" style:color>
			<MoneyDisplay {cents} {currency} class="" />
		</p>
	</div>
{/snippet}

{#if loading || loadError || summary}
	<div class="card overflow-hidden {summary && !loading && !loadError ? 'p-0 sm:p-0' : ''}">
		{#if loading}
			<p class="text-sm" style:color="var(--text-muted)">{$_('common.loading')}</p>
		{:else if loadError}
			<div class="space-y-3 text-center text-sm" style:color="var(--text-muted)">
				<p>{loadError}</p>
				<button type="button" class="btn-primary" onclick={() => void load()}>
					{$_('common.retry')}
				</button>
			</div>
		{:else if summary}
			<div class="stat-grid grid-cols-2 md:grid-cols-3">
				{@render tile($_('stats.context.income'), summary.income_total, 'var(--primary)')}
				{@render tile($_('stats.context.expense'), summary.expense_total, 'var(--danger)')}
				<div>
					<p class="stat-label">{$_('stats.context.count')}</p>
					<p class="stat-value">{transactionCount ?? summary.transaction_count}</p>
				</div>
				{#if summary.lent_total !== undefined}
					{@render tile($_('stats.context.lent'), summary.lent_total, undefined)}
				{/if}
				{#if summary.borrowed_total !== undefined}
					{@render tile($_('stats.context.borrowed'), summary.borrowed_total, undefined)}
				{/if}
				{#if summary.paid_total !== undefined}
					{@render tile($_('stats.context.paid'), summary.paid_total, undefined)}
				{/if}
				{#if summary.remaining_amount !== undefined}
					{@render tile($_('stats.context.remaining'), summary.remaining_amount, undefined)}
				{/if}
			</div>
		{/if}
	</div>
{/if}
