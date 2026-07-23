<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import { budgetRemainingCell } from '$lib/budget-display';
	import {
		getStatsByCategory,
		getStatsBySubcategory,
		getStatsByPeriod,
		getStatsSummary,
		getBudgetSummary,
		getUIMeta,
		searchStats,
		type Account,
		type Category,
		type Credit,
		type Debtor,
		type StatsCategoryItem,
		type StatsSubcategoryItem,
		type StatsPeriodItem,
		type StatsSummary,
		type BudgetSummaryItem,
		type Transaction
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import EntityLink from '$lib/components/EntityLink.svelte';
	import FilterPanel from '$lib/components/FilterPanel.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import Select from '$lib/components/Select.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { formatMoneyForDisplay } from '$lib/money-display';
	import {
		categoryDisplayLabel,
		categorySelectLabel,
		duplicateCategoryNames
	} from '$lib/category-label';
	import {
		fromDateLocalEnd,
		fromDateLocalStart,
		dateOnlyLocalValue,
		todayDateLocal
	} from '$lib/dates';
	import { formatStatsPeriod } from '$lib/stats-period';
	import { groupSubcategoriesByCategory, subcategoryShareOfParent } from '$lib/stats-subcategory';
	import {
		accountSelectOptions,
		accountsFromUIMeta,
		categorySelectOptions
	} from '$lib/select-options';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { user } from '$lib/stores/auth';
	import { toast } from '$lib/toast';

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let filterLoading = $state(false);
	let summary = $state<StatsSummary | null>(null);
	let byCategory = $state<StatsCategoryItem[]>([]);
	let bySubcategory = $state<StatsSubcategoryItem[]>([]);
	let budgetByCategory = $state<Record<string, BudgetSummaryItem>>({});
	let byPeriod = $state<StatsPeriodItem[]>([]);
	let searchRows = $state<Transaction[]>([]);
	let accounts = $state<Account[]>([]);
	let categories = $state<Category[]>([]);
	let debtorByName = $state<Record<string, string>>({});
	let creditByName = $state<Record<string, string>>({});

	let fromLocal = $state('');
	let toLocal = $state('');
	let type = $state('');
	let accountId = $state('');
	let categoryId = $state('');
	let groupBy = $state<'day' | 'week' | 'month'>('month');
	let search = $state('');
	let filtersAutoApplyReady = $state(false);
	let lastFiltersKey = $state('');
	let metaLoaded = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');
	const typeOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: '', label: tr('transactions.filter.all') },
			{ value: 'expense', label: tr('transactions.type.expense') },
			{ value: 'income', label: tr('transactions.type.income') }
		];
	});
	const accountOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: '', label: tr('import.export.all_accounts') },
			...accountSelectOptions(accounts)
		];
	});
	const categoryOptions = $derived.by(() => {
		void $locale;
		const filtered =
			type === 'income' || type === 'expense'
				? categories.filter((cat) => cat.type === type)
				: categories;
		return [
			{ value: '', label: tr('import.export.all_categories') },
			...categorySelectOptions(filtered, (cat) => categorySelectLabel(cat, categories))
		];
	});
	const duplicateCategoryNameSet = $derived(
		duplicateCategoryNames(byCategory.map((row) => ({ name: row.category_name, type: row.type })))
	);
	const byCategoryIncome = $derived(
		byCategory
			.filter((row) => row.type === 'income')
			.sort((a, b) => b.total - a.total || a.category_name.localeCompare(b.category_name, 'ru'))
	);
	const byCategoryExpense = $derived(
		byCategory
			.filter((row) => row.type === 'expense')
			.sort((a, b) => b.total - a.total || a.category_name.localeCompare(b.category_name, 'ru'))
	);
	const subByCategory = $derived(groupSubcategoriesByCategory(bySubcategory));
	const showCategoryIncome = $derived(type !== 'expense');
	const showCategoryExpense = $derived(type !== 'income');

	function statsCategoryLabel(name: string, type: 'income' | 'expense'): string {
		return categoryDisplayLabel(name, type, duplicateCategoryNameSet);
	}
	const groupByOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: 'day', label: tr('stats.group.day') },
			{ value: 'week', label: tr('stats.group.week') },
			{ value: 'month', label: tr('stats.group.month') }
		];
	});
	const periodMax = $derived(
		Math.max(1, ...byPeriod.map((row) => Math.max(Math.abs(row.income), Math.abs(row.expense))))
	);

	function periodLabel(period: string): string {
		void $locale;
		return formatStatsPeriod(period, groupBy, $locale ?? 'ru');
	}

	onMount(async () => {
		await loadMeta();
		await loadStats(true);
		lastFiltersKey = currentFiltersKey();
		filtersAutoApplyReady = true;
	});

	$effect(() => {
		const nextKey = currentFiltersKey();
		if (!filtersAutoApplyReady) return;
		if (nextKey === lastFiltersKey) return;
		lastFiltersKey = nextKey;
		void loadStats(false);
	});

	$effect(() => {
		if (type !== 'income' && type !== 'expense') return;
		if (!categoryId) return;
		const selected = categories.find((cat) => cat.id === categoryId);
		if (selected && selected.type !== type) {
			categoryId = '';
		}
	});

	function currentFiltersKey(): string {
		return JSON.stringify({
			fromLocal,
			toLocal,
			type,
			accountId,
			categoryId,
			groupBy,
			search: search.trim()
		});
	}

	async function loadMeta() {
		try {
			const meta = await getUIMeta();
			accounts = accountsFromUIMeta(
				meta.accounts.filter((acc) => acc.status === 'active'),
				meta.banks
			) as Account[];
			const mergedCategories = [...meta.expense_categories, ...meta.income_categories];
			const uniqueByID: Record<string, Category> = {};
			for (const cat of mergedCategories) uniqueByID[cat.id] = cat;
			categories = Object.values(uniqueByID).sort((a, b) => a.name.localeCompare(b.name, 'ru'));
			debtorByName = toDebtorMap(meta.debtors);
			creditByName = toCreditMap([...meta.active_credits, ...meta.closed_credits]);
			metaLoaded = true;
		} catch (err) {
			toast.fromError(err);
		}
	}

	function statsParams() {
		const params: Record<string, string> = {};
		if (fromLocal) params.from = fromDateLocalStart(fromLocal, tz);
		if (toLocal) params.to = fromDateLocalEnd(toLocal, tz);
		if (type) params.type = type;
		if (accountId) params.account_id = accountId;
		if (categoryId) params.category_id = categoryId;
		return params;
	}

	function budgetMonthKey(): string {
		const raw = (toLocal || fromLocal || todayDateLocal(tz)).split('T')[0];
		const [y, m] = raw.split('-');
		return `${y}-${m}`;
	}

	async function loadStats(initial = false) {
		if (!metaLoaded && initial) {
			await loadMeta();
		}
		if (initial) loading = true;
		else filterLoading = true;
		try {
			const params = statsParams();
			const month = budgetMonthKey();
			const [summaryRes, categoryRes, subcategoryRes, periodRes, budgetRes] = await Promise.all([
				getStatsSummary(params),
				getStatsByCategory(params),
				getStatsBySubcategory(params),
				getStatsByPeriod({ ...params, group_by: groupBy }),
				getBudgetSummary(month).catch(() => ({
					items: [] as BudgetSummaryItem[],
					month,
					can_copy_from_previous: false
				}))
			]);
			summary = summaryRes;
			byCategory = categoryRes.items;
			bySubcategory = subcategoryRes.items;
			byPeriod = periodRes.items;
			const bmap: Record<string, BudgetSummaryItem> = {};
			for (const b of budgetRes.items) {
				if (b.scope === 'category' && b.category_id) {
					bmap[b.category_id] = b;
				}
			}
			budgetByCategory = bmap;
			if (search.trim()) {
				const searchRes = await searchStats({
					...params,
					q: search.trim(),
					page: '1',
					limit: '20'
				});
				searchRows = searchRes.data;
			} else {
				searchRows = [];
			}
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, {
				background: !initial,
				hasData: summary != null || byCategory.length > 0
			});
			if (msg) loadError = msg;
		} finally {
			loading = false;
			filterLoading = false;
		}
	}

	function resetFilters() {
		fromLocal = '';
		toLocal = '';
		type = '';
		accountId = '';
		categoryId = '';
		groupBy = 'month';
		search = '';
		lastFiltersKey = currentFiltersKey();
		void loadStats(false);
	}

	function buildQuery(parts: Array<[string, string]>): string {
		return parts
			.map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
			.join('&');
	}

	function categoryDrilldownQuery(item: StatsCategoryItem): string {
		const parts: Array<[string, string]> = [['category_id', item.category_id]];
		if (fromLocal) parts.push(['from_local', dateOnlyLocalValue(fromLocal)]);
		if (toLocal) parts.push(['to_local', dateOnlyLocalValue(toLocal)]);
		if (type) parts.push(['type', type]);
		if (accountId) parts.push(['account_id', accountId]);
		return buildQuery(parts);
	}

	function normalizeNameKey(name: string): string {
		return name.trim().toLowerCase();
	}

	function toDebtorMap(items: Debtor[]): Record<string, string> {
		const out: Record<string, string> = {};
		for (const item of items) {
			out[normalizeNameKey(item.name)] = item.id;
		}
		return out;
	}

	function toCreditMap(items: Credit[]): Record<string, string> {
		const out: Record<string, string> = {};
		for (const item of items) {
			const name = item.name?.trim();
			if (!name) continue;
			out[normalizeNameKey(name)] = item.id;
		}
		return out;
	}

	function debtorIDFromDescription(tx: Transaction): string | null {
		if (tx.category_name !== 'Долги' && tx.category_name !== 'Debts') return null;
		const text = tx.description?.trim() ?? '';
		const match =
			/^(?:Дал в долг|Взял в долг|Возврат долга|Погашение долга|Частичный возврат долга|Частичное погашение долга):\s*([^—]+?)(?:\s*—.*)?$/u.exec(
				text
			);
		if (!match) return null;
		return debtorByName[normalizeNameKey(match[1] ?? '')] ?? null;
	}

	function debtorNameFromDescription(tx: Transaction): string | null {
		if (tx.category_name !== 'Долги' && tx.category_name !== 'Debts') return null;
		const text = tx.description?.trim() ?? '';
		const match =
			/^(?:Дал в долг|Взял в долг|Возврат долга|Погашение долга|Частичный возврат долга|Частичное погашение долга):\s*([^—]+?)(?:\s*—.*)?$/u.exec(
				text
			);
		return match?.[1]?.trim() ?? null;
	}

	function creditIDFromDescription(tx: Transaction): string | null {
		if (tx.category_name !== 'Кредиты' && tx.category_name !== 'Credits') return null;
		const text = tx.description?.trim() ?? '';
		if (!text || text === 'Кредит') return null;
		return creditByName[normalizeNameKey(text)] ?? null;
	}

	function creditNameFromDescription(tx: Transaction): string | null {
		if (tx.category_name !== 'Кредиты' && tx.category_name !== 'Credits') return null;
		const text = tx.description?.trim() ?? '';
		if (!text || text === 'Кредит' || text === 'Credits') return null;
		return text;
	}
</script>

<svelte:head>
	<title>{$_('stats.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-4">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/stats', label: $_('stats.title') }
		]}
	/>
	<h1 class="text-2xl font-semibold">{$_('stats.title')}</h1>

	<FilterPanel>
		<div class="filter-panel-body grid items-end gap-3 sm:grid-cols-2 md:mt-0 mt-3 lg:grid-cols-4">
			<DateTimePicker
				label={$_('stats.filters.from')}
				bind:value={fromLocal}
				{...dateOnlyPicker}
				usePortal
			/>
			<DateTimePicker
				label={$_('stats.filters.to')}
				bind:value={toLocal}
				{...dateOnlyPicker}
				usePortal
			/>
			<Select
				id="stats-type"
				label={$_('stats.filters.type')}
				bind:value={type}
				options={typeOptions}
				usePortal
			/>
			<Select
				id="stats-account"
				label={$_('stats.filters.account')}
				bind:value={accountId}
				options={accountOptions}
				usePortal
			/>
			<Select
				id="stats-category"
				label={$_('stats.filters.category')}
				bind:value={categoryId}
				options={categoryOptions}
				usePortal
			/>
			<Select
				id="stats-group-by"
				label={$_('stats.filters.groupBy')}
				bind:value={groupBy}
				options={groupByOptions}
				usePortal
			/>
			<label class="block min-w-0 sm:col-span-2">
				<span class="mb-1.5 block text-sm font-medium">{$_('stats.filters.search')}</span>
				<input
					class="input w-full"
					bind:value={search}
					placeholder={$_('stats.filters.searchPlaceholder')}
				/>
			</label>
			<div class="flex items-end gap-2 sm:col-span-2 lg:col-span-4 lg:justify-end">
				<button type="button" class="btn-ghost min-h-11" onclick={resetFilters}>
					{$_('transactions.filters.reset')}
				</button>
			</div>
		</div>
	</FilterPanel>

	<PageLoadGate {loading} error={loadError} onretry={() => void loadStats(true)} inline>
		<div class="relative space-y-4" class:opacity-60={filterLoading}>
			{#if filterLoading}
				<p class="text-sm" style:color="var(--text-muted)">{$_('common.loading')}</p>
			{/if}
			<div class="grid gap-3 sm:grid-cols-3">
				<div class="card">
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.summary.income')}</p>
					<p class="text-xl font-semibold">
						<MoneyDisplay cents={summary?.income_total ?? 0} {currency} class="" />
					</p>
				</div>
				<div class="card">
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.summary.expense')}</p>
					<p class="text-xl font-semibold">
						<MoneyDisplay cents={summary?.expense_total ?? 0} {currency} class="" />
					</p>
				</div>
				<div class="card">
					<p class="text-xs" style:color="var(--text-muted)">{$_('stats.summary.count')}</p>
					<p class="tabular-nums text-xl font-semibold">{summary?.transaction_count ?? 0}</p>
				</div>
			</div>

			{#if search.trim()}
				<div class="card md:overflow-x-auto">
					<h2 class="mb-2 text-lg font-medium">{$_('stats.section.search')}</h2>
					<TransactionList
						transactions={searchRows}
						siblings={searchRows}
						{tz}
						emptyMessage={$_('transactions.empty')}
						showDescription
						showAmountSign
						descriptionExtra={searchDescriptionLinks}
					/>
				</div>
			{/if}

			<div class="card md:overflow-x-auto">
				<h2 class="mb-2 text-lg font-medium">{$_('stats.section.categories')}</h2>
				{#if byCategory.length === 0}
					<EmptyStateCard message={$_('transactions.empty')} />
				{:else}
					<div class="space-y-6">
						{#if showCategoryIncome}
							<section>
								{#if type === ''}
									<h3 class="mb-2 text-sm font-medium" style:color="var(--text-muted)">
										{$_('stats.summary.income')}
									</h3>
								{/if}
								{#if byCategoryIncome.length === 0}
									<EmptyStateCard message={$_('transactions.empty')} />
								{:else}
									{@render categorySection(byCategoryIncome)}
								{/if}
							</section>
						{/if}
						{#if showCategoryExpense}
							<section>
								{#if type === ''}
									<h3 class="mb-2 text-sm font-medium" style:color="var(--text-muted)">
										{$_('stats.summary.expense')}
									</h3>
								{/if}
								{#if byCategoryExpense.length === 0}
									<EmptyStateCard message={$_('transactions.empty')} />
								{:else}
									{@render categorySection(byCategoryExpense, true)}
								{/if}
							</section>
						{/if}
					</div>
				{/if}
			</div>

			<div class="card md:overflow-x-auto">
				<h2 class="mb-2 text-lg font-medium">{$_('stats.section.period')}</h2>
				{#if byPeriod.length === 0}
					<EmptyStateCard message={$_('transactions.empty')} />
				{:else}
					<div class="mb-3 space-y-2 md:hidden">
						{#each byPeriod as row (row.period)}
							<div class="space-y-1">
								<div class="text-xs" style:color="var(--text-muted)">
									{periodLabel(row.period)}
								</div>
								<div class="flex gap-1">
									<div
										class="h-2 rounded bg-emerald-500"
										style:width={`${Math.max(2, (Math.abs(row.income) / periodMax) * 100)}%`}
										title={`${$_('stats.summary.income')}: ${formatMoneyForDisplay({ cents: row.income })}`}
									></div>
									<div
										class="h-2 rounded bg-rose-500"
										style:width={`${Math.max(2, (Math.abs(row.expense) / periodMax) * 100)}%`}
										title={`${$_('stats.summary.expense')}: ${formatMoneyForDisplay({ cents: row.expense })}`}
									></div>
								</div>
							</div>
						{/each}
					</div>
					<table class="hidden w-full text-left text-sm md:table">
						<thead>
							<tr style:color="var(--text-muted)">
								<th class="p-2">{$_('stats.period')}</th>
								<th class="p-2">{$_('stats.summary.income')}</th>
								<th class="p-2">{$_('stats.summary.expense')}</th>
							</tr>
						</thead>
						<tbody>
							{#each byPeriod as row (row.period)}
								<tr class="border-t" style:border-color="var(--border)">
									<td class="p-2">{periodLabel(row.period)}</td>
									<td class="p-2"><MoneyDisplay cents={row.income} class="" /></td>
									<td class="p-2"><MoneyDisplay cents={row.expense} class="" /></td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			</div>
		</div>
	</PageLoadGate>
</div>

{#snippet categorySection(rows: StatsCategoryItem[], showBudget = false)}
	<div class="space-y-3 md:hidden">
		{#each rows as row (row.category_id)}
			{@const subs = subByCategory[row.category_id] ?? []}
			<article class="rounded-xl border p-3" style:border-color="var(--border)">
				<a
					href={resolve(`/transactions?${categoryDrilldownQuery(row)}`)}
					class="font-medium hover:underline"
					style:color="var(--primary)"
				>
					{statsCategoryLabel(row.category_name, row.type)}
				</a>
				<dl class="mt-2 grid gap-1 text-sm">
					<div class="flex justify-between gap-2">
						<dt style:color="var(--text-muted)">{$_('transactions.col.amount')}</dt>
						<dd><MoneyDisplay cents={row.total} class="" /></dd>
					</div>
					{#if showBudget}
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('budget.stats.planned')}</dt>
							<dd>
								{#if budgetByCategory[row.category_id]}
									<MoneyDisplay
										value={budgetByCategory[row.category_id].planned_display}
										class=""
									/>
								{:else}
									—
								{/if}
							</dd>
						</div>
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('budget.stats.remaining')}</dt>
							<dd class="tabular-nums">
								{budgetByCategory[row.category_id]
									? budgetRemainingCell(budgetByCategory[row.category_id])
									: '—'}
							</dd>
						</div>
					{/if}
					<div class="flex justify-between gap-2">
						<dt style:color="var(--text-muted)">%</dt>
						<dd>{row.percentage.toFixed(1)}</dd>
					</div>
					<div class="flex justify-between gap-2">
						<dt style:color="var(--text-muted)">{$_('stats.summary.count')}</dt>
						<dd>{row.count}</dd>
					</div>
				</dl>
				{#if subs.length > 0}
					<details class="mt-2 border-t pt-2" style:border-color="var(--border)">
						<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
							{$_('stats.subcategories')} ({subs.length})
						</summary>
						<ul class="mt-2 space-y-1.5 text-sm">
							{#each subs as sub (sub.subcategory_id)}
								<li class="flex items-baseline justify-between gap-2">
									<span>{sub.subcategory_name}</span>
									<span class="shrink-0 tabular-nums" style:color="var(--text-muted)">
										<MoneyDisplay cents={sub.total} class="" />
										· {subcategoryShareOfParent(sub.total, row.total).toFixed(0)}% · {sub.count}
									</span>
								</li>
							{/each}
						</ul>
					</details>
				{/if}
			</article>
		{/each}
	</div>
	<table class="hidden w-full text-left text-sm md:table">
		<thead>
			<tr style:color="var(--text-muted)">
				<th class="p-2">{$_('transactions.col.category')}</th>
				<th class="p-2">{$_('transactions.col.amount')}</th>
				{#if showBudget}
					<th class="p-2">{$_('budget.stats.planned')}</th>
					<th class="p-2">{$_('budget.stats.remaining')}</th>
				{/if}
				<th class="p-2">%</th>
				<th class="p-2">{$_('stats.summary.count')}</th>
			</tr>
		</thead>
		<tbody>
			{#each rows as row (row.category_id)}
				{@const subs = subByCategory[row.category_id] ?? []}
				<tr class="border-t" style:border-color="var(--border)">
					<td class="p-2 align-top">
						<a
							href={resolve(`/transactions?${categoryDrilldownQuery(row)}`)}
							class="hover:underline"
							style:color="var(--primary)"
						>
							{statsCategoryLabel(row.category_name, row.type)}
						</a>
						{#if subs.length > 0}
							<details class="mt-1">
								<summary class="cursor-pointer text-xs" style:color="var(--text-muted)">
									{$_('stats.subcategories')} ({subs.length})
								</summary>
								<ul class="mt-1 space-y-1 text-xs">
									{#each subs as sub (sub.subcategory_id)}
										<li class="flex items-baseline justify-between gap-3 pl-1">
											<span>{sub.subcategory_name}</span>
											<span class="shrink-0 tabular-nums" style:color="var(--text-muted)">
												<MoneyDisplay cents={sub.total} class="" />
												· {subcategoryShareOfParent(sub.total, row.total).toFixed(0)}% · {sub.count}
											</span>
										</li>
									{/each}
								</ul>
							</details>
						{/if}
					</td>
					<td class="p-2 align-top"><MoneyDisplay cents={row.total} class="" /></td>
					{#if showBudget}
						<td class="p-2 align-top">
							{#if budgetByCategory[row.category_id]}
								<MoneyDisplay value={budgetByCategory[row.category_id].planned_display} class="" />
							{:else}
								—
							{/if}
						</td>
						<td class="p-2 align-top tabular-nums">
							{budgetByCategory[row.category_id]
								? budgetRemainingCell(budgetByCategory[row.category_id])
								: '—'}
						</td>
					{/if}
					<td class="p-2 align-top">{row.percentage.toFixed(1)}</td>
					<td class="p-2 align-top">{row.count}</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/snippet}

{#snippet searchDescriptionLinks(tx: Transaction)}
	{@const debtorId = debtorIDFromDescription(tx)}
	{@const creditId = creditIDFromDescription(tx)}
	{#if debtorId}
		<EntityLink
			kind="debtor"
			id={debtorId}
			label={debtorNameFromDescription(tx) ?? $_('debtors.title')}
			class="ml-1"
		/>
	{:else if creditId}
		<EntityLink
			kind="credit"
			id={creditId}
			label={creditNameFromDescription(tx) ?? $_('credits.title')}
			class="ml-1"
		/>
	{/if}
{/snippet}
