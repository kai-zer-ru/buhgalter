<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		createRecurringOperation,
		deleteRecurringOperation,
		getTransaction,
		listAccounts,
		listCategories,
		listRecurringOperations,
		listSubcategories,
		updateRecurringOperation,
		type Account,
		type Category,
		type RecurringOperation,
		type Subcategory
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import { confirm } from '$lib/confirm';
	import {
		todayDateLocal,
		fromDatetimeLocalValue,
		toDatetimeLocalValue,
		formatAPIDateTimeForDisplay
	} from '$lib/dates';
	import { formatMoneyDisplay } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let items = $state<RecurringOperation[]>([]);
	let accounts = $state<Account[]>([]);
	let categories = $state<Category[]>([]);
	let subcategories = $state<Subcategory[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let editId = $state<string | null>(null);
	let formOpen = $state(false);

	let type = $state<'income' | 'expense'>('expense');
	let amount = $state('0');
	let description = $state('');
	let accountId = $state('');
	let categoryId = $state('');
	let subcategoryId = $state('');
	let period = $state<'week' | 'two_weeks' | 'month' | 'year'>('month');
	let weekday = $state('1');
	let dayOfMonth = $state('1');
	let startDate = $state('');
	let timeLocal = $state('00:00');
	let active = $state(true);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const categoryOptions = $derived(
		categories
			.filter((item) => item.type === type && !item.is_system)
			.map((item) => ({ value: item.id, label: item.name }))
	);
	const accountOptions = $derived(accounts.map((item) => ({ value: item.id, label: item.name })));
	const subcategoryOptions = $derived(
		subcategories.map((item) => ({ value: item.id, label: item.name }))
	);

	onMount(() => {
		startDate = todayDateLocal(tz).slice(0, 10);
		syncDayOfMonthFromStartDate('month', startDate);
		void loadAll();
	});

	function dayFromDate(value: string): string {
		const day = Number((value || '').split('-')[2] ?? '');
		if (!Number.isFinite(day) || day < 1 || day > 31) return '1';
		return String(day);
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
		error = '';
		try {
			const [ops, accs, expenseCats, incomeCats] = await Promise.all([
				listRecurringOperations(),
				listAccounts('active'),
				listCategories('expense'),
				listCategories('income')
			]);
			items = ops;
			accounts = accs;
			const uniqueByID: Record<string, Category> = {};
			for (const cat of [...expenseCats, ...incomeCats]) uniqueByID[cat.id] = cat;
			categories = Object.values(uniqueByID);
			if (!accountId && accounts.length > 0) accountId = accounts[0].id;
			if (!categoryId) categoryId = firstCategoryByType(type)?.id ?? '';
			await loadSubcategories();
			await prefillFromQueryTransaction();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
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
			amount = formatMoneyDisplay(tx.amount_display);
			description = tx.description ?? '';
			accountId = tx.account_id;
			categoryId = tx.category_id ?? '';
			subcategoryId = tx.subcategory_id ?? '';
			const local = toDatetimeLocalValue(tx.transaction_date, tz);
			startDate = (local.split('T')[0] ?? todayDateLocal(tz).slice(0, 10)) as string;
			active = true;
			formOpen = true;
			await loadSubcategories();
			await goto(resolve('/recurring-operations'), {
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
		amount = '0';
		description = '';
		accountId = accounts[0]?.id ?? '';
		const firstCategory = firstCategoryByType('expense');
		categoryId = firstCategory?.id ?? '';
		subcategoryId = '';
		period = 'month';
		weekday = '1';
		dayOfMonth = '1';
		startDate = todayDateLocal(tz).slice(0, 10);
		timeLocal = '00:00';
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
		amount = formatMoneyDisplay(item.amount_display);
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
			error = err instanceof ApiError ? err.message : $_('common.error');
		}
	}

	async function submit(e: Event) {
		e.preventDefault();
		saving = true;
		error = '';
		try {
			const payload = {
				type,
				amount,
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
				start_date: fromDatetimeLocalValue(`${startDate}T00:00`, tz),
				time_local: timeLocal || '00:00',
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
			error = err instanceof ApiError ? err.message : $_('common.error');
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

	function onStartDateChange(nextStartDate: string) {
		startDate = nextStartDate;
		syncDayOfMonthFromStartDate(period, nextStartDate);
	}
</script>

<svelte:head>
	<title>{$_('recurring.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-5">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/recurring-operations', label: $_('recurring.title') }
		]}
	/>

	<h1 class="text-2xl font-semibold">{$_('recurring.title')}</h1>

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if items.length === 0}
		<p style:color="var(--text-muted)">{$_('recurring.empty')}</p>
	{:else}
		<div class="card md:overflow-x-auto">
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
									• {formatMoneyDisplay(item.amount_display)}
								</div>
							</td>
							<td class="p-3">
								{#if item.period === 'week'}
									{$_('recurring.period.week')}
								{:else if item.period === 'two_weeks'}
									{$_('recurring.period.twoWeeks')}
								{:else if item.period === 'month'}
									{$_('recurring.period.month')}
								{:else}
									{$_('recurring.period.year')}
								{/if}
							</td>
							<td class="p-3">{item.account_name}</td>
							<td class="p-3">{formatAPIDateTimeForDisplay(item.next_run_at, tz)}</td>
							<td class="p-3 text-right">
								<div class="flex justify-end gap-1">
									<IconButton
										icon="edit"
										label={$_('common.edit')}
										onclick={() => beginEdit(item)}
									/>
									<IconButton
										icon="delete"
										label={$_('common.delete')}
										variant="danger"
										onclick={() => void remove(item)}
									/>
								</div>
							</td>
						</tr>
						{#if editId === item.id}
							<tr class="border-t" style:border-color="var(--border)">
								<td colspan="5" class="p-3">
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
													for="recurring-amount-edit"
												>
													{$_('transactions.field.amount')}
												</label>
												<MoneyInput id="recurring-amount-edit" bind:value={amount} />
											</div>
											<div>
												<label
													class="mb-1 block text-sm"
													style:color="var(--text-muted)"
													for="recurring-description-edit"
												>
													{$_('transactions.field.description')}
												</label>
												<input
													id="recurring-description-edit"
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
													for="recurring-period-edit">{$_('recurring.period')}</label
												>
												<select
													id="recurring-period-edit"
													class="input w-full"
													bind:value={period}
													onchange={(e) =>
														onPeriodChange(
															(e.currentTarget as HTMLSelectElement).value as typeof period
														)}
												>
													<option value="week">{$_('recurring.period.week')}</option>
													<option value="two_weeks">{$_('recurring.period.twoWeeks')}</option>
													<option value="month">{$_('recurring.period.month')}</option>
													<option value="year">{$_('recurring.period.year')}</option>
												</select>
											</div>
											<div>
												<label
													class="mb-1 block text-sm"
													style:color="var(--text-muted)"
													for="recurring-start-date-edit">{$_('recurring.startDate')}</label
												>
												<input
													id="recurring-start-date-edit"
													class="input w-full"
													type="date"
													bind:value={startDate}
													onchange={(e) =>
														onStartDateChange((e.currentTarget as HTMLInputElement).value)}
													required
												/>
											</div>
											{#if period === 'week' || period === 'two_weeks'}
												<div>
													<label
														class="mb-1 block text-sm"
														style:color="var(--text-muted)"
														for="recurring-weekday-edit">{$_('recurring.weekday')}</label
													>
													<select
														id="recurring-weekday-edit"
														class="input w-full"
														bind:value={weekday}
													>
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
														for="recurring-day-of-month-edit">{$_('recurring.dayOfMonth')}</label
													>
													<input
														id="recurring-day-of-month-edit"
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
												{saving ? $_('common.loading') : $_('common.save')}
											</button>
											<button type="button" class="btn-ghost" onclick={resetForm}>
												{$_('common.cancel')}
											</button>
										</div>
										{#if error}
											<p class="text-sm" style:color="var(--danger)">{error}</p>
										{/if}
									</form>
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<div class="pt-1">
		<button type="button" class="btn-primary" onclick={toggleForm}>
			{formOpen ? $_('recurring.hideForm') : $_('recurring.add')}
		</button>
	</div>

	{#if formOpen && !editId}
		<form class="card space-y-4" onsubmit={submit}>
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
					<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="recurring-amount">
						{$_('transactions.field.amount')}
					</label>
					<MoneyInput id="recurring-amount" bind:value={amount} />
				</div>
				<div>
					<label
						class="mb-1 block text-sm"
						style:color="var(--text-muted)"
						for="recurring-description"
					>
						{$_('transactions.field.description')}
					</label>
					<input
						id="recurring-description"
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
					<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="recurring-period"
						>{$_('recurring.period')}</label
					>
					<select
						id="recurring-period"
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
					<label
						class="mb-1 block text-sm"
						style:color="var(--text-muted)"
						for="recurring-start-date">{$_('recurring.startDate')}</label
					>
					<input
						id="recurring-start-date"
						class="input w-full"
						type="date"
						bind:value={startDate}
						onchange={(e) => onStartDateChange((e.currentTarget as HTMLInputElement).value)}
						required
					/>
				</div>
				{#if period === 'week' || period === 'two_weeks'}
					<div>
						<label
							class="mb-1 block text-sm"
							style:color="var(--text-muted)"
							for="recurring-weekday">{$_('recurring.weekday')}</label
						>
						<select id="recurring-weekday" class="input w-full" bind:value={weekday}>
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
							for="recurring-day-of-month">{$_('recurring.dayOfMonth')}</label
						>
						<input
							id="recurring-day-of-month"
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
					{saving ? $_('common.loading') : editId ? $_('common.save') : $_('common.create')}
				</button>
				{#if editId}
					<button type="button" class="btn-ghost" onclick={resetForm}>{$_('common.cancel')}</button>
				{/if}
			</div>
			{#if error}
				<p class="text-sm" style:color="var(--danger)">{error}</p>
			{/if}
		</form>
	{/if}
</div>
