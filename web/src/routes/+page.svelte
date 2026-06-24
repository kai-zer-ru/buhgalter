<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { ApiError, getDashboard, type Dashboard } from '$lib/api/client';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import TransactionForm from '$lib/components/TransactionForm.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import { formatBalance } from '$lib/finance';
	import { fromCents } from '$lib/money';
	import { dedupeTransferLegs } from '$lib/transaction-display';
	import { user } from '$lib/stores/auth';

	let dash = $state<Dashboard | null>(null);
	let loading = $state(true);
	let error = $state('');
	let txOpen = $state(false);
	let transferOpen = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const recentTx = $derived(dash ? dedupeTransferLegs(dash.recent_transactions) : []);

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			dash = await getDashboard();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{$_('dashboard.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-semibold">{$_('dashboard.title')}</h1>
		<div class="flex gap-2">
			{#if dash?.accounts.length === 0}
				<a href={resolve('/accounts/new')} class="btn-primary">{$_('accounts.new')}</a>
			{/if}
			<button type="button" class="btn-primary" onclick={() => (txOpen = true)}>
				+ {$_('transactions.new')}
			</button>
			<button type="button" class="btn-ghost" onclick={() => (transferOpen = true)}>
				{$_('transactions.transfer')}
			</button>
		</div>
	</div>

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if error}
		<p style:color="var(--danger)">{error}</p>
	{:else if dash}
		<div class="card">
			<p class="text-sm" style:color="var(--text-muted)">{$_('dashboard.total')}</p>
			<p class="text-3xl font-semibold tabular-nums">
				{formatBalance(fromCents(dash.total_balance), $user?.currency ?? 'RUB')}
			</p>
			{#if dash.total_forecast !== dash.total_balance}
				<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
					{$_('dashboard.withPlans')}:
					{formatBalance(fromCents(dash.total_forecast), $user?.currency ?? 'RUB')}
				</p>
			{/if}
		</div>

		{#if dash.debts_summary && (dash.debts_summary.i_owe > 0 || dash.debts_summary.owed_to_me > 0)}
			<div class="card space-y-1">
				{#if dash.debts_summary.i_owe > 0}
					<p class="tabular-nums" style:color="var(--danger)">
						{$_('debts.summary.iOwe')}:
						{formatBalance(fromCents(dash.debts_summary.i_owe), $user?.currency ?? 'RUB')}
					</p>
				{/if}
				{#if dash.debts_summary.owed_to_me > 0}
					<p class="tabular-nums" style:color="var(--primary)">
						{$_('debts.summary.owedToMe')}:
						{formatBalance(fromCents(dash.debts_summary.owed_to_me), $user?.currency ?? 'RUB')}
					</p>
				{/if}
				<p class="text-sm">
					<a href={resolve('/debts')} style:color="var(--primary)">{$_('debts.more')}</a>
				</p>
			</div>
		{/if}

		{#if dash.accounts.length === 0}
			<EmptyStateCard message={$_('dashboard.accountsEmpty')} />
		{:else}
			<div class="grid gap-4 sm:grid-cols-2">
				{#each dash.accounts as acc (acc.id)}
					<a
						href={resolve(`/accounts/${acc.id}`)}
						class="card flex items-center gap-4 transition hover:opacity-90"
					>
						<AccountIcon type={acc.type} bankIcon={acc.bank_icon} size={48} />
						<div class="min-w-0 flex-1">
							<p class="truncate font-medium">{acc.name}</p>
							<p class="mt-1 text-xl font-semibold tabular-nums">
								{formatBalance(acc.balance_display, $user?.currency ?? 'RUB')}
							</p>
							{#if acc.has_future_this_month}
								<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
									{$_('dashboard.withPlans')}:
									{formatBalance(acc.forecast_display, $user?.currency ?? 'RUB')}
								</p>
							{/if}
						</div>
					</a>
				{/each}
			</div>
		{/if}

		<section>
			<h2 class="mb-3 text-lg font-medium">{$_('dashboard.recent')}</h2>
			{#if recentTx.length === 0}
				<EmptyStateCard message={$_('transactions.empty')} />
			{:else}
				<div class="card md:overflow-x-auto">
					<TransactionList
						transactions={recentTx}
						siblings={dash.recent_transactions}
						{tz}
						emptyMessage={$_('transactions.empty')}
						showDescription
						showAmountSign
					/>
				</div>
				<p class="mt-2">
					<a href={resolve('/transactions')} style:color="var(--primary)"
						>{$_('transactions.all')}</a
					>
				</p>
			{/if}
		</section>
	{/if}
</div>

<TransactionForm bind:open={txOpen} onclose={() => (txOpen = false)} onsaved={load} />
<TransferForm bind:open={transferOpen} onclose={() => (transferOpen = false)} onsaved={load} />
