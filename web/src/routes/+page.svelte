<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		deleteTransaction,
		getDashboard,
		type Dashboard,
		type Transaction
	} from '$lib/api/client';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import TransactionForm from '$lib/components/TransactionForm.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import NewTransactionButtons from '$lib/components/NewTransactionButtons.svelte';
	import { confirm } from '$lib/confirm';
	import { formatBalance } from '$lib/finance';
	import { fromCents } from '$lib/money';
	import { dedupeTransferLegs } from '$lib/transaction-display';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let dash = $state<Dashboard | null>(null);
	let loading = $state(true);
	let error = $state('');
	let txOpen = $state(false);
	let transferOpen = $state(false);
	let editTx = $state<Transaction | null>(null);
	let editTransfer = $state<Transaction | null>(null);
	let repeatTx = $state<Transaction | null>(null);
	let repeatTransfer = $state<Transaction | null>(null);
	let newTxType = $state<'expense' | 'income'>('expense');

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

	function openNewTransaction(type: 'expense' | 'income') {
		editTx = null;
		repeatTx = null;
		newTxType = type;
		txOpen = true;
	}

	function openEdit(tx: Transaction) {
		if (tx.credit_payment_linked) return;
		if (tx.type === 'transfer' && tx.transfer_group_id) {
			editTransfer = tx;
			editTx = null;
			repeatTransfer = null;
			repeatTx = null;
			transferOpen = true;
			return;
		}
		editTransfer = null;
		repeatTransfer = null;
		repeatTx = null;
		editTx = tx;
		txOpen = true;
	}

	function openRepeat(tx: Transaction) {
		if (tx.credit_payment_linked) return;
		if (tx.type === 'transfer' && tx.transfer_group_id) {
			editTransfer = null;
			editTx = null;
			repeatTx = null;
			repeatTransfer = tx;
			transferOpen = true;
			return;
		}
		editTransfer = null;
		editTx = null;
		repeatTransfer = null;
		repeatTx = tx;
		newTxType = tx.type === 'income' ? 'income' : 'expense';
		txOpen = true;
	}

	async function removeTx(tx: Transaction) {
		const msg =
			tx.type === 'transfer' && tx.transfer_group_id
				? $_('transactions.confirm.deleteTransfer')
				: $_('transactions.confirm.delete');
		const ok = await confirm({ message: msg, confirmLabel: $_('common.delete'), danger: true });
		if (!ok) return;
		try {
			await deleteTransaction(tx.id);
			toast($_('common.deleted'));
			await load();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		}
	}
</script>

<svelte:head>
	<title>{$_('dashboard.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-semibold">{$_('dashboard.title')}</h1>
		<div class="flex shrink-0 items-center gap-1">
			{#if dash?.accounts.length === 0}
				<a
					href={resolve('/accounts/new')}
					class="btn-primary hidden sm:inline-flex min-h-11 items-center"
				>
					{$_('accounts.new')}
				</a>
			{/if}
			<NewTransactionButtons
				onincome={() => openNewTransaction('income')}
				onexpense={() => openNewTransaction('expense')}
				ontransfer={() => {
					editTransfer = null;
					repeatTransfer = null;
					transferOpen = true;
				}}
			/>
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
							{#if acc.forecast_balance !== acc.balance}
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
						showEdit
						showDelete
						onmakeRecurring={(tx) =>
							void goto(resolve(`/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`))}
						onrepeat={openRepeat}
						onedit={openEdit}
						ondelete={(tx) => void removeTx(tx)}
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

<TransactionForm
	bind:open={txOpen}
	defaultType={newTxType}
	transaction={editTx}
	repeatFrom={repeatTx}
	onclose={() => {
		txOpen = false;
		editTx = null;
		repeatTx = null;
	}}
	onsaved={load}
/>
<TransferForm
	bind:open={transferOpen}
	editTx={editTransfer}
	repeatFrom={repeatTransfer}
	siblings={dash?.recent_transactions ?? []}
	onclose={() => {
		transferOpen = false;
		editTransfer = null;
		repeatTransfer = null;
	}}
	onsaved={load}
/>
