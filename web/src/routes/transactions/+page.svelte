<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page as pageStore } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		deleteTransaction,
		getUIMeta,
		listTransactions,
		type Account,
		type Category,
		type Transaction
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import NewTransactionButtons from '$lib/components/NewTransactionButtons.svelte';
	import TransactionContextStats from '$lib/components/TransactionContextStats.svelte';
	import TransactionFilters from '$lib/components/TransactionFilters.svelte';
	import TransactionForm from '$lib/components/TransactionForm.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import TransactionPagination from '$lib/components/TransactionPagination.svelte';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import { fromDateLocalEnd, fromDateLocalStart } from '$lib/dates';
	import { dedupeTransferLegs } from '$lib/transaction-display';

	let transactions = $state<Transaction[]>([]);
	let total = $state(0);
	let page = $state(1);
	const limit = 50;
	let loading = $state(true);
	let filterLoading = $state(false);
	let error = $state('');
	let txOpen = $state(false);
	let transferOpen = $state(false);
	let editTx = $state<Transaction | null>(null);
	let editTransfer = $state<Transaction | null>(null);
	let newTxType = $state<'expense' | 'income'>('expense');
	let accounts = $state<Account[]>([]);
	let categories = $state<Category[]>([]);

	let fromLocal = $state('');
	let toLocal = $state('');
	let type = $state('');
	let categoryId = $state('');
	let accountId = $state('');
	let kind = $state('');
	let search = $state('');
	let filtersAutoApplyReady = $state(false);
	let lastFiltersKey = $state('');

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const visibleTx = $derived(dedupeTransferLegs(transactions));

	onMount(async () => {
		readURLState();
		await Promise.all([loadFilterOptions(), load(true)]);
		lastFiltersKey = currentFiltersKey();
		filtersAutoApplyReady = true;
	});

	$effect(() => {
		const nextKey = currentFiltersKey();
		if (!filtersAutoApplyReady) return;
		if (nextKey === lastFiltersKey) return;
		lastFiltersKey = nextKey;
		page = 1;
		void pushURLAndReload();
	});

	function readURLState() {
		const q = $pageStore.url.searchParams;
		page = Number(q.get('page') || '1');
		fromLocal = q.get('from_local') ?? '';
		toLocal = q.get('to_local') ?? '';
		type = q.get('type') ?? '';
		categoryId = q.get('category_id') ?? '';
		accountId = q.get('account_id') ?? '';
		kind = q.get('kind') ?? '';
		search = q.get('search') ?? '';
	}

	function statsContextParams() {
		const params: Record<string, string> = {};
		if (fromLocal) params.from = fromDateLocalStart(fromLocal, tz);
		if (toLocal) params.to = fromDateLocalEnd(toLocal, tz);
		if (type) params.type = type;
		if (categoryId) params.category_id = categoryId;
		if (accountId) params.account_id = accountId;
		if (kind) params.kind = kind;
		if (search.trim()) params.search = search.trim();
		return params;
	}

	function requestParams() {
		return { ...statsContextParams(), page: String(page), limit: String(limit) };
	}

	function currentFiltersKey(): string {
		return JSON.stringify({
			fromLocal,
			toLocal,
			type,
			categoryId,
			accountId,
			kind,
			search: search.trim()
		});
	}

	async function loadFilterOptions() {
		const meta = await getUIMeta();
		accounts = meta.accounts
			.filter((acc) => acc.status === 'active')
			.map((acc) => ({ id: acc.id, name: acc.name }) as Account);
		const uniqueByID: Record<string, Category> = {};
		for (const cat of [...meta.expense_categories, ...meta.income_categories]) {
			uniqueByID[cat.id] = cat;
		}
		categories = Object.values(uniqueByID).sort((a, b) => a.name.localeCompare(b.name, 'ru'));
	}

	async function load(initial = false) {
		if (initial) loading = true;
		else filterLoading = true;
		error = '';
		try {
			const result = await listTransactions(requestParams());
			transactions = result.data;
			total = result.meta.total;
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
			filterLoading = false;
		}
	}

	async function pushURLAndReload() {
		const basePath = resolve('/transactions');
		const queryParts = [`page=${encodeURIComponent(String(page))}`];
		if (fromLocal) queryParts.push(`from_local=${encodeURIComponent(fromLocal)}`);
		if (toLocal) queryParts.push(`to_local=${encodeURIComponent(toLocal)}`);
		if (type) queryParts.push(`type=${encodeURIComponent(type)}`);
		if (categoryId) queryParts.push(`category_id=${encodeURIComponent(categoryId)}`);
		if (accountId) queryParts.push(`account_id=${encodeURIComponent(accountId)}`);
		if (kind) queryParts.push(`kind=${encodeURIComponent(kind)}`);
		if (search.trim()) queryParts.push(`search=${encodeURIComponent(search.trim())}`);
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query string is appended to resolved base path
		await goto(`${basePath}?${queryParts.join('&')}`, {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
		await load();
	}

	async function resetFilters() {
		fromLocal = '';
		toLocal = '';
		type = '';
		categoryId = '';
		accountId = '';
		kind = '';
		search = '';
		page = 1;
		lastFiltersKey = currentFiltersKey();
		await pushURLAndReload();
	}

	async function onPageChange(nextPage: number) {
		page = nextPage;
		await pushURLAndReload();
	}

	function openNewTransaction(type: 'expense' | 'income') {
		editTx = null;
		newTxType = type;
		txOpen = true;
	}

	function openEdit(tx: Transaction) {
		if (tx.credit_payment_linked) return;
		if (tx.type === 'transfer' && tx.transfer_group_id) {
			editTransfer = tx;
			editTx = null;
			transferOpen = true;
			return;
		}
		editTransfer = null;
		editTx = tx;
		txOpen = true;
	}

	async function removeTx(tx: Transaction) {
		const ok = await confirm({
			message: $_('transactions.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
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
	<title>{$_('transactions.all')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-4">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/transactions', label: $_('transactions.all') }
		]}
	/>

	<div class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-semibold">{$_('transactions.all')}</h1>
		<div class="flex shrink-0 items-center gap-1">
			<NewTransactionButtons
				onincome={() => openNewTransaction('income')}
				onexpense={() => openNewTransaction('expense')}
				ontransfer={() => {
					editTransfer = null;
					transferOpen = true;
				}}
			/>
		</div>
	</div>

	<TransactionFilters
		bind:fromLocal
		bind:toLocal
		bind:type
		bind:categoryId
		bind:accountId
		bind:kind
		bind:search
		{accounts}
		{categories}
		onreset={resetFilters}
	/>

	<TransactionContextStats params={statsContextParams()} />

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if error}
		<p style:color="var(--danger)">{error}</p>
	{:else}
		<div class="relative space-y-3">
			<div class="card md:overflow-x-auto" class:opacity-60={filterLoading}>
				<TransactionList
					transactions={visibleTx}
					siblings={transactions}
					{tz}
					emptyMessage={$_('transactions.empty')}
					showEdit
					showDelete
					onmakeRecurring={(tx) =>
						void goto(resolve(`/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`))}
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
		<TransactionPagination {page} {limit} {total} onchange={onPageChange} />
	{/if}
</div>

<TransactionForm
	bind:open={txOpen}
	defaultType={newTxType}
	transaction={editTx}
	onclose={() => {
		txOpen = false;
		editTx = null;
	}}
	onsaved={load}
/>
<TransferForm
	bind:open={transferOpen}
	editTx={editTransfer}
	siblings={transactions}
	onclose={() => {
		transferOpen = false;
		editTransfer = null;
	}}
	onsaved={load}
/>
