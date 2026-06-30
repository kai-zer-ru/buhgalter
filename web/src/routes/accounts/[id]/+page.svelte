<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		archiveAccount,
		deleteAccount,
		deleteTransaction,
		getAccount,
		getAccountBalance,
		listBanks,
		listCategories,
		listTransactions,
		setPrimaryAccount,
		unarchiveAccount,
		updateAccount,
		type Account,
		type AccountBalanceSummary,
		type Bank,
		type Category,
		type Transaction
	} from '$lib/api/client';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import BackLink from '$lib/components/BackLink.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import TransactionFilters from '$lib/components/TransactionFilters.svelte';
	import TransactionForm from '$lib/components/TransactionForm.svelte';
	import NewTransactionButtons from '$lib/components/NewTransactionButtons.svelte';
	import TransactionContextStats from '$lib/components/TransactionContextStats.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import TransactionPagination from '$lib/components/TransactionPagination.svelte';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import { formatBalance } from '$lib/finance';
	import { formatMoneyForInput, toAPIAmount } from '$lib/money';
	import { fromDateLocalEnd, fromDateLocalStart } from '$lib/dates';
	import { dedupeTransferLegs } from '$lib/transaction-display';
	import { user } from '$lib/stores/auth';

	let acc = $state<Account | null>(null);
	let accBalance = $state<AccountBalanceSummary | null>(null);
	let banks = $state<Bank[]>([]);
	let categories = $state<Category[]>([]);
	let transactions = $state<Transaction[]>([]);
	let txTotal = $state(0);
	let txPage = $state(1);
	const txLimit = 20;
	let editing = $state(false);
	let name = $state('');
	let bankId = $state('');
	let initialBalance = $state('');
	let loading = $state(true);
	let filterLoading = $state(false);
	let saving = $state(false);
	let error = $state('');
	let txOpen = $state(false);
	let transferOpen = $state(false);
	let editTx = $state<Transaction | null>(null);
	let editTransfer = $state<Transaction | null>(null);
	let repeatTx = $state<Transaction | null>(null);
	let repeatTransfer = $state<Transaction | null>(null);
	let newTxType = $state<'expense' | 'income'>('expense');
	let fromLocal = $state('');
	let toLocal = $state('');
	let typeFilter = $state('');
	let categoryFilter = $state('');
	let kindFilter = $state('');
	let searchFilter = $state('');
	let filtersAutoApplyReady = $state(false);
	let lastFiltersKey = $state('');

	const id = $derived($page.params.id ?? '');
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const visibleTx = $derived(dedupeTransferLegs(transactions));
	const bankOptions = $derived(banks.map((bank) => ({ value: bank.id, label: bank.name })));

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

	async function load() {
		if (!id) return;
		loading = true;
		error = '';
		try {
			const [account, accountBalance, bankList, expenseCats, incomeCats] = await Promise.all([
				getAccount(id),
				getAccountBalance(id),
				listBanks(),
				listCategories('expense'),
				listCategories('income')
			]);
			acc = account;
			accBalance = accountBalance;
			banks = bankList;
			const uniqueCatsByID: Record<string, Category> = {};
			for (const cat of [...expenseCats, ...incomeCats]) uniqueCatsByID[cat.id] = cat;
			categories = Object.values(uniqueCatsByID).sort((a, b) => a.name.localeCompare(b.name, 'ru'));
			name = account.name;
			bankId = account.bank_id ?? '';
			initialBalance = formatMoneyForInput(account.balance_display);
			editing = $page.url.searchParams.get('edit') === '1';
			await loadTransactions();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}

	async function loadTransactions() {
		if (!id) return;
		filterLoading = true;
		try {
			const params: Record<string, string> = {
				account_id: id,
				page: String(txPage),
				limit: String(txLimit)
			};
			if (fromLocal) params.from = fromDateLocalStart(fromLocal, tz);
			if (toLocal) params.to = fromDateLocalEnd(toLocal, tz);
			if (typeFilter) params.type = typeFilter;
			if (categoryFilter) params.category_id = categoryFilter;
			if (kindFilter) params.kind = kindFilter;
			if (searchFilter.trim()) params.search = searchFilter.trim();
			const result = await listTransactions(params);
			transactions = result.data;
			txTotal = result.meta.total;
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			filterLoading = false;
		}
	}

	async function applyURLFilters() {
		const basePath = resolve(`/accounts/${id}`);
		const queryParts = [`page=${encodeURIComponent(String(txPage))}`];
		if (fromLocal) queryParts.push(`from_local=${encodeURIComponent(fromLocal)}`);
		if (toLocal) queryParts.push(`to_local=${encodeURIComponent(toLocal)}`);
		if (typeFilter) queryParts.push(`type=${encodeURIComponent(typeFilter)}`);
		if (categoryFilter) queryParts.push(`category_id=${encodeURIComponent(categoryFilter)}`);
		if (kindFilter) queryParts.push(`kind=${encodeURIComponent(kindFilter)}`);
		if (searchFilter.trim()) queryParts.push(`search=${encodeURIComponent(searchFilter.trim())}`);
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query string is appended to resolved base path
		await goto(`${basePath}?${queryParts.join('&')}`, {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
		await loadTransactions();
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!acc) return;
		saving = true;
		error = '';
		try {
			acc = await updateAccount(acc.id, {
				name,
				bank_id: acc.type === 'bank' ? bankId : undefined,
				initial_balance: toAPIAmount(initialBalance)
			});
			accBalance = await getAccountBalance(acc.id);
			editing = false;
			toast($_('common.saved'));
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			saving = false;
		}
	}

	async function toggleArchive() {
		if (!acc) return;
		try {
			acc = acc.status === 'active' ? await archiveAccount(acc.id) : await unarchiveAccount(acc.id);
			accBalance = await getAccountBalance(acc.id);
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		}
	}

	async function makePrimary() {
		if (!acc || acc.is_primary) return;
		try {
			acc = await setPrimaryAccount(acc.id);
			toast($_('common.saved'));
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		}
	}

	async function remove() {
		if (!acc) return;
		const ok = await confirm({
			message: $_('accounts.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteAccount(acc.id);
			toast($_('common.deleted'));
			await goto(resolve('/accounts'));
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		}
	}

	function openNewTransaction(type: 'expense' | 'income') {
		editTx = null;
		repeatTx = null;
		newTxType = type;
		txOpen = true;
	}

	function accountActions(includeTransactions = false): RowAction[] {
		if (!acc) return [];
		const actions: RowAction[] = [];
		if (includeTransactions) {
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
					onclick: () => {
						editTransfer = null;
						repeatTransfer = null;
						transferOpen = true;
					}
				}
			);
		}
		actions.push({
			icon: 'edit',
			label: $_('accounts.action.edit'),
			onclick: () => (editing = true)
		});
		if (!acc.is_primary) {
			actions.push({
				icon: 'save',
				label: $_('accounts.primary.set'),
				onclick: () => void makePrimary()
			});
		}
		actions.push(
			{
				icon: 'archive',
				label: $_('accounts.action.archive'),
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
			await deleteTransaction(tx.id);
			toast($_('common.deleted'));
			await load();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		}
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

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if !acc}
		<p style:color="var(--danger)">{error || $_('common.error')}</p>
	{:else}
		<div class="card">
			<div class="flex items-start gap-4">
				<AccountIcon type={acc.type} bankIcon={acc.bank_icon} size={56} />
				<div class="min-w-0 flex-1">
					{#if editing}
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
							{#if acc.type === 'bank'}
								<Select
									label={$_('accounts.field.bank')}
									bind:value={bankId}
									options={bankOptions}
									usePortal
								/>
							{/if}
							<div>
								<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="acc-balance">
									{$_('accounts.field.balance')}
								</label>
								<MoneyInput id="acc-balance" bind:value={initialBalance} />
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
									{formatBalance(
										accBalance?.balance_display ?? acc.balance_display,
										$user?.currency ?? 'RUB'
									)}
								</p>
								{#if accBalance ? accBalance.forecast_balance !== accBalance.balance : false}
									<p class="mt-1 text-sm tabular-nums" style:color="var(--text-muted)">
										{$_('dashboard.withPlans')}:
										{formatBalance(
											accBalance?.forecast_display ?? acc.balance_display,
											$user?.currency ?? 'RUB'
										)}
									</p>
								{/if}
							</div>
							{#if acc.status === 'active'}
								<div class="flex shrink-0 items-center gap-1">
									<div class="hidden items-center gap-1 md:flex">
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

			{#if error}
				<p class="mt-3 text-sm" style:color="var(--danger)">{error}</p>
			{/if}
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

			<TransactionContextStats
				params={{
					account_id: id,
					from: fromLocal ? fromDateLocalStart(fromLocal, tz) : '',
					to: toLocal ? fromDateLocalEnd(toLocal, tz) : '',
					type: typeFilter,
					category_id: categoryFilter,
					kind: kindFilter,
					search: searchFilter
				}}
			/>

			<div class:opacity-60={filterLoading} class="card md:overflow-x-auto">
				<TransactionList
					transactions={visibleTx}
					siblings={transactions}
					{tz}
					emptyMessage={$_('transactions.empty')}
					showDescription
					showAmountSign
					singleAccount
					showEdit
					showDelete
					onmakeRecurring={(tx) =>
						void goto(resolve(`/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`))}
					onrepeat={openRepeat}
					onedit={openEdit}
					ondelete={(tx) => void removeTx(tx)}
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
		<TransactionPagination page={txPage} limit={txLimit} total={txTotal} onchange={onPageChange} />
	{/if}
</div>

<TransactionForm
	bind:open={txOpen}
	accountId={id}
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
	accountId={id}
	editTx={editTransfer}
	repeatFrom={repeatTransfer}
	siblings={transactions}
	onclose={() => {
		transferOpen = false;
		editTransfer = null;
		repeatTransfer = null;
	}}
	onsaved={load}
/>
