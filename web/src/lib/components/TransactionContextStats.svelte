<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { getStatsContext, type StatsContext } from '$lib/api/client';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
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
			const msg = reportPageLoadFailure(err, { hasData: !!summary });
			if (msg) loadError = msg;
			if (!summary) summary = null;
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

<div class="card overflow-hidden p-0 sm:p-0">
	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if summary}
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
	</PageLoadGate>
</div>
