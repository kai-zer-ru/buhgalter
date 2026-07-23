<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		createRecurringOperation,
		deleteRecurringOperation,
		getTransaction,
		getUIMeta,
		listRecurringOperations,
		listSubcategories,
		updateRecurringOperation,
		type Account,
		type Category,
		type RecurringOperation,
		type Subcategory
	} from '$lib/api/client';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import Select from '$lib/components/Select.svelte';
	import { confirm } from '$lib/confirm';
	import {
		todayDateLocal,
		fromDatetimeLocalValue,
		toDatetimeLocalValue,
		formatAPIOperationDateTimeForDisplay
	} from '$lib/dates';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { formatMoneyForInput, toAPIAmount } from '$lib/money';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import {
		accountSelectOptions,
		accountsFromUIMeta,
		categorySelectOptions,
		subcategorySelectOptions
	} from '$lib/select-options';
	import { user } from '$lib/stores/auth';

	let items = $state<RecurringOperation[]>([]);
	let accounts = $state<Account[]>([]);
	let categories = $state<Category[]>([]);
	let subcategories = $state<Subcategory[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let saving = $state(false);
	let editId = $state<string | null>(null);
	let formOpen = $state(false);

	let type = $state<'income' | 'expense'>('expense');
	let amount = $state('');
	let description = $state('');
	let accountId = $state('');
	let categoryId = $state('');
	let subcategoryId = $state('');
	let period = $state<'week' | 'two_weeks' | 'month' | 'year'>('month');
	let weekday = $state('1');
	let dayOfMonth = $state('1');
	let startDate = $state('');
	let timeLocal = $state('08:00');
	let active = $state(true);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const categoryOptions = $derived(
		categorySelectOptions(categories.filter((item) => item.type === type && !item.is_system))
	);
	const accountOptions = $derived(accountSelectOptions(accounts));
	const subcategoryOptions = $derived(subcategorySelectOptions(subcategories));

	onMount(() => {
		startDate = todayDateLocal(tz).slice(0, 10);
		syncDayOfMonthFromStartDate('month', startDate);
		void loadAll();
	});

	function dayFromDate(value: string): string {
		const day = Number((value || '').split('T')[0]?.split('-')[2] ?? '');
		if (!Number.isFinite(day) || day < 1 || day > 31) return '1';
		return String(day);
	}

	function dateOnly(value: string): string {
		return (value || '').split('T')[0] ?? '';
	}

	function syncDayOfMonthFromStartDate(nextPeriod: typeof period, nextStartDate: string) {
		if (nextPeriod !== 'month') return;
		dayOfMonth = dayFromDate(nextStartDate);
	}

	function firstCategoryByType(nextType: 'income' | 'expense') {
		return categories.find((item) => item.type === nextType && !item.is_system);
	}

	async function loadAll() {
		loading = true;
		try {
			const [ops, meta] = await Promise.all([listRecurringOperations(), getUIMeta()]);
			items = ops;
			accounts = accountsFromUIMeta(
				meta.accounts.filter((acc) => acc.status === 'active'),
				meta.banks
			) as Account[];
			const uniqueByID: Record<string, Category> = {};
			for (const cat of [...meta.expense_categories, ...meta.income_categories]) {
				uniqueByID[cat.id] = cat;
			}
			categories = Object.values(uniqueByID);
			if (!accountId && accounts.length > 0) accountId = accounts[0].id;
			if (!categoryId) categoryId = firstCategoryByType(type)?.id ?? '';
			await loadSubcategories();
			await prefillFromQueryTransaction();
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: items.length > 0 });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	async function prefillFromQueryTransaction() {
		const txID = $page.url.searchParams.get('from_tx');
		if (!txID) return;
		try {
			const tx = await getTransaction(txID);
			if (tx.type === 'transfer') return;
			type = tx.type;
			amount = formatMoneyForInput(tx.amount_display);
			description = tx.description ?? '';
			accountId = tx.account_id;
			categoryId = tx.category_id ?? '';
			subcategoryId = tx.subcategory_id ?? '';
			const local = toDatetimeLocalValue(tx.transaction_date, tz);
			startDate = (local.split('T')[0] ?? todayDateLocal(tz).slice(0, 10)) as string;
			active = true;
			formOpen = true;
			await loadSubcategories();
			await goto(resolve('/settings/recurring-operations'), {
				replaceState: true,
				noScroll: true,
				keepFocus: true
			});
			toast($_('recurring.prefilled'));
		} catch {
			// Ignore optional prefill failures.
		}
	}

	async function loadSubcategories() {
		if (!categoryId) {
			subcategories = [];
			subcategoryId = '';
			return;
		}
		try {
			subcategories = await listSubcategories(categoryId);
			if (subcategoryId && !subcategories.some((item) => item.id === subcategoryId)) {
				subcategoryId = '';
			}
		} catch {
			subcategories = [];
			subcategoryId = '';
		}
	}

	async function onTypeChange(nextType: 'income' | 'expense') {
		type = nextType;
		const first = firstCategoryByType(nextType);
		categoryId = first?.id ?? '';
		await loadSubcategories();
	}

	function resetForm() {
		editId = null;
		type = 'expense';
		amount = '';
		description = '';
		accountId = accounts[0]?.id ?? '';
		const firstCategory = firstCategoryByType('expense');
		categoryId = firstCategory?.id ?? '';
		subcategoryId = '';
		period = 'month';
		weekday = '1';
		dayOfMonth = '1';
		startDate = todayDateLocal(tz).slice(0, 10);
		timeLocal = '08:00';
		active = true;
		void loadSubcategories();
	}

	function beginEdit(item: RecurringOperation) {
		if (editId === item.id) {
			resetForm();
			return;
		}
		formOpen = false;
		editId = item.id;
		type = item.type;
		amount = formatMoneyForInput(item.amount_display);
		description = item.description ?? '';
		accountId = item.account_id;
		categoryId = item.category_id;
		subcategoryId = item.subcategory_id ?? '';
		period = item.period;
		weekday = String(item.weekday ?? 1);
		dayOfMonth = String(item.day_of_month ?? 1);
		startDate = toDatetimeLocalValue(item.start_date, tz).slice(0, 10);
		syncDayOfMonthFromStartDate(item.period, startDate);
		timeLocal = item.time_local || '00:00';
		active = item.active;
		void loadSubcategories();
	}

	async function remove(item: RecurringOperation) {
		const ok = await confirm({
			message: $_('recurring.confirmDelete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteRecurringOperation(item.id);
			toast($_('common.deleted'));
			await loadAll();
			if (editId === item.id) resetForm();
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function submit(e: Event) {
		e.preventDefault();
		saving = true;
		try {
			const payload = {
				type,
				amount: toAPIAmount(amount),
				description: description.trim() || undefined,
				account_id: accountId,
				category_id: categoryId,
				subcategory_id: subcategoryId || undefined,
				period,
				weekday: period === 'week' || period === 'two_weeks' ? Number(weekday) : undefined,
				day_of_month:
					period === 'year'
						? Number(dayOfMonth)
						: period === 'month'
							? Number(dayFromDate(startDate))
							: undefined,
				start_date: fromDatetimeLocalValue(`${dateOnly(startDate)}T00:00`, tz),
				time_local: timeLocal || '08:00',
				active
			};
			if (editId) {
				await updateRecurringOperation(editId, payload);
			} else {
				await createRecurringOperation(payload);
			}
			toast($_('common.saved'));
			await loadAll();
			resetForm();
			formOpen = false;
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}

	function toggleForm() {
		if (formOpen && !editId) {
			formOpen = false;
			return;
		}
		formOpen = true;
		if (editId) resetForm();
	}

	function onPeriodChange(nextPeriod: typeof period) {
		period = nextPeriod;
		syncDayOfMonthFromStartDate(nextPeriod, startDate);
	}

	function periodLabel(itemPeriod: RecurringOperation['period']): string {
		switch (itemPeriod) {
			case 'week':
				return $_('recurring.period.week');
			case 'two_weeks':
				return $_('recurring.period.twoWeeks');
			case 'month':
				return $_('recurring.period.month');
			default:
				return $_('recurring.period.year');
		}
	}

	function rowActions(item: RecurringOperation): RowAction[] {
		return [
			{
				icon: 'edit',
				label: $_('common.edit'),
				onclick: () => beginEdit(item)
			},
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void remove(item)
			}
		];
	}

	$effect(() => {
		syncDayOfMonthFromStartDate(period, startDate);
	});
</script>

{#snippet operationForm(formPrefix: 'create' | 'edit' | 'edit-mobile')}
	<form class="space-y-4" onsubmit={submit}>
		<div class="flex flex-wrap gap-2">
			<button
				type="button"
				class={type === 'expense' ? 'tab tab-active' : 'tab'}
				onclick={() => void onTypeChange('expense')}
			>
				{$_('transactions.type.expense')}
			</button>
			<button
				type="button"
				class={type === 'income' ? 'tab tab-active' : 'tab'}
				onclick={() => void onTypeChange('income')}
			>
				{$_('transactions.type.income')}
			</button>
		</div>

		<div class="grid gap-3 md:grid-cols-2">
			<div>
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="recurring-amount-{formPrefix}"
				>
					{$_('transactions.field.amount')}
				</label>
				<MoneyInput id="recurring-amount-{formPrefix}" bind:value={amount} />
			</div>
			<div>
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="recurring-description-{formPrefix}"
				>
					{$_('transactions.field.description')}
				</label>
				<input
					id="recurring-description-{formPrefix}"
					class="input w-full"
					bind:value={description}
					placeholder={$_('transactions.field.description')}
					maxlength="160"
				/>
			</div>
		</div>

		<div class="grid gap-3 md:grid-cols-3">
			<Select
				label={$_('transactions.field.account')}
				bind:value={accountId}
				options={accountOptions}
				usePortal
			/>
			<Select
				label={$_('transactions.field.category')}
				bind:value={categoryId}
				options={categoryOptions}
				onchange={() => void loadSubcategories()}
				usePortal
			/>
			<Select
				label={$_('transactions.field.subcategory')}
				bind:value={subcategoryId}
				options={[{ value: '', label: '—' }, ...subcategoryOptions]}
				disabled={subcategoryOptions.length === 0}
				usePortal
			/>
		</div>

		<div class="grid gap-3 md:grid-cols-3">
			<div>
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="recurring-period-{formPrefix}">{$_('recurring.period')}</label
				>
				<select
					id="recurring-period-{formPrefix}"
					class="input w-full"
					bind:value={period}
					onchange={(e) =>
						onPeriodChange((e.currentTarget as HTMLSelectElement).value as typeof period)}
				>
					<option value="week">{$_('recurring.period.week')}</option>
					<option value="two_weeks">{$_('recurring.period.twoWeeks')}</option>
					<option value="month">{$_('recurring.period.month')}</option>
					<option value="year">{$_('recurring.period.year')}</option>
				</select>
			</div>
			<div>
				<DateTimePicker
					id="recurring-start-date-{formPrefix}"
					label={$_('recurring.startDate')}
					bind:value={startDate}
					{...dateOnlyPicker}
					usePortal
					required
				/>
			</div>
			{#if period === 'week' || period === 'two_weeks'}
				<div>
					<label
						class="mb-1 block text-sm"
						style:color="var(--text-muted)"
						for="recurring-weekday-{formPrefix}">{$_('recurring.weekday')}</label
					>
					<select id="recurring-weekday-{formPrefix}" class="input w-full" bind:value={weekday}>
						<option value="1">{$_('datetime.weekday.mon')}</option>
						<option value="2">{$_('datetime.weekday.tue')}</option>
						<option value="3">{$_('datetime.weekday.wed')}</option>
						<option value="4">{$_('datetime.weekday.thu')}</option>
						<option value="5">{$_('datetime.weekday.fri')}</option>
						<option value="6">{$_('datetime.weekday.sat')}</option>
						<option value="7">{$_('datetime.weekday.sun')}</option>
					</select>
				</div>
			{:else if period === 'year'}
				<div>
					<label
						class="mb-1 block text-sm"
						style:color="var(--text-muted)"
						for="recurring-day-of-month-{formPrefix}">{$_('recurring.dayOfMonth')}</label
					>
					<input
						id="recurring-day-of-month-{formPrefix}"
						class="input w-full"
						type="number"
						min="1"
						max="31"
						bind:value={dayOfMonth}
					/>
				</div>
			{/if}
		</div>
		<details>
			<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
				{$_('recurring.timeAdvanced')}
			</summary>
			<div class="mt-2">
				<input class="input w-full" type="time" bind:value={timeLocal} step="60" />
			</div>
		</details>
		<label class="inline-flex items-center gap-2 text-sm">
			<input type="checkbox" bind:checked={active} />
			{$_('recurring.active')}
		</label>
		<div class="flex flex-wrap gap-2">
			<button type="submit" class="btn-primary" disabled={saving}>
				{saving
					? $_('common.loading')
					: formPrefix === 'create'
						? $_('common.create')
						: $_('common.save')}
			</button>
			{#if formPrefix !== 'create'}
				<button type="button" class="btn-ghost" onclick={resetForm}>{$_('common.cancel')}</button>
			{/if}
		</div>
	</form>
{/snippet}

<div class="space-y-5">
	<SectionHeader title={$_('nav.recurring')}>
		{#snippet actions()}
			<button type="button" class="btn-primary shrink-0" onclick={toggleForm}>
				{formOpen && !editId ? $_('common.cancel') : $_('recurring.add')}
			</button>
		{/snippet}
	</SectionHeader>

	{#if formOpen && !editId}
		<div class="card">
			{@render operationForm('create')}
		</div>
	{/if}

	<PageLoadGate {loading} error={loadError} onretry={() => void loadAll()} inline>
		{#if items.length === 0 && !formOpen}
			<EmptyStateCard message={$_('recurring.empty')} />
		{:else if items.length > 0}
			<div class="card md:overflow-x-auto">
				<div class="hidden md:block">
					<table class="w-full text-left text-sm">
						<thead>
							<tr style:color="var(--text-muted)">
								<th class="p-3">{$_('transactions.col.description')}</th>
								<th class="p-3">{$_('recurring.period')}</th>
								<th class="p-3">{$_('transactions.field.account')}</th>
								<th class="p-3">{$_('recurring.nextRun')}</th>
								<th class="p-3"></th>
							</tr>
						</thead>
						<tbody>
							{#each items as item (item.id)}
								<tr class="border-t" style:border-color="var(--border)">
									<td class="p-3">
										<div class="font-medium">{item.description || item.category_name}</div>
										<div class="text-xs" style:color="var(--text-muted)">
											{item.category_name}
											{#if item.subcategory_name}
												• {item.subcategory_name}
											{/if}
											• <MoneyDisplay value={item.amount_display} class="" />
										</div>
									</td>
									<td class="p-3">{periodLabel(item.period)}</td>
									<td class="p-3">{item.account_name}</td>
									<td class="p-3">{formatAPIOperationDateTimeForDisplay(item.next_run_at, tz)}</td>
									<td class="p-3 text-right">
										<RowActionsMenu actions={rowActions(item)} />
									</td>
								</tr>
								{#if editId === item.id}
									<tr class="border-t" style:border-color="var(--border)">
										<td colspan="5" class="p-3">
											{@render operationForm('edit')}
										</td>
									</tr>
								{/if}
							{/each}
						</tbody>
					</table>
				</div>

				<div class="space-y-3 p-3 md:hidden">
					{#each items as item (item.id)}
						<article class="rounded-xl border p-4" style:border-color="var(--border)">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0">
									<p class="font-medium">{item.description || item.category_name}</p>
									<p class="mt-1 text-xs" style:color="var(--text-muted)">
										{item.category_name}
										{#if item.subcategory_name}
											• {item.subcategory_name}
										{/if}
									</p>
								</div>
								<p class="shrink-0 text-sm font-semibold tabular-nums">
									<MoneyDisplay value={item.amount_display} class="" />
								</p>
							</div>
							<dl class="mt-3 grid gap-2 text-sm">
								<div class="flex justify-between gap-2">
									<dt style:color="var(--text-muted)">{$_('recurring.period')}</dt>
									<dd>{periodLabel(item.period)}</dd>
								</div>
								<div class="flex justify-between gap-2">
									<dt style:color="var(--text-muted)">{$_('transactions.field.account')}</dt>
									<dd class="text-right">{item.account_name}</dd>
								</div>
								<div class="flex justify-between gap-2">
									<dt style:color="var(--text-muted)">{$_('recurring.nextRun')}</dt>
									<dd class="text-right">
										{formatAPIOperationDateTimeForDisplay(item.next_run_at, tz)}
									</dd>
								</div>
							</dl>
							<div class="mt-3 flex justify-end">
								<RowActionsMenu actions={rowActions(item)} />
							</div>
						</article>
						{#if editId === item.id}
							<div class="rounded-xl border p-4" style:border-color="var(--border)">
								{@render operationForm('edit-mobile')}
							</div>
						{/if}
					{/each}
				</div>
			</div>
		{/if}
	</PageLoadGate>
</div>
