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

	const currency = $derived($user?.currency ?? 'RUB');
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

<div class="card">
	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if summary}
			<div class="grid gap-3 md:grid-cols-3">
				<div>
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.income')}</p>
					<p class="tabular-nums font-medium">
						<MoneyDisplay cents={summary.income_total} {currency} class="" />
					</p>
				</div>
				<div>
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.expense')}</p>
					<p class="tabular-nums font-medium">
						<MoneyDisplay cents={summary.expense_total} {currency} class="" />
					</p>
				</div>
				<div>
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.count')}</p>
					<p class="tabular-nums font-medium">{transactionCount ?? summary.transaction_count}</p>
				</div>
				{#if summary.lent_total !== undefined}
					<div>
						<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.lent')}</p>
						<p class="tabular-nums font-medium">
							<MoneyDisplay cents={summary.lent_total} {currency} class="" />
						</p>
					</div>
				{/if}
				{#if summary.borrowed_total !== undefined}
					<div>
						<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.borrowed')}</p>
						<p class="tabular-nums font-medium">
							<MoneyDisplay cents={summary.borrowed_total} {currency} class="" />
						</p>
					</div>
				{/if}
				{#if summary.paid_total !== undefined}
					<div>
						<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.paid')}</p>
						<p class="tabular-nums font-medium">
							<MoneyDisplay cents={summary.paid_total} {currency} class="" />
						</p>
					</div>
				{/if}
				{#if summary.remaining_amount !== undefined}
					<div>
						<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.remaining')}</p>
						<p class="tabular-nums font-medium">
							<MoneyDisplay cents={summary.remaining_amount} {currency} class="" />
						</p>
					</div>
				{/if}
			</div>
		{/if}
	</PageLoadGate>
</div>
