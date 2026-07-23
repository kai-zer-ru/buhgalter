<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		getAccount,
		getAccountBalance,
		listAccounts,
		listBanks,
		listCategories,
		listTransactions,
		setPrimaryAccount,
		type Account,
		type AccountBalanceSummary,
		type Bank,
		type Category,
		type Transaction
	} from '$lib/api/client';
	import { unarchiveAccount, updateAccount } from '$lib/offline/accounts-api';
	import { deleteTransaction, deleteTransfer } from '$lib/offline/transactions-api';
	import { mergeOutboxTransactions, refreshMergeMeta } from '$lib/offline/merge';
	import { applyOutboxToAccountBalance, applyOutboxToAccounts } from '$lib/offline/local-state';
	import { outboxTick } from '$lib/offline/store';
	import { dataRefreshTick, localDataTick, scheduleSyncOutbox } from '$lib/offline/sync';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import BackLink from '$lib/components/BackLink.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import TransactionFilters from '$lib/components/TransactionFilters.svelte';
	import { resolveAppPath } from '$lib/android/form-nav';
	import {
		transactionEditPath,
		transactionNewPath,
		transferEditPath,
		transferNewPath,
		accountChargeFeePath,
		accountAutoTopupPath
	} from '$lib/android/form-routes';
	import NewTransactionButtons from '$lib/components/NewTransactionButtons.svelte';
	import TransactionContextStats from '$lib/components/TransactionContextStats.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import TransactionPagination from '$lib/components/TransactionPagination.svelte';
	import { isAutoTopupEligible, resolveAutoTopupSourceName } from '$lib/accounts/auto-topup';
	import { accountSelectOptions } from '$lib/select-options';
	import { isCreditCard } from '$lib/credit-card';
	import {
		promptArchiveAccount,
		executeArchiveAccount,
		promptDeleteAccount,
		executeDeleteAccount
	} from '$lib/accounts/account-inactive-prompt';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { canSetAsPrimary, formatAccountInitialBalanceForEdit } from '$lib/accounts';
	import { toAPIAmount } from '$lib/money';
	import { fromDateLocalEnd, fromDateLocalStart } from '$lib/dates';
	import { dedupeTransferLegs } from '$lib/transaction-display';
	import { user } from '$lib/stores/auth';
	import { refCacheReadyAny, refCacheUpdate } from '$lib/offline/ref-cache';
	import { refCachePathMatches } from '$lib/offline/ref-cache-watch';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { assignIfChanged } from '$lib/state-utils';

	let acc = $state<Account | null>(null);
	let accBalanceBase = $state<AccountBalanceSummary | null>(null);
	let banks = $state<Bank[]>([]);
	let categories = $state<Category[]>([]);
	let serverTransactions = $state<Transaction[]>([]);
	let txTotal = $state(0);
	let txPage = $state(1);
	const txLimit = 20;
	let editing = $state(false);
	let name = $state('');
	let bankId = $state('');
	let creditLimit = $state('');
	let paymentAccountId = $state('');
	let initialBalance = $state('');
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let filterLoading = $state(false);
	let saving = $state(false);
	let fromLocal = $state('');
	let toLocal = $state('');
	let typeFilter = $state('');
	let categoryFilter = $state('');
	let kindFilter = $state('');
	let searchFilter = $state('');
	let filtersAutoApplyReady = $state(false);
	let lastFiltersKey = $state('');

	const id = $derived($page.params.id ?? '');
	const accountFrom = $derived(id ? `/accounts/${id}` : '/accounts');
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const transactions = $derived.by(() => {
		void $outboxTick;
		void $localDataTick;
		return mergeOutboxTransactions(serverTransactions);
	});
	const visibleTx = $derived(dedupeTransferLegs(transactions));
	const bankOptions = $derived(banks.map((bank) => ({ value: bank.id, label: bank.name })));
	let allAccountsBase = $state<Account[]>([]);
	const allAccounts = $derived.by(() => {
		void $outboxTick;
		void $localDataTick;
		return applyOutboxToAccounts(allAccountsBase, tz);
	});
	const accBalance = $derived.by(() => {
		void $outboxTick;
		void $localDataTick;
		return accBalanceBase ? applyOutboxToAccountBalance(accBalanceBase, tz) : null;
	});
	const debitPaymentOptions = $derived(
		accountSelectOptions(
			allAccounts.filter((a) => a.status === 'active' && a.type !== 'credit_card' && a.id !== id)
		)
	);

	$effect(() => {
		const tick = $dataRefreshTick;
		if (tick === 0 || !id) return;
		// Reload account + balance + txs (not only the list) after mutations.
		void load({ background: true });
	});

	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || !acc || loadedForID !== id) return;
		if (
			refCachePathMatches(update.path, `/api/v1/accounts/${id}`) ||
			refCachePathMatches(update.path, `/api/v1/accounts/${id}/balance`)
		) {
			void load({ background: true });
		} else if (refCachePathMatches(update.path, '/api/v1/transactions')) {
			void loadTransactions({ background: true });
		}
	});

	let loadedForID = $state('');
	$effect(() => {
		if (!id) return;
		if (loadedForID === id) return;
		loadedForID = id;
		filtersAutoApplyReady = false;
		readURLFilters();
		void load().then(() => {
			lastFiltersKey = currentFiltersKey();
			filtersAutoApplyReady = true;
		});
	});

	$effect(() => {
		const nextKey = currentFiltersKey();
		if (!filtersAutoApplyReady) return;
		if (nextKey === lastFiltersKey) return;
		lastFiltersKey = nextKey;
		txPage = 1;
		void applyURLFilters();
	});

	function currentFiltersKey(): string {
		return JSON.stringify({
			fromLocal,
			toLocal,
			type: typeFilter,
			categoryId: categoryFilter,
			kind: kindFilter,
			search: searchFilter.trim()
		});
	}

	function accountStatsContextParams() {
		const params: Record<string, string> = {
			account_id: id,
			from: fromLocal ? fromDateLocalStart(fromLocal, tz) : '',
			to: toLocal ? fromDateLocalEnd(toLocal, tz) : '',
			type: typeFilter,
			category_id: categoryFilter,
			search: searchFilter
		};
		if (kindFilter) {
			params.kind = kindFilter;
		} else {
			params.include_future = 'true';
		}
		return params;
	}

	function readURLFilters() {
		const q = $page.url.searchParams;
		txPage = Number(q.get('page') || '1');
		fromLocal = q.get('from_local') ?? '';
		toLocal = q.get('to_local') ?? '';
		typeFilter = q.get('type') ?? '';
		categoryFilter = q.get('category_id') ?? '';
		kindFilter = q.get('kind') ?? '';
		searchFilter = q.get('search') ?? '';
	}

	async function load(opts: { background?: boolean } = {}) {
		if (!id) return;
		const hasCache = refCacheReadyAny([
			`/api/v1/accounts/${id}`,
			`/api/v1/accounts/${id}/balance`,
			'/api/v1/banks'
		]);
		if (!opts.background && !hasCache) loading = true;
		try {
			await refreshMergeMeta().catch(() => undefined);
			const [account, accountBalance, bankList, expenseCats, incomeCats, accountList] =
				await Promise.all([
					getAccount(id),
					getAccountBalance(id),
					listBanks(),
					listCategories('expense'),
					listCategories('income'),
					listAccounts()
				]);
			acc = opts.background ? assignIfChanged(acc, account) : account;
			accBalanceBase = opts.background
				? assignIfChanged(accBalanceBase, accountBalance)
				: accountBalance;
			banks = opts.background ? assignIfChanged(banks, bankList) : bankList;
			allAccountsBase = opts.background
				? assignIfChanged(allAccountsBase, accountList)
				: accountList;
			const uniqueCatsByID: Record<string, Category> = {};
			for (const cat of [...expenseCats, ...incomeCats]) uniqueCatsByID[cat.id] = cat;
			const nextCategories = Object.values(uniqueCatsByID).sort((a, b) =>
				a.name.localeCompare(b.name, 'ru')
			);
			categories = opts.background ? assignIfChanged(categories, nextCategories) : nextCategories;
			name = account.name;
			bankId = account.bank_id ?? '';
			creditLimit = account.credit_limit_display ?? '';
			paymentAccountId = account.payment_account_id ?? '';
			initialBalance = formatAccountInitialBalanceForEdit(account.initial_balance);
			editing = $page.url.searchParams.get('edit') === '1' && account.status !== 'deleted';
			await loadTransactions({ background: opts.background });
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { background: opts.background, hasData: !!acc });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	async function loadTransactions(opts: { background?: boolean } = {}) {
		if (!id) return;
		if (!opts.background) filterLoading = true;
		try {
			const params: Record<string, string> = {
				account_id: id,
				page: String(txPage),
				limit: String(txLimit),
				sort: 'date_desc'
			};
			if (fromLocal) params.from = fromDateLocalStart(fromLocal, tz);
			if (toLocal) params.to = fromDateLocalEnd(toLocal, tz);
			if (typeFilter) params.type = typeFilter;
			if (categoryFilter) params.category_id = categoryFilter;
			if (kindFilter) params.kind = kindFilter;
			if (searchFilter.trim()) params.search = searchFilter.trim();
			const result = await listTransactions(params);
			const nextData = result.data;
			serverTransactions = opts.background
				? assignIfChanged(serverTransactions, nextData)
				: nextData;
			txTotal = opts.background ? assignIfChanged(txTotal, result.meta.total) : result.meta.total;
			if (!opts.background) scheduleSyncOutbox();
		} catch (err) {
			toast.fromError(err);
		} finally {
			filterLoading = false;
		}
	}

	async function applyURLFilters() {
		const queryParts = [`page=${encodeURIComponent(String(txPage))}`];
		if (fromLocal) queryParts.push(`from_local=${encodeURIComponent(fromLocal)}`);
		if (toLocal) queryParts.push(`to_local=${encodeURIComponent(toLocal)}`);
		if (typeFilter) queryParts.push(`type=${encodeURIComponent(typeFilter)}`);
		if (categoryFilter) queryParts.push(`category_id=${encodeURIComponent(categoryFilter)}`);
		if (kindFilter) queryParts.push(`kind=${encodeURIComponent(kindFilter)}`);
		if (searchFilter.trim()) queryParts.push(`search=${encodeURIComponent(searchFilter.trim())}`);
		await goto(resolveAppPath(`/accounts/${id}?${queryParts.join('&')}`), {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
		await loadTransactions();
	}

	function applyLimitToBalance() {
		if (creditLimit.trim()) initialBalance = creditLimit;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!acc) return;
		saving = true;
		try {
			acc = await updateAccount(acc.id, {
				name,
				bank_id: acc.type === 'bank' || acc.type === 'credit_card' ? bankId : undefined,
				initial_balance: toAPIAmount(initialBalance),
				credit_limit: acc.type === 'credit_card' ? toAPIAmount(creditLimit) : undefined,
				payment_account_id: acc.type === 'credit_card' ? paymentAccountId || null : undefined
			});
			accBalanceBase = await getAccountBalance(acc.id);
			editing = false;
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}

	async function toggleArchive() {
		if (!acc) return;
		if (acc.status === 'active') {
			const activeOnly = allAccounts.filter((a) => a.status === 'active');
			const confirmed = await promptArchiveAccount({ acc, activeAccounts: activeOnly });
			if (!confirmed.ok) return;
			try {
				acc = await executeArchiveAccount(acc, confirmed.transferToAccountId);
				accBalanceBase = await getAccountBalance(acc.id);
				toast($_('common.saved'));
			} catch (err) {
				toast.fromError(err);
			}
			return;
		}
		try {
			acc = await unarchiveAccount(acc.id);
			accBalanceBase = await getAccountBalance(acc.id);
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function makePrimary() {
		if (!acc || !canSetAsPrimary(acc)) return;
		try {
			acc = await setPrimaryAccount(acc.id);
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function remove() {
		if (!acc) return;
		const confirmed = await promptDeleteAccount({ acc, activeAccounts: allAccounts });
		if (!confirmed.ok) return;
		try {
			await executeDeleteAccount(acc, confirmed.transferToAccountId);
			toast($_('common.deleted'));
			await goto(resolve('/accounts?status=deleted'));
		} catch (err) {
			toast.fromError(err);
		}
	}

	function openNewTransaction(type: 'expense' | 'income') {
		void goto(resolve(transactionNewPath({ type, accountId: id, from: accountFrom })));
	}

	function openTransfer() {
		void goto(resolve(transferNewPath({ accountId: id, from: accountFrom })));
	}

	function openPayTransfer() {
		void goto(resolve(transferNewPath({ payCardId: id, from: accountFrom })));
	}

	const accountTxReadOnly = $derived(acc?.status === 'deleted');

	function accountActions(includeTransactions = false): RowAction[] {
		if (!acc || acc.status === 'deleted') return [];
		const actions: RowAction[] = [];
		if (acc.status === 'active' && includeTransactions) {
			actions.push(
				{
					icon: 'income',
					label: $_('transactions.type.income'),
					onclick: () => openNewTransaction('income')
				},
				{
					icon: 'expense',
					label: $_('transactions.type.expense'),
					onclick: () => openNewTransaction('expense')
				},
				{
					icon: 'transfer',
					label: $_('transactions.transfer'),
					onclick: openTransfer
				}
			);
		}
		if (acc.status === 'active' && isCreditCard(acc)) {
			actions.push(
				{
					icon: 'transfer',
					label: $_('accounts.creditCard.pay'),
					onclick: openPayTransfer
				},
				{
					icon: 'expense',
					label: $_('accounts.creditCard.chargeFee'),
					onclick: () => {
						if (!acc) return;
						void goto(resolve(accountChargeFeePath(acc.id, `/accounts/${acc.id}`)));
					}
				}
			);
		}
		actions.push({
			icon: 'edit',
			label: $_('accounts.action.edit'),
			onclick: () => (editing = true)
		});
		if (acc.status === 'active' && isAutoTopupEligible(acc)) {
			actions.push({
				icon: 'transfer',
				label: $_('accounts.action.autoTopup'),
				onclick: () => {
					if (!acc) return;
					void goto(resolve(accountAutoTopupPath(acc.id, `/accounts/${acc.id}`)));
				}
			});
		}
		if (canSetAsPrimary(acc)) {
			actions.push({
				icon: 'save',
				label: $_('accounts.primary.set'),
				onclick: () => void makePrimary()
			});
		}
		actions.push(
			{
				icon: 'archive',
				label:
					acc.status === 'active' ? $_('accounts.action.archive') : $_('accounts.action.unarchive'),
				onclick: () => void toggleArchive()
			},
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void remove()
			}
		);
		return actions;
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

	function openEdit(tx: Transaction) {
		if (tx.credit_payment_linked) return;
		if (tx.type === 'transfer' && tx.transfer_group_id) {
			void goto(resolve(transferEditPath(tx.transfer_group_id, accountFrom)));
			return;
		}
		void goto(resolve(transactionEditPath(tx.id, accountFrom)));
	}

	function openRepeat(tx: Transaction) {
		if (tx.credit_payment_linked) return;
		if (tx.type === 'transfer' && tx.transfer_group_id) {
			void goto(resolve(transferNewPath({ repeatId: tx.id, from: accountFrom })));
			return;
		}
		void goto(
			resolve(
				transactionNewPath({
					type: tx.type === 'income' ? 'income' : 'expense',
					accountId: id,
					repeatId: tx.id,
					from: accountFrom
				})
			)
		);
	}

	function openMakeRecurring(tx: Transaction) {
		void goto(resolve(`/settings/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`));
	}

	async function resetFilters() {
		fromLocal = '';
		toLocal = '';
		typeFilter = '';
		categoryFilter = '';
		kindFilter = '';
		searchFilter = '';
		txPage = 1;
		lastFiltersKey = currentFiltersKey();
		await applyURLFilters();
	}

	async function onPageChange(nextPage: number) {
		txPage = nextPage;
		await applyURLFilters();
	}
</script>

<div class="space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/accounts', label: $_('accounts.title') },
			{ href: '/accounts', label: acc?.name ?? $_('common.loading') }
		]}
	/>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if acc}
			<div class="card">
				<div class="flex items-start gap-4">
					<AccountIcon type={acc.type} bankIcon={acc.bank_icon} size={56} />
					<div class="min-w-0 flex-1">
						{#if editing && acc.status !== 'deleted'}
							<form class="space-y-3" onsubmit={save}>
								<div>
									<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="acc-name">
										{$_('accounts.field.name')}
									</label>
									<input
										id="acc-name"
										class="input w-full"
										bind:value={name}
										required
										maxlength="64"
									/>
								</div>
								{#if acc.type === 'bank' || acc.type === 'credit_card'}
									<Select
										label={$_('accounts.field.bank')}
										bind:value={bankId}
										options={bankOptions}
										usePortal
									/>
								{/if}
								{#if acc.type === 'credit_card'}
									<div>
										<label
											class="mb-1 block text-sm"
											style:color="var(--text-muted)"
											for="acc-credit-limit"
										>
											{$_('accounts.field.creditLimit')}
										</label>
										<MoneyInput id="acc-credit-limit" bind:value={creditLimit} />
									</div>
									<Select
										label={$_('accounts.field.paymentAccount')}
										bind:value={paymentAccountId}
										options={[
											{ value: '', label: $_('accounts.creditCard.paymentAccountDefault') },
											...debitPaymentOptions
										]}
										usePortal
									/>
								{/if}
								<div>
									<label
										class="mb-1 block text-sm"
										style:color="var(--text-muted)"
										for="acc-balance"
									>
										{$_('accounts.field.balance')}
									</label>
									<MoneyInput id="acc-balance" bind:value={initialBalance} />
									{#if acc.type === 'credit_card'}
										<button
											type="button"
											class="btn-ghost mt-1 text-sm"
											onclick={applyLimitToBalance}
										>
											{$_('accounts.creditCard.limitButton')}
										</button>
									{/if}
								</div>
								<div class="flex gap-2">
									<button type="submit" class="btn-primary" disabled={saving}>
										{saving ? $_('common.loading') : $_('common.save')}
									</button>
									<button type="button" class="btn-ghost" onclick={() => (editing = false)}>
										{$_('common.cancel')}
									</button>
								</div>
							</form>
						{:else}
							<div class="flex items-start justify-between gap-2">
								<div class="min-w-0">
									{#if acc.status === 'archived' || acc.status === 'deleted'}
										<p class="mb-1 text-sm" style:color="var(--text-muted)">
											{acc.status === 'archived'
												? $_('accounts.banner.archived')
												: $_('accounts.banner.deleted')}
										</p>
									{/if}
									<div class="flex items-center gap-2">
										<h1 class="text-2xl font-semibold">{acc.name}</h1>
										{#if acc.is_primary}
											<span
												class="shrink-0"
												style:color="var(--primary)"
												title={$_('accounts.primary.badge')}
												aria-label={$_('accounts.primary.badge')}
											>
												<svg
													aria-hidden="true"
													class="h-5 w-5"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
												>
													<path d="M20 6 9 17l-5-5" />
												</svg>
											</span>
										{/if}
									</div>
									<p class="mt-1 text-3xl font-semibold tabular-nums">
										<MoneyDisplay
											value={accBalance?.balance_display ?? acc.balance_display}
											currency={$user?.currency ?? 'RUB'}
											class=""
										/>
									</p>
									{#if acc.credit_limit_display}
										<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
											{$_('accounts.field.creditLimit')}:
											<MoneyDisplay
												value={acc.credit_limit_display}
												currency={$user?.currency ?? 'RUB'}
												class=""
											/>
										</p>
									{/if}
									{#if acc.type === 'bank'}
										{@const autoTopupSource = resolveAutoTopupSourceName(acc, allAccounts)}
										{#if autoTopupSource}
											<p class="mt-1 text-sm" style:color="var(--text-muted)">
												{$_('accounts.autoTopup.status', { values: { source: autoTopupSource } })}
											</p>
										{/if}
									{/if}
									{#if accBalance ? accBalance.forecast_balance !== accBalance.balance : false}
										<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
											{$_('dashboard.withPlans')}:
											<MoneyDisplay
												value={accBalance?.forecast_display ?? acc.balance_display}
												currency={$user?.currency ?? 'RUB'}
												class=""
											/>
										</p>
									{/if}
								</div>
								{#if acc.status === 'active'}
									<div class="flex shrink-0 items-center gap-1">
										<div class="hidden items-center gap-1 md:flex">
											<NewTransactionButtons
												onincome={() => openNewTransaction('income')}
												onexpense={() => openNewTransaction('expense')}
												ontransfer={openTransfer}
											/>
										</div>
										<div class="md:hidden">
											<RowActionsMenu actions={accountActions(true)} />
										</div>
										<div class="hidden md:block">
											<RowActionsMenu actions={accountActions(false)} />
										</div>
									</div>
								{/if}
							</div>
						{/if}
					</div>
				</div>
			</div>

			<div class="relative space-y-3">
				<TransactionFilters
					bind:fromLocal
					bind:toLocal
					accountId=""
					accounts={[]}
					{categories}
					showAccount={false}
					expandSearchToEnd={true}
					onreset={resetFilters}
					bind:type={typeFilter}
					bind:categoryId={categoryFilter}
					bind:kind={kindFilter}
					bind:search={searchFilter}
				/>

				<TransactionContextStats params={accountStatsContextParams()} transactionCount={txTotal} />

				<div class:opacity-60={filterLoading} class="card md:overflow-x-auto">
					<TransactionList
						transactions={visibleTx}
						siblings={transactions}
						{tz}
						emptyMessage={$_('transactions.empty')}
						showDescription
						showAmountSign
						singleAccount
						showEdit={!accountTxReadOnly}
						showDelete={!accountTxReadOnly}
						onrepeat={openRepeat}
						onmakeRecurring={openMakeRecurring}
						onedit={accountTxReadOnly ? undefined : openEdit}
						ondelete={accountTxReadOnly ? undefined : (tx) => void removeTx(tx)}
					/>
				</div>
				{#if filterLoading}
					<p
						class="absolute inset-0 flex items-center justify-center text-sm"
						style:color="var(--text-muted)"
					>
						{$_('common.loading')}
					</p>
				{/if}
			</div>
			<TransactionPagination
				page={txPage}
				limit={txLimit}
				total={txTotal}
				onchange={onPageChange}
			/>
		{/if}
	</PageLoadGate>
</div>
