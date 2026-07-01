<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		deleteTransaction,
		getDashboard,
		listTransactions,
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

	const txLimit = 10;
	let pastTx = $state<Transaction[]>([]);
	let pastTotal = $state(0);
	let pastLoading = $state(false);
	let plannedTx = $state<Transaction[]>([]);
	let plannedTotal = $state(0);
	let plannedLoading = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const pastVisible = $derived(dedupeTransferLegs(pastTx));
	const plannedVisible = $derived(dedupeTransferLegs(plannedTx));
	const txSiblings = $derived([...pastTx, ...plannedTx]);
	const hasDebts = $derived(
		dash != null && (dash.debts_summary.i_owe > 0 || dash.debts_summary.owed_to_me > 0)
	);

	onMount(() => {
		void loadAll();
	});

	async function loadDashboard() {
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

	async function loadPastTx() {
		pastLoading = true;
		try {
			const res = await listTransactions({
				kind: 'manual',
				sort: 'date_desc',
				page: '1',
				limit: String(txLimit)
			});
			pastTx = res.data;
			pastTotal = res.meta.total;
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			pastLoading = false;
		}
	}

	async function loadPlannedTx() {
		plannedLoading = true;
		try {
			const res = await listTransactions({
				kind: 'future',
				sort: 'date_asc',
				page: '1',
				limit: String(txLimit)
			});
			plannedTx = res.data;
			plannedTotal = res.meta.total;
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			plannedLoading = false;
		}
	}

	async function loadAll() {
		await loadDashboard();
		await Promise.all([loadPastTx(), loadPlannedTx()]);
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
			await loadAll();
		} catch (err) {
			toast(err instanceof ApiError ? err.message : $_('common.error'), 'error');
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
		<div class="grid gap-4 sm:grid-cols-2">
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
			<a href={resolve('/debts')} class="card block transition hover:opacity-90">
				<p class="text-sm" style:color="var(--text-muted)">{$_('debts.title')}</p>
				<div class="mt-1 space-y-1">
					{#if hasDebts}
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
					{:else}
						<p class="text-3xl font-semibold tabular-nums" style:color="var(--primary)">
							{$_('debts.summary.none')}
						</p>
					{/if}
				</div>
			</a>
		</div>

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
			{#if pastLoading && plannedLoading && pastTotal === 0 && plannedTotal === 0}
				<p style:color="var(--text-muted)">{$_('common.loading')}</p>
			{:else}
				<div class="card overflow-hidden">
					{#if !pastLoading && !plannedLoading && pastTotal === 0 && plannedTotal === 0}
						<p
							class="flex min-h-[7rem] items-center justify-center px-4 py-6 text-center text-sm"
							style:color="var(--text-muted)"
						>
							{$_('transactions.empty')}
						</p>
					{:else}
						{#if plannedTotal > 0 || plannedLoading}
							<details
								class:border-b={pastTotal > 0 || pastLoading}
								style:border-color="var(--border)"
							>
								<summary
									class="cursor-pointer list-none px-4 py-3 text-sm font-medium select-none [&::-webkit-details-marker]:hidden"
								>
									{$_('dashboard.group.planned')}
									<span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)">
										({plannedTotal})
									</span>
								</summary>
								{#if plannedLoading}
									<p class="px-4 pb-4 text-sm" style:color="var(--text-muted)">
										{$_('common.loading')}
									</p>
								{:else}
									<div class="md:overflow-x-auto">
										<TransactionList
											transactions={plannedVisible}
											siblings={txSiblings}
											{tz}
											emptyMessage={$_('transactions.empty')}
											showDescription
											showAmountSign
											showEdit
											showDelete
											onmakeRecurring={(tx) =>
												void goto(
													resolve(`/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`)
												)}
											onrepeat={openRepeat}
											onedit={openEdit}
											ondelete={(tx) => void removeTx(tx)}
										/>
									</div>
								{/if}
							</details>
						{/if}

						{#if pastTotal > 0 || pastLoading}
							<details open>
								<summary
									class="cursor-pointer list-none px-4 py-3 text-sm font-medium select-none [&::-webkit-details-marker]:hidden"
								>
									{$_('dashboard.group.past')}
									<span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)">
										({pastTotal})
									</span>
								</summary>
								{#if pastLoading}
									<p class="px-4 pb-4 text-sm" style:color="var(--text-muted)">
										{$_('common.loading')}
									</p>
								{:else}
									<div class="md:overflow-x-auto">
										<TransactionList
											transactions={pastVisible}
											siblings={txSiblings}
											{tz}
											emptyMessage={$_('transactions.empty')}
											showDescription
											showAmountSign
											showEdit
											showDelete
											onmakeRecurring={(tx) =>
												void goto(
													resolve(`/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`)
												)}
											onrepeat={openRepeat}
											onedit={openEdit}
											ondelete={(tx) => void removeTx(tx)}
										/>
									</div>
								{/if}
							</details>
						{/if}
					{/if}
					<div class="border-t px-4 py-3" style:border-color="var(--border)">
						<a href={resolve('/transactions')} class="btn-ghost">
							{$_('transactions.all')}
						</a>
					</div>
				</div>
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
	onsaved={loadAll}
/>
<TransferForm
	bind:open={transferOpen}
	editTx={editTransfer}
	repeatFrom={repeatTransfer}
	siblings={txSiblings}
	onclose={() => {
		transferOpen = false;
		editTransfer = null;
		repeatTransfer = null;
	}}
	onsaved={loadAll}
/>
