<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import {
		getBudgetSummary,
		getDashboard,
		listTransactions,
		type BudgetSummaryItem,
		type Dashboard,
		type Transaction
	} from '$lib/api/client';
	import { deleteTransaction, deleteTransfer } from '$lib/offline/transactions-api';
	import {
		mergeOutboxTransactions,
		refreshMergeMeta,
		mergeAccountsFallback
	} from '$lib/offline/merge';
	import { applyOutboxToDashboard } from '$lib/offline/local-state';
	import { outboxTick } from '$lib/offline/store';
	import { dataRefreshTick, localDataTick, scheduleSyncOutbox } from '$lib/offline/sync';
	import { refCacheReady, refCacheUpdate, readRefCache } from '$lib/offline/ref-cache';
	import { refCachePathMatches } from '$lib/offline/ref-cache-watch';
	import { assignIfChanged } from '$lib/state-utils';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import NewTransactionButtons from '$lib/components/NewTransactionButtons.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import {
		accountNewPath,
		transactionEditPath,
		transactionNewPath,
		transferEditPath,
		transferNewPath
	} from '$lib/android/form-routes';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { confirm } from '$lib/confirm';
	import { budgetStatusLine } from '$lib/budget-display';
	import { dedupeTransferLegs } from '$lib/transaction-display';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import { resolveAutoTopupSourceName } from '$lib/accounts/auto-topup';
	import { groupAccountsByType, accountGroupKind } from '$lib/accounts/group-by-type';
	import AccountGroupPanel from '$lib/components/AccountGroupPanel.svelte';
	import CollapsibleSection from '$lib/components/CollapsibleSection.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { reportPageLoadFailure } from '$lib/page-load';

	const DASHBOARD_PATH = '/api/v1/dashboard';
	const PAST_TX_PATH = '/api/v1/transactions?kind=manual&limit=10&page=1&sort=date_desc';
	const PLANNED_TX_PATH = '/api/v1/transactions?kind=future&limit=10&page=1&sort=date_desc';
	const BUDGET_PATH = '/api/v1/budgets/summary';

	let dashBase = $state<Dashboard | null>(readRefCache<Dashboard>(DASHBOARD_PATH));
	let loading = $state(!refCacheReady(DASHBOARD_PATH));
	let loadError = $state<string | null>(null);

	const txLimit = 10;
	const pastCached = readRefCache<{ data: Transaction[]; meta: { total: number } }>(PAST_TX_PATH);
	let serverPastTx = $state<Transaction[]>(pastCached?.data ?? []);
	let pastTotal = $state(pastCached?.meta.total ?? 0);
	let pastLoading = $state(!refCacheReady(PAST_TX_PATH));
	const plannedCached = readRefCache<{ data: Transaction[]; meta: { total: number } }>(
		PLANNED_TX_PATH
	);
	let serverPlannedTx = $state<Transaction[]>(plannedCached?.data ?? []);
	let plannedTotal = $state(plannedCached?.meta.total ?? 0);
	let plannedLoading = $state(!refCacheReady(PLANNED_TX_PATH));
	const budgetCached = readRefCache<{ items: BudgetSummaryItem[] }>(BUDGET_PATH);
	let budgetItems = $state<BudgetSummaryItem[]>(budgetCached?.items ?? []);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const dash = $derived.by(() => {
		void $outboxTick;
		void $localDataTick;
		return dashBase ? applyOutboxToDashboard(dashBase, tz) : null;
	});
	const pastTx = $derived.by(() => {
		void $outboxTick;
		void $localDataTick;
		return mergeOutboxTransactions(serverPastTx);
	});
	const plannedTx = $derived.by(() => {
		void $outboxTick;
		void $localDataTick;
		return mergeOutboxTransactions(serverPlannedTx);
	});
	const pastVisible = $derived(dedupeTransferLegs(pastTx));
	const plannedVisible = $derived(dedupeTransferLegs(plannedTx));
	const txSiblings = $derived([...pastTx, ...plannedTx]);
	const hasCreditCards = $derived(dash != null && dash.credit_cards_summary != null);
	const hasDebts = $derived(
		dash != null && (dash.debts_summary.i_owe > 0 || dash.debts_summary.owed_to_me > 0)
	);
	const currency = $derived($user?.currency ?? 'RUB');
	const accountGroups = $derived(dash ? groupAccountsByType(dash.accounts) : []);
	const recentTotal = $derived(pastTotal + plannedTotal);
	const categoryBudgets = $derived(
		[...budgetItems].filter((b) => b.scope !== 'all_expense').sort((a, b) => b.percent - a.percent)
	);
	const allExpenseBudget = $derived(budgetItems.find((b) => b.scope === 'all_expense'));

	function budgetProgressClass(status: string) {
		if (status === 'exceeded') return 'bg-red-500';
		if (status === 'warning') return 'bg-amber-500';
		return 'bg-emerald-500';
	}

	onMount(() => {
		void loadAll();
	});

	$effect(() => {
		const tick = $dataRefreshTick;
		if (tick === 0) return;
		void loadAll({ background: true });
	});

	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || !dashBase) return;
		if (refCachePathMatches(update.path, DASHBOARD_PATH)) {
			void loadDashboard({ background: true });
		}
		if (refCachePathMatches(update.path, PAST_TX_PATH)) {
			void loadPastTx({ background: true });
		}
		if (refCachePathMatches(update.path, PLANNED_TX_PATH)) {
			void loadPlannedTx({ background: true });
		}
		if (refCachePathMatches(update.path, BUDGET_PATH)) {
			void loadBudget({ background: true });
		}
	});

	async function loadDashboard(opts: { background?: boolean } = {}) {
		if (!opts.background && !refCacheReady(DASHBOARD_PATH)) loading = true;
		try {
			const next = await getDashboard();
			dashBase = opts.background ? assignIfChanged(dashBase, next) : next;
			if (dashBase) mergeAccountsFallback(dashBase.accounts);
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { background: opts.background, hasData: !!dashBase });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	async function loadPastTx(opts: { background?: boolean } = {}) {
		if (!opts.background && !refCacheReady(PAST_TX_PATH)) pastLoading = true;
		try {
			const res = await listTransactions({
				kind: 'manual',
				sort: 'date_desc',
				page: '1',
				limit: String(txLimit)
			});
			const nextData = res.data;
			serverPastTx = opts.background ? assignIfChanged(serverPastTx, nextData) : nextData;
			pastTotal = opts.background ? assignIfChanged(pastTotal, res.meta.total) : res.meta.total;
		} catch (err) {
			toast.fromError(err);
		} finally {
			pastLoading = false;
		}
	}

	async function loadPlannedTx(opts: { background?: boolean } = {}) {
		if (!opts.background && !refCacheReady(PLANNED_TX_PATH)) plannedLoading = true;
		try {
			const res = await listTransactions({
				kind: 'future',
				sort: 'date_desc',
				page: '1',
				limit: String(txLimit)
			});
			const nextData = res.data;
			serverPlannedTx = opts.background ? assignIfChanged(serverPlannedTx, nextData) : nextData;
			plannedTotal = opts.background
				? assignIfChanged(plannedTotal, res.meta.total)
				: res.meta.total;
		} catch (err) {
			toast.fromError(err);
		} finally {
			plannedLoading = false;
		}
	}

	async function loadBudget(opts: { background?: boolean } = {}) {
		try {
			const res = await getBudgetSummary();
			const next = res.items;
			budgetItems = opts.background ? assignIfChanged(budgetItems, next) : next;
		} catch {
			if (!opts.background) budgetItems = [];
		}
	}

	async function loadAll(opts: { background?: boolean } = {}) {
		await refreshMergeMeta().catch(() => undefined);
		await loadDashboard(opts);
		await Promise.all([loadPastTx(opts), loadPlannedTx(opts), loadBudget(opts)]);
		if (!opts.background) scheduleSyncOutbox();
		const { publishWidgetSnapshot } = await import('$lib/widgets/publish');
		void publishWidgetSnapshot();
	}

	function openNewTransaction(type: 'expense' | 'income') {
		void goto(resolve(transactionNewPath({ type, from: '/' })));
	}

	function openEdit(tx: Transaction) {
		if (tx.credit_payment_linked) return;
		if (tx.type === 'transfer' && tx.transfer_group_id) {
			void goto(resolve(transferEditPath(tx.transfer_group_id, '/')));
			return;
		}
		void goto(resolve(transactionEditPath(tx.id, '/')));
	}

	function openRepeat(tx: Transaction) {
		if (tx.credit_payment_linked) return;
		if (tx.type === 'transfer' && tx.transfer_group_id) {
			void goto(resolve(transferNewPath({ repeatId: tx.id, from: '/' })));
			return;
		}
		void goto(
			resolve(
				transactionNewPath({
					type: tx.type === 'income' ? 'income' : 'expense',
					repeatId: tx.id,
					from: '/'
				})
			)
		);
	}

	function openMakeRecurring(tx: Transaction) {
		void goto(resolve(`/settings/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`));
	}

	async function removeTx(tx: Transaction) {
		const msg =
			tx.type === 'transfer' && tx.transfer_group_id
				? $_('transactions.confirm.deleteTransfer')
				: $_('transactions.confirm.delete');
		const ok = await confirm({ message: msg, confirmLabel: $_('common.delete'), danger: true });
		if (!ok) return;
		try {
			if (tx.type === 'transfer' && tx.transfer_group_id) {
				await deleteTransfer(tx.transfer_group_id);
			} else {
				await deleteTransaction(tx.id);
			}
			toast($_('common.deleted'));
		} catch (err) {
			toast.fromError(err);
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
					href={resolve(accountNewPath('/'))}
					class="btn-primary hidden sm:inline-flex min-h-11 items-center"
				>
					{$_('accounts.new')}
				</a>
			{/if}
			<NewTransactionButtons
				onincome={() => openNewTransaction('income')}
				onexpense={() => openNewTransaction('expense')}
				ontransfer={() => void goto(resolve(transferNewPath({ from: '/' })))}
			/>
		</div>
	</div>

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
					<details
						class={allExpenseBudget ? 'border-t pt-2' : ''}
						style:border-color="var(--border)"
					>
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

	<PageLoadGate {loading} error={loadError} onretry={() => void loadAll()} inline>
		{#if dash}
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
				<div class="space-y-6">
					{#each accountGroups as group (accountGroupKind(group))}
						{@const kind = accountGroupKind(group)}
						<AccountGroupPanel {kind} count={group.length}>
							<div class="grid gap-4 sm:grid-cols-2">
								{#each group as acc (acc.id)}
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
														{$_('accounts.autoTopup.status', {
															values: { source: autoTopupSource }
														})}
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
						</AccountGroupPanel>
					{/each}
				</div>
			{/if}

			<CollapsibleSection
				label={$_('dashboard.recent')}
				count={!pastLoading && !plannedLoading ? recentTotal : null}
			>
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
												onrepeat={openRepeat}
												onmakeRecurring={openMakeRecurring}
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
												onrepeat={openRepeat}
												onmakeRecurring={openMakeRecurring}
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
			</CollapsibleSection>
		{/if}
	</PageLoadGate>
</div>
