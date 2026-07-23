<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page as pageStore } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		getUIMeta,
		listTransactions,
		type Account,
		type Category,
		type Transaction
	} from '$lib/api/client';
	import { deleteTransaction, deleteTransfer } from '$lib/offline/transactions-api';
	import { mergeOutboxTransactions, refreshMergeMeta } from '$lib/offline/merge';
	import { outboxTick } from '$lib/offline/store';
	import { dataRefreshTick, localDataTick, scheduleSyncOutbox } from '$lib/offline/sync';
	import { refCacheReady, refCacheUpdate } from '$lib/offline/ref-cache';
	import { refCachePathMatches } from '$lib/offline/ref-cache-watch';
	import { assignIfChanged } from '$lib/state-utils';
	import { accountsFromUIMeta } from '$lib/select-options';
	import BackLink from '$lib/components/BackLink.svelte';
	import NewTransactionButtons from '$lib/components/NewTransactionButtons.svelte';
	import { resolveAppPath } from '$lib/android/form-nav';
	import {
		transactionEditPath,
		transactionNewPath,
		transferEditPath,
		transferNewPath
	} from '$lib/android/form-routes';
	import TransactionContextStats from '$lib/components/TransactionContextStats.svelte';
	import TransactionFilters from '$lib/components/TransactionFilters.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import TransactionPagination from '$lib/components/TransactionPagination.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { confirm } from '$lib/confirm';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import { fromDateLocalEnd, fromDateLocalStart } from '$lib/dates';
	import { dedupeTransferLegs } from '$lib/transaction-display';

	let serverTransactions = $state<Transaction[]>([]);
	let total = $state(0);
	let serverPastTx = $state<Transaction[]>([]);
	let pastTotal = $state(0);
	let pastLoading = $state(false);
	let serverPlannedTx = $state<Transaction[]>([]);
	let plannedTotal = $state(0);
	let plannedLoading = $state(false);
	let page = $state(1);
	const limit = 20;
	let loading = $state(!refCacheReady('/api/v1/ui/meta'));
	let filterLoading = $state(false);
	let loadError = $state<string | null>(null);

	const listFrom = '/transactions';
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
	const transactions = $derived.by(() => {
		void $outboxTick;
		void $localDataTick;
		return mergeOutboxTransactions(serverTransactions);
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
	const visibleTx = $derived(dedupeTransferLegs(transactions));
	const pastVisible = $derived(dedupeTransferLegs(pastTx));
	const plannedVisible = $derived(dedupeTransferLegs(plannedTx));
	const txSiblings = $derived(splitMode ? [...pastTx, ...plannedTx] : transactions);
	const listTransactionCount = $derived(splitMode ? plannedTotal + pastTotal : total);

	$effect(() => {
		const tick = $dataRefreshTick;
		if (tick === 0 || !filtersAutoApplyReady) return;
		void load(false, { background: true });
	});

	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || !filtersAutoApplyReady) return;
		if (
			refCachePathMatches(update.path, '/api/v1/ui/meta') ||
			refCachePathMatches(update.path, '/api/v1/transactions')
		) {
			void load(false, { background: true });
		}
	});

	onMount(async () => {
		readURLState();
		await refreshMergeMeta().catch(() => undefined);
		await Promise.all([loadFilterOptions(), load(true)]);
		scheduleSyncOutbox();
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

	async function loadFilterOptions(opts: { background?: boolean } = {}) {
		try {
			const meta = await getUIMeta();
			const nextAccounts = accountsFromUIMeta(
				meta.accounts.filter((acc) => acc.status === 'active'),
				meta.banks
			) as Account[];
			const uniqueByID: Record<string, Category> = {};
			for (const cat of [...meta.expense_categories, ...meta.income_categories]) {
				uniqueByID[cat.id] = cat;
			}
			const nextCategories = Object.values(uniqueByID).sort((a, b) =>
				a.name.localeCompare(b.name, 'ru')
			);
			accounts = opts.background ? assignIfChanged(accounts, nextAccounts) : nextAccounts;
			categories = opts.background ? assignIfChanged(categories, nextCategories) : nextCategories;
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, {
				background: opts.background,
				hasData: categories.length > 0
			});
			if (msg) loadError = msg;
		}
	}

	async function loadSplit(opts: { background?: boolean } = {}) {
		const base = baseFilterParams();
		if (!opts.background) {
			pastLoading = true;
			plannedLoading = true;
		}
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
			const nextPast = pastRes.data;
			serverPastTx = opts.background ? assignIfChanged(serverPastTx, nextPast) : nextPast;
			pastTotal = opts.background
				? assignIfChanged(pastTotal, pastRes.meta.total)
				: pastRes.meta.total;
			const nextPlanned = plannedRes.data;
			serverPlannedTx = opts.background
				? assignIfChanged(serverPlannedTx, nextPlanned)
				: nextPlanned;
			plannedTotal = opts.background
				? assignIfChanged(plannedTotal, plannedRes.meta.total)
				: plannedRes.meta.total;
			loadError = null;
		} catch (err) {
			const hasData = pastTotal > 0 || plannedTotal > 0;
			const msg = reportPageLoadFailure(err, { background: opts.background, hasData });
			if (msg) loadError = msg;
		} finally {
			pastLoading = false;
			plannedLoading = false;
		}
	}

	async function loadFlat(opts: { background?: boolean } = {}) {
		try {
			const result = await listTransactions(requestParams());
			const nextData = result.data;
			serverTransactions = opts.background
				? assignIfChanged(serverTransactions, nextData)
				: nextData;
			total = opts.background ? assignIfChanged(total, result.meta.total) : result.meta.total;
			loadError = null;
		} catch (err) {
			const hasData = total > 0 || serverTransactions.length > 0;
			const msg = reportPageLoadFailure(err, { background: opts.background, hasData });
			if (msg) loadError = msg;
		}
	}

	async function load(initial = false, opts: { background?: boolean } = {}) {
		if (initial && !opts.background && !refCacheReady('/api/v1/ui/meta')) loading = true;
		else if (!initial && !opts.background) filterLoading = true;
		try {
			if (splitMode) {
				await loadSplit(opts);
			} else {
				await loadFlat(opts);
			}
			scheduleSyncOutbox();
		} finally {
			loading = false;
			filterLoading = false;
		}
	}

	async function pushURLAndReload() {
		const queryParts = [`page=${encodeURIComponent(String(page))}`];
		if (fromLocal) queryParts.push(`from_local=${encodeURIComponent(fromLocal)}`);
		if (toLocal) queryParts.push(`to_local=${encodeURIComponent(toLocal)}`);
		if (type) queryParts.push(`type=${encodeURIComponent(type)}`);
		if (categoryId) queryParts.push(`category_id=${encodeURIComponent(categoryId)}`);
		if (accountId) queryParts.push(`account_id=${encodeURIComponent(accountId)}`);
		if (kind) queryParts.push(`kind=${encodeURIComponent(kind)}`);
		if (search.trim()) queryParts.push(`search=${encodeURIComponent(search.trim())}`);
		await goto(resolveAppPath(`/transactions?${queryParts.join('&')}`), {
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
		void goto(resolve(transactionNewPath({ type, from: listFrom })));
	}

	function openEdit(tx: Transaction) {
		if (tx.credit_payment_linked) return;
		if (tx.type === 'transfer' && tx.transfer_group_id) {
			void goto(resolve(transferEditPath(tx.transfer_group_id, listFrom)));
			return;
		}
		void goto(resolve(transactionEditPath(tx.id, listFrom)));
	}

	function openRepeat(tx: Transaction) {
		if (tx.credit_payment_linked) return;
		if (tx.type === 'transfer' && tx.transfer_group_id) {
			void goto(resolve(transferNewPath({ repeatId: tx.id, from: listFrom })));
			return;
		}
		void goto(
			resolve(
				transactionNewPath({
					type: tx.type === 'income' ? 'income' : 'expense',
					repeatId: tx.id,
					from: listFrom
				})
			)
		);
	}

	function openMakeRecurring(tx: Transaction) {
		void goto(resolve(`/settings/recurring-operations?from_tx=${encodeURIComponent(tx.id)}`));
	}

	async function removeTx(tx: Transaction) {
		const ok = await confirm({
			message: $_('transactions.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
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
				ontransfer={() => void goto(resolve(transferNewPath({ from: listFrom })))}
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

	<PageLoadGate {loading} error={loadError} onretry={() => void load(false)} inline>
		{#if splitMode}
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
						showAmountSign
						showEdit
						showDelete
						onrepeat={openRepeat}
						onmakeRecurring={openMakeRecurring}
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
	</PageLoadGate>
</div>
