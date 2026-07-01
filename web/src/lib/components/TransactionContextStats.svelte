<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { getStatsContext, type StatsContext } from '$lib/api/client';
	import { formatBalance } from '$lib/finance';
	import { fromCents } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	type Props = {
		params?: Record<string, string>;
	};

	let { params = {} }: Props = $props();
	let summary = $state<StatsContext | null>(null);
	let loading = $state(false);

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
		} catch (err) {
			toast.fromError(err);
			summary = null;
		} finally {
			loading = false;
		}
	}
</script>

<div class="card">
	{#if loading}
		<p class="text-sm" style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if summary}
		<div class="grid gap-3 md:grid-cols-3">
			<div>
				<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.income')}</p>
				<p class="tabular-nums font-medium">
					{formatBalance(fromCents(summary.income_total), currency)}
				</p>
			</div>
			<div>
				<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.expense')}</p>
				<p class="tabular-nums font-medium">
					{formatBalance(fromCents(summary.expense_total), currency)}
				</p>
			</div>
			<div>
				<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.count')}</p>
				<p class="tabular-nums font-medium">{summary.transaction_count}</p>
			</div>
			{#if summary.lent_total !== undefined}
				<div>
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.lent')}</p>
					<p class="tabular-nums font-medium">
						{formatBalance(fromCents(summary.lent_total), currency)}
					</p>
				</div>
			{/if}
			{#if summary.borrowed_total !== undefined}
				<div>
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.borrowed')}</p>
					<p class="tabular-nums font-medium">
						{formatBalance(fromCents(summary.borrowed_total), currency)}
					</p>
				</div>
			{/if}
			{#if summary.paid_total !== undefined}
				<div>
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.paid')}</p>
					<p class="tabular-nums font-medium">
						{formatBalance(fromCents(summary.paid_total), currency)}
					</p>
				</div>
			{/if}
			{#if summary.remaining_amount !== undefined}
				<div>
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.context.remaining')}</p>
					<p class="tabular-nums font-medium">
						{formatBalance(fromCents(summary.remaining_amount), currency)}
					</p>
				</div>
			{/if}
		</div>
	{/if}
</div>
