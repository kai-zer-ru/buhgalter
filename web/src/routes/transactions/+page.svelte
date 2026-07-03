<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page as pageStore } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		deleteTransaction,
		getUIMeta,
		listTransactions,
		type Account,
		type Category,
		type Transaction
	} from '$lib/api/client';
	import { accountsFromUIMeta } from '$lib/select-options';
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
	let pastTx = $state<Transaction[]>([]);
	let pastTotal = $state(0);
	let pastLoading = $state(false);
	let plannedTx = $state<Transaction[]>([]);
	let plannedTotal = $state(0);
	let plannedLoading = $state(false);
	let page = $state(1);
	const limit = 20;
	let loading = $state(true);
	let filterLoading = $state(false);
	let txOpen = $state(false);
	let transferOpen = $state(false);
	let editTx = $state<Transaction | null>(null);
	let editTransfer = $state<Transaction | null>(null);
	let repeatTx = $state<Transaction | null>(null);
	let repeatTransfer = $state<Transaction | null>(null);
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
	const splitMode = $derived(!kind);
	const visibleTx = $derived(dedupeTransferLegs(transactions));
	const pastVisible = $derived(dedupeTransferLegs(pastTx));
	const plannedVisible = $derived(dedupeTransferLegs(plannedTx));
	const txSiblings = $derived(splitMode ? [...pastTx, ...plannedTx] : transactions);
	const listTransactionCount = $derived(splitMode ? plannedTotal + pastTotal : total);

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

	function baseFilterParams() {
		const params: Record<string, string> = {};
		if (fromLocal) params.from = fromDateLocalStart(fromLocal, tz);
		if (toLocal) params.to = fromDateLocalEnd(toLocal, tz);
		if (type) params.type = type;
		if (categoryId) params.category_id = categoryId;
		if (accountId) params.account_id = accountId;
		if (search.trim()) params.search = search.trim();
		return params;
	}

	function statsContextParams() {
		const params = baseFilterParams();
		if (kind) {
			params.kind = kind;
		} else {
			params.include_future = 'true';
		}
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
		accounts = accountsFromUIMeta(
			meta.accounts.filter((acc) => acc.status === 'active'),
			meta.banks
		) as Account[];
		const uniqueByID: Record<string, Category> = {};
		for (const cat of [...meta.expense_categories, ...meta.income_categories]) {
			uniqueByID[cat.id] = cat;
		}
		categories = Object.values(uniqueByID).sort((a, b) => a.name.localeCompare(b.name, 'ru'));
	}

	async function loadSplit() {
		const base = baseFilterParams();
		pastLoading = true;
		plannedLoading = true;
		try {
			const [pastRes, plannedRes] = await Promise.all([
				listTransactions({
					...base,
					kind: 'manual',
					sort: 'date_desc',
					page: String(page),
					limit: String(limit)
				}),
				listTransactions({
					...base,
					kind: 'future',
					sort: 'date_desc',
					page: '1',
					limit: String(limit)
				})
			]);
			pastTx = pastRes.data;
			pastTotal = pastRes.meta.total;
			plannedTx = plannedRes.data;
			plannedTotal = plannedRes.meta.total;
		} catch (err) {
			toast.fromError(err);
		} finally {
			pastLoading = false;
			plannedLoading = false;
		}
	}

	async function loadFlat() {
		try {
			const result = await listTransactions(requestParams());
			transactions = result.data;
			total = result.meta.total;
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function load(initial = false) {
		if (initial) loading = true;
		else filterLoading = true;
		try {
			if (splitMode) {
				await loadSplit();
			} else {
				await loadFlat();
			}
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
			toast.fromError(err);
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
					repeatTransfer = null;
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

	<TransactionContextStats params={statsContextParams()} transactionCount={listTransactionCount} />

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if splitMode}
		<div class="relative space-y-3">
			<div class="card overflow-hidden" class:opacity-60={filterLoading}>
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
		<TransactionPagination {page} {limit} total={pastTotal} onchange={onPageChange} />
	{:else}
		<div class="relative space-y-3">
			<div class="card md:overflow-x-auto" class:opacity-60={filterLoading}>
				<TransactionList
					transactions={visibleTx}
					siblings={transactions}
					{tz}
					emptyMessage={$_('transactions.empty')}
					showDescription
					showEdit
					showDelete
					onmakeRecurring={(tx) =>
						void goto(
							resolve(`/settings/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`)
						)}
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
		<TransactionPagination {page} {limit} {total} onchange={onPageChange} />
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
	siblings={txSiblings}
	onclose={() => {
		transferOpen = false;
		editTransfer = null;
		repeatTransfer = null;
	}}
	onsaved={load}
/>
