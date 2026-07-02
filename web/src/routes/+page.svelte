<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		deleteTransaction,
		getBudgetSummary,
		getDashboard,
		listTransactions,
		type BudgetSummaryItem,
		type Dashboard,
		type Transaction
	} from '$lib/api/client';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import TransactionForm from '$lib/components/TransactionForm.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import NewTransactionButtons from '$lib/components/NewTransactionButtons.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { confirm } from '$lib/confirm';
	import { dedupeTransferLegs } from '$lib/transaction-display';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import { tr } from '$lib/i18n';
	import { budgetStatusLine } from '$lib/budget-display';
	import { resolveAutoTopupSourceName } from '$lib/accounts/auto-topup';

	let dash = $state<Dashboard | null>(null);
	let loading = $state(true);
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
	let budgetItems = $state<BudgetSummaryItem[]>([]);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const categoryBudgets = $derived(
		[...budgetItems].filter((b) => b.scope !== 'all_expense').sort((a, b) => b.percent - a.percent)
	);
	const allExpenseBudget = $derived(budgetItems.find((b) => b.scope === 'all_expense'));
	const pastVisible = $derived(dedupeTransferLegs(pastTx));
	const plannedVisible = $derived(dedupeTransferLegs(plannedTx));
	const txSiblings = $derived([...pastTx, ...plannedTx]);
	const hasDebts = $derived(
		dash != null && (dash.debts_summary.i_owe > 0 || dash.debts_summary.owed_to_me > 0)
	);
	const hasCreditCards = $derived(dash != null && dash.credit_cards_summary != null);
	const currency = $derived($user?.currency ?? 'RUB');

	let accountsPanelEl = $state<HTMLDetailsElement | undefined>();
	let accountsDesktop = $state(false);

	$effect(() => {
		const el = accountsPanelEl;
		if (!el) return;

		const mq = window.matchMedia('(min-width: 640px)');
		accountsDesktop = mq.matches;
		el.open = mq.matches;

		const onChange = (event: MediaQueryListEvent) => {
			accountsDesktop = event.matches;
			if (event.matches) el.open = true;
		};
		mq.addEventListener('change', onChange);
		return () => mq.removeEventListener('change', onChange);
	});

	function onAccountsPanelToggle() {
		if (accountsDesktop && accountsPanelEl) accountsPanelEl.open = true;
	}

	function budgetProgressClass(status: string) {
		if (status === 'exceeded') return 'bg-red-500';
		if (status === 'warning') return 'bg-amber-500';
		return 'bg-emerald-500';
	}

	onMount(() => {
		void loadAll();
	});

	async function loadDashboard() {
		loading = true;
		try {
			dash = await getDashboard();
		} catch (err) {
			toast.fromError(err);
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
			toast.fromError(err);
		} finally {
			pastLoading = false;
		}
	}

	async function loadPlannedTx() {
		plannedLoading = true;
		try {
			const res = await listTransactions({
				kind: 'future',
				sort: 'date_desc',
				page: '1',
				limit: String(txLimit)
			});
			plannedTx = res.data;
			plannedTotal = res.meta.total;
		} catch (err) {
			toast.fromError(err);
		} finally {
			plannedLoading = false;
		}
	}

	async function loadBudget() {
		try {
			const res = await getBudgetSummary();
			budgetItems = res.items;
		} catch {
			budgetItems = [];
		}
	}

	async function loadAll() {
		await loadDashboard();
		await Promise.all([loadPastTx(), loadPlannedTx(), loadBudget()]);
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
			toast.fromError(err);
		}
	}
</script>

<svelte:head>
	<title>{$_('dashboard.title')} — {$_('app.title')}</title>
</svelte:head>

{#snippet budgetWidget()}
	{#if budgetItems.length > 0}
		<div class="card space-y-2">
			<div class="flex items-center justify-between gap-2">
				<p class="text-sm" style:color="var(--text-muted)">{$_('budget.widget.title')}</p>
				<a href={resolve('/budget')} class="text-xs hover:underline" style:color="var(--primary)">
					{$_('budget.widget.more')} →
				</a>
			</div>
			{#if allExpenseBudget}
				<div>
					<div class="flex items-baseline justify-between gap-2">
						<span class="truncate text-sm font-medium">{allExpenseBudget.name}</span>
						<span class="shrink-0 text-xs" style:color="var(--text-muted)">
							<MoneyDisplay value={allExpenseBudget.spent_display} class="" />
							/
							<MoneyDisplay value={allExpenseBudget.planned_display} class="" />
						</span>
					</div>
					<div
						class="mt-1.5 h-1.5 overflow-hidden rounded-full"
						style:background-color="color-mix(in srgb, var(--border) 80%, transparent)"
					>
						<div
							class="h-full transition-all {budgetProgressClass(allExpenseBudget.status)}"
							style="width: {Math.min(allExpenseBudget.percent, 100)}%"
						></div>
					</div>
					<p class="mt-1 text-xs tabular-nums" style:color="var(--text-muted)">
						{budgetStatusLine(allExpenseBudget)}
					</p>
				</div>
			{/if}
			{#if categoryBudgets.length > 0}
				<details class={allExpenseBudget ? 'border-t pt-2' : ''} style:border-color="var(--border)">
					<summary
						class="cursor-pointer list-none text-xs font-medium select-none [&::-webkit-details-marker]:hidden"
						style:color="var(--text-muted)"
					>
						{tr('budget.widget.categories', {
							values: { count: String(categoryBudgets.length) }
						})}
					</summary>
					<ul class="mt-2 space-y-1.5">
						{#each categoryBudgets as item (item.id)}
							<li>
								<div class="flex items-center justify-between gap-2 text-xs">
									<span class="truncate">{item.name}</span>
									<span class="shrink-0 tabular-nums" style:color="var(--text-muted)">
										{item.percent}%
									</span>
								</div>
								<div
									class="mt-0.5 h-1 overflow-hidden rounded-full"
									style:background-color="color-mix(in srgb, var(--border) 80%, transparent)"
								>
									<div
										class="h-full {budgetProgressClass(item.status)}"
										style="width: {Math.min(item.percent, 100)}%"
									></div>
								</div>
							</li>
						{/each}
					</ul>
				</details>
			{/if}
		</div>
	{/if}
{/snippet}

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
	{:else if dash}
		{#if hasCreditCards}
			<div class="space-y-4">
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div class="card">
						<p class="text-sm" style:color="var(--text-muted)">{$_('dashboard.total')}</p>
						<p class="text-3xl font-semibold tabular-nums">
							<MoneyDisplay cents={dash.total_balance} {currency} class="" />
						</p>
						{#if dash.total_forecast !== dash.total_balance}
							<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
								{$_('dashboard.withPlans')}:
								<MoneyDisplay cents={dash.total_forecast} {currency} class="" />
							</p>
						{/if}
					</div>
					<div class="card">
						<p class="text-sm" style:color="var(--text-muted)">{$_('dashboard.creditCards')}</p>
						<p class="text-3xl font-semibold tabular-nums">
							<MoneyDisplay
								value={dash.credit_cards_summary!.total_balance_display}
								{currency}
								class=""
							/>
						</p>
						<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
							{$_('accounts.field.creditLimit')}:
							<MoneyDisplay
								value={dash.credit_cards_summary!.total_limit_display}
								{currency}
								class=""
							/>
						</p>
						{#if dash.credit_cards_summary!.total_forecast !== dash.credit_cards_summary!.total_balance}
							<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
								{$_('dashboard.withPlans')}:
								<MoneyDisplay
									value={dash.credit_cards_summary!.total_forecast_display}
									{currency}
									class=""
								/>
							</p>
						{/if}
					</div>
				</div>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<a href={resolve('/debts')} class="card block self-start transition hover:opacity-90">
						<p class="text-sm" style:color="var(--text-muted)">{$_('debts.title')}</p>
						<div class="mt-1 space-y-1">
							{#if hasDebts}
								{#if dash.debts_summary.i_owe > 0}
									<p class="tabular-nums" style:color="var(--danger)">
										{$_('debts.summary.iOwe')}:
										<MoneyDisplay cents={dash.debts_summary.i_owe} {currency} class="" />
									</p>
								{/if}
								{#if dash.debts_summary.owed_to_me > 0}
									<p class="tabular-nums" style:color="var(--primary)">
										{$_('debts.summary.owedToMe')}:
										<MoneyDisplay cents={dash.debts_summary.owed_to_me} {currency} class="" />
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
			</div>
		{:else}
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 sm:items-stretch">
				<div class="card flex h-full flex-col">
					<p class="text-sm" style:color="var(--text-muted)">{$_('dashboard.total')}</p>
					<p class="text-3xl font-semibold tabular-nums">
						<MoneyDisplay cents={dash.total_balance} {currency} class="" />
					</p>
					{#if dash.total_forecast !== dash.total_balance}
						<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
							{$_('dashboard.withPlans')}:
							<MoneyDisplay cents={dash.total_forecast} {currency} class="" />
						</p>
					{/if}
				</div>
				<a href={resolve('/debts')} class="card flex h-full flex-col transition hover:opacity-90">
					<p class="text-sm" style:color="var(--text-muted)">{$_('debts.title')}</p>
					<div class="mt-1 space-y-1">
						{#if hasDebts}
							{#if dash.debts_summary.i_owe > 0}
								<p class="tabular-nums" style:color="var(--danger)">
									{$_('debts.summary.iOwe')}:
									<MoneyDisplay cents={dash.debts_summary.i_owe} {currency} class="" />
								</p>
							{/if}
							{#if dash.debts_summary.owed_to_me > 0}
								<p class="tabular-nums" style:color="var(--primary)">
									{$_('debts.summary.owedToMe')}:
									<MoneyDisplay cents={dash.debts_summary.owed_to_me} {currency} class="" />
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
		{/if}

		{@render budgetWidget()}

		{#if dash.accounts.length === 0}
			<EmptyStateCard message={$_('dashboard.accountsEmpty')} />
		{:else}
			<details
				class="dashboard-accounts-panel"
				bind:this={accountsPanelEl}
				ontoggle={onAccountsPanelToggle}
			>
				<summary class="dashboard-accounts-summary sm:hidden">
					<span>
						{$_('dashboard.accountsSection')}
						<span class="font-normal tabular-nums" style:color="var(--text-muted)">
							({dash.accounts.length})
						</span>
					</span>
					<svg
						class="dashboard-accounts-chevron"
						aria-hidden="true"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<path d="m6 9 6 6 6-6" />
					</svg>
				</summary>
				<div class="grid gap-4 pt-3 sm:grid-cols-2 sm:pt-0">
					{#each dash.accounts as acc (acc.id)}
						<a
							href={resolve(`/accounts/${acc.id}`)}
							class="card flex items-center gap-4 transition hover:opacity-90"
						>
							<AccountIcon type={acc.type} bankIcon={acc.bank_icon} size={48} />
							<div class="min-w-0 flex-1">
								<p class="truncate font-medium">{acc.name}</p>
								<p class="mt-1 text-xl font-semibold tabular-nums">
									<MoneyDisplay value={acc.balance_display} {currency} class="" />
								</p>
								{#if acc.credit_limit_display}
									<p class="mt-0.5 text-sm tabular-nums" style:color="var(--text-muted)">
										{$_('accounts.field.creditLimit')}:
										<MoneyDisplay value={acc.credit_limit_display} {currency} class="" />
									</p>
								{/if}
								{#if acc.type === 'bank'}
									{@const autoTopupSource = resolveAutoTopupSourceName(acc, dash.accounts)}
									{#if autoTopupSource}
										<p class="mt-1 text-sm" style:color="var(--text-muted)">
											{$_('accounts.autoTopup.status', { values: { source: autoTopupSource } })}
										</p>
									{/if}
								{/if}
								{#if acc.forecast_balance !== acc.balance}
									<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
										{$_('dashboard.withPlans')}:
										<MoneyDisplay value={acc.forecast_display} {currency} class="" />
									</p>
								{/if}
							</div>
						</a>
					{/each}
				</div>
			</details>
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
													resolve(
														`/settings/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`
													)
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
													resolve(
														`/settings/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`
													)
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
