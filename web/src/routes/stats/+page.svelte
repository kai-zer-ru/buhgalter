<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import {
		ApiError,
		getStatsByCategory,
		getStatsByPeriod,
		getStatsBySubcategory,
		getStatsSummary,
		listAccounts,
		listCategories,
		listCredits,
		listDebtors,
		searchStats,
		type Account,
		type Category,
		type Credit,
		type Debtor,
		type StatsCategoryItem,
		type StatsPeriodItem,
		type StatsSubcategoryItem,
		type StatsSummary,
		type Transaction
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import EntityLink from '$lib/components/EntityLink.svelte';
	import Select from '$lib/components/Select.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import { formatBalance } from '$lib/finance';
	import { fromDatetimeLocalValue } from '$lib/dates';
	import { formatMoneyDisplay, fromCents } from '$lib/money';
	import { user } from '$lib/stores/auth';

	let loading = $state(true);
	let error = $state('');
	let summary = $state<StatsSummary | null>(null);
	let byCategory = $state<StatsCategoryItem[]>([]);
	let bySubcategory = $state<StatsSubcategoryItem[]>([]);
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
			...accounts.map((acc) => ({ value: acc.id, label: acc.name }))
		];
	});
	const categoryOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: '', label: tr('import.export.all_categories') },
			...categories.map((cat) => ({ value: cat.id, label: cat.name }))
		];
	});
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

	onMount(() => void load());

	function statsParams() {
		const params: Record<string, string> = {};
		if (fromLocal) params.from = fromDatetimeLocalValue(fromLocal, tz);
		if (toLocal) params.to = fromDatetimeLocalValue(toLocal, tz);
		if (type) params.type = type;
		if (accountId) params.account_id = accountId;
		if (categoryId) params.category_id = categoryId;
		return params;
	}

	async function load() {
		loading = true;
		error = '';
		try {
			const params = statsParams();
			const [
				summaryRes,
				categoryRes,
				subcategoryRes,
				periodRes,
				accountList,
				expenseCategories,
				incomeCategories,
				debtors,
				activeCredits,
				closedCredits
			] = await Promise.all([
				getStatsSummary(params),
				getStatsByCategory(params),
				getStatsBySubcategory(params),
				getStatsByPeriod({ ...params, group_by: groupBy }),
				listAccounts('active'),
				listCategories('expense'),
				listCategories('income'),
				listDebtors(),
				listCredits({ status: 'active' }),
				listCredits({ status: 'closed' })
			]);
			summary = summaryRes;
			byCategory = categoryRes.items;
			bySubcategory = subcategoryRes.items;
			byPeriod = periodRes.items;
			accounts = accountList;
			const mergedCategories = [...expenseCategories, ...incomeCategories];
			const uniqueByID: Record<string, Category> = {};
			for (const cat of mergedCategories) uniqueByID[cat.id] = cat;
			categories = Object.values(uniqueByID).sort((a, b) => a.name.localeCompare(b.name, 'ru'));
			debtorByName = toDebtorMap(debtors);
			creditByName = toCreditMap([...activeCredits, ...closedCredits]);
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
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}

	function buildQuery(parts: Array<[string, string]>): string {
		return parts
			.map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
			.join('&');
	}

	function categoryDrilldownQuery(item: StatsCategoryItem): string {
		const parts: Array<[string, string]> = [['category_id', item.category_id]];
		if (fromLocal) parts.push(['from', fromDatetimeLocalValue(fromLocal, tz)]);
		if (toLocal) parts.push(['to', fromDatetimeLocalValue(toLocal, tz)]);
		if (type) parts.push(['type', type]);
		if (accountId) parts.push(['account_id', accountId]);
		return buildQuery(parts);
	}

	function subcategoryDrilldownQuery(item: StatsSubcategoryItem): string {
		const parts: Array<[string, string]> = [
			['category_id', item.category_id],
			['search', item.subcategory_name]
		];
		if (fromLocal) parts.push(['from', fromDatetimeLocalValue(fromLocal, tz)]);
		if (toLocal) parts.push(['to', fromDatetimeLocalValue(toLocal, tz)]);
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
	<BackLink href="/" label={$_('dashboard.title')} />
	<h1 class="text-2xl font-semibold">{$_('stats.title')}</h1>

	<details class="filter-panel card" open>
		<summary class="md:hidden">{$_('transactions.filters.toggle')}</summary>
		<div class="grid items-end gap-3 sm:grid-cols-2 md:mt-0 mt-3 lg:grid-cols-4">
			<DateTimePicker label={$_('stats.filters.from')} bind:value={fromLocal} usePortal />
			<DateTimePicker label={$_('stats.filters.to')} bind:value={toLocal} usePortal />
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
					class="input h-11 w-full"
					bind:value={search}
					placeholder={$_('stats.filters.searchPlaceholder')}
				/>
			</label>
			<div class="flex flex-wrap items-center gap-2 sm:col-span-2 lg:col-span-4 lg:justify-end">
				<button type="button" class="btn-primary min-h-11" onclick={() => void load()}>
					{$_('stats.filters.apply')}
				</button>
				<button type="button" class="btn-ghost min-h-11" onclick={() => void load()}>
					{$_('stats.filters.refresh')}
				</button>
			</div>
		</div>
	</details>

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if error}
		<p style:color="var(--danger)">{error}</p>
	{:else}
		<div class="grid gap-3 sm:grid-cols-3">
			<div class="card">
				<p class="text-xs" style:color="var(--text-muted)">{$_('stats.summary.income')}</p>
				<p class="tabular-nums text-xl font-semibold">
					{formatBalance(fromCents(summary?.income_total ?? 0), currency)}
				</p>
			</div>
			<div class="card">
				<p class="text-xs" style:color="var(--text-muted)">{$_('stats.summary.expense')}</p>
				<p class="tabular-nums text-xl font-semibold">
					{formatBalance(fromCents(summary?.expense_total ?? 0), currency)}
				</p>
			</div>
			<div class="card">
				<p class="text-xs" style:color="var(--text-muted)">{$_('stats.summary.delta')}</p>
				<p class="tabular-nums text-xl font-semibold">
					{formatBalance(fromCents(summary?.balance_delta ?? 0), currency)}
				</p>
				<p class="text-xs" style:color="var(--text-muted)">
					{$_('stats.summary.count')}: {summary?.transaction_count ?? 0}
				</p>
			</div>
		</div>

		<div class="grid gap-4 lg:grid-cols-2">
			<div class="card overflow-x-auto">
				<h2 class="mb-2 text-lg font-medium">{$_('stats.section.period')}</h2>
				{#if byPeriod.length === 0}
					<EmptyStateCard message={$_('transactions.empty')} />
				{:else}
					<div class="mb-3 space-y-2 md:hidden">
						{#each byPeriod as row (row.period)}
							<div class="space-y-1">
								<div class="text-xs" style:color="var(--text-muted)">{row.period}</div>
								<div class="flex gap-1">
									<div
										class="h-2 rounded bg-emerald-500"
										style:width={`${Math.max(2, (Math.abs(row.income) / periodMax) * 100)}%`}
										title={`${$_('stats.summary.income')}: ${formatMoneyDisplay(fromCents(row.income))}`}
									></div>
									<div
										class="h-2 rounded bg-rose-500"
										style:width={`${Math.max(2, (Math.abs(row.expense) / periodMax) * 100)}%`}
										title={`${$_('stats.summary.expense')}: ${formatMoneyDisplay(fromCents(row.expense))}`}
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
									<td class="p-2">{row.period}</td>
									<td class="p-2 tabular-nums">{formatMoneyDisplay(fromCents(row.income))}</td>
									<td class="p-2 tabular-nums">{formatMoneyDisplay(fromCents(row.expense))}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			</div>

			<div class="card overflow-x-auto">
				<h2 class="mb-2 text-lg font-medium">{$_('stats.section.categories')}</h2>
				{#if byCategory.length === 0}
					<EmptyStateCard message={$_('transactions.empty')} />
				{:else}
					<div class="space-y-3 md:hidden">
						{#each byCategory as row (row.category_id)}
							<article class="rounded-xl border p-3" style:border-color="var(--border)">
								<a
									href={resolve(`/transactions?${categoryDrilldownQuery(row)}`)}
									class="font-medium hover:underline"
									style:color="var(--primary)"
								>
									{row.category_name}
								</a>
								<dl class="mt-2 grid gap-1 text-sm">
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">{$_('transactions.col.amount')}</dt>
										<dd class="tabular-nums">{formatMoneyDisplay(fromCents(row.total))}</dd>
									</div>
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">%</dt>
										<dd>{row.percentage.toFixed(1)}</dd>
									</div>
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">{$_('stats.summary.count')}</dt>
										<dd>{row.count}</dd>
									</div>
								</dl>
							</article>
						{/each}
					</div>
					<table class="hidden w-full text-left text-sm md:table">
						<thead>
							<tr style:color="var(--text-muted)">
								<th class="p-2">{$_('transactions.col.category')}</th>
								<th class="p-2">{$_('transactions.col.amount')}</th>
								<th class="p-2">%</th>
								<th class="p-2">{$_('stats.summary.count')}</th>
							</tr>
						</thead>
						<tbody>
							{#each byCategory as row (row.category_id)}
								<tr class="border-t" style:border-color="var(--border)">
									<td class="p-2">
										<a
											href={resolve(`/transactions?${categoryDrilldownQuery(row)}`)}
											class="hover:underline"
											style:color="var(--primary)"
										>
											{row.category_name}
										</a>
									</td>
									<td class="p-2 tabular-nums">{formatMoneyDisplay(fromCents(row.total))}</td>
									<td class="p-2">{row.percentage.toFixed(1)}</td>
									<td class="p-2">{row.count}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
				{#if bySubcategory.length > 0}
					<h3 class="mb-2 mt-4 text-sm font-semibold" style:color="var(--text-muted)">
						{$_('transactions.field.subcategory')}
					</h3>
					<div class="space-y-3 md:hidden">
						{#each bySubcategory as row (`${row.category_id}:${row.subcategory_id}`)}
							<article class="rounded-xl border p-3" style:border-color="var(--border)">
								<a
									href={resolve(`/transactions?${subcategoryDrilldownQuery(row)}`)}
									class="font-medium hover:underline"
									style:color="var(--primary)"
								>
									{row.subcategory_name}
								</a>
								<dl class="mt-2 grid gap-1 text-sm">
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">{$_('transactions.col.amount')}</dt>
										<dd class="tabular-nums">{formatMoneyDisplay(fromCents(row.total))}</dd>
									</div>
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">%</dt>
										<dd>{row.percentage.toFixed(1)}</dd>
									</div>
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">{$_('stats.summary.count')}</dt>
										<dd>{row.count}</dd>
									</div>
								</dl>
							</article>
						{/each}
					</div>
					<table class="hidden w-full text-left text-sm md:table">
						<thead>
							<tr style:color="var(--text-muted)">
								<th class="p-2">{$_('transactions.field.subcategory')}</th>
								<th class="p-2">{$_('transactions.col.amount')}</th>
								<th class="p-2">%</th>
								<th class="p-2">{$_('stats.summary.count')}</th>
							</tr>
						</thead>
						<tbody>
							{#each bySubcategory as row (`${row.category_id}:${row.subcategory_id}`)}
								<tr class="border-t" style:border-color="var(--border)">
									<td class="p-2">
										<a
											href={resolve(`/transactions?${subcategoryDrilldownQuery(row)}`)}
											class="hover:underline"
											style:color="var(--primary)"
										>
											{row.subcategory_name}
										</a>
									</td>
									<td class="p-2 tabular-nums">{formatMoneyDisplay(fromCents(row.total))}</td>
									<td class="p-2">{row.percentage.toFixed(1)}</td>
									<td class="p-2">{row.count}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
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
					descriptionExtra={searchDescriptionLinks}
				/>
			</div>
		{/if}
	{/if}
</div>

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
