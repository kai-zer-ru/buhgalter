<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		copyBudgetsFromPreviousMonth,
		copyBudgetToNextMonth,
		createBudget,
		deleteBudget,
		getBudgetSummary,
		getUIMeta,
		listSubcategories,
		updateBudget,
		type BudgetScope,
		type BudgetSummaryItem,
		type Category,
		type Subcategory,
		type UIMetaAccountRef,
		type Bank
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import IntegerInput from '$lib/components/IntegerInput.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import Select from '$lib/components/Select.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { confirm } from '$lib/confirm';
	import { formatMoneyDisplay, formatMoneyForInput, toAPIAmount } from '$lib/money';
	import { toast } from '$lib/toast';
	import { tr } from '$lib/i18n';
	import { budgetStatusLine } from '$lib/budget-display';
	import {
		accountRefSelectOption,
		categorySelectOptions,
		subcategorySelectOptions
	} from '$lib/select-options';

	let items = $state<BudgetSummaryItem[]>([]);
	let canCopyFromPrevious = $state(false);
	let categories = $state<Category[]>([]);
	let accounts = $state<UIMetaAccountRef[]>([]);
	let banks = $state<Bank[]>([]);
	let subcategories = $state<Subcategory[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let formOpen = $state(false);
	let editId = $state<string | null>(null);

	let name = $state('');
	let scope = $state<BudgetScope>('category');
	let categoryId = $state('');
	let subcategoryId = $state('');
	let accountId = $state('');
	let amount = $state('');
	let alertAtPercent = $state('90');
	let isActive = $state(true);
	let copyForward = $state(false);
	let copying = $state(false);

	const month = $derived($page.url.searchParams.get('month') ?? currentMonthKey());

	const monthLabel = $derived.by(() => {
		void $_;
		const [y, m] = month.split('-').map(Number);
		return new Intl.DateTimeFormat(undefined, { month: 'long', year: 'numeric' }).format(
			new Date(y, m - 1, 1)
		);
	});

	const expenseCategories = $derived(
		categories.filter((c) => c.type === 'expense' && !c.is_system)
	);
	const hasAllExpenseBudget = $derived(
		items.some((i) => i.scope === 'all_expense' && i.id !== editId)
	);
	const usedCategoryIds = $derived(
		new Set(
			items.filter((i) => i.scope === 'category' && i.id !== editId).map((i) => i.category_id)
		)
	);
	const usedSubcategoryIds = $derived(
		new Set(
			items.filter((i) => i.scope === 'subcategory' && i.id !== editId).map((i) => i.subcategory_id)
		)
	);
	const categoryOptions = $derived(
		categorySelectOptions(expenseCategories.filter((c) => !usedCategoryIds.has(c.id)))
	);
	const accountOptions = $derived([
		{ value: '', label: $_('budget.field.account_all') },
		...accounts.map((a) => accountRefSelectOption(a, banks))
	]);
	const scopeOptions = $derived(
		[
			{ value: 'category', label: $_('budget.scope.category') },
			{ value: 'subcategory', label: $_('budget.scope.subcategory') },
			{ value: 'all_expense', label: $_('budget.scope.all_expense') }
		].filter((o) => o.value !== 'all_expense' || !hasAllExpenseBudget)
	);
	const subcategoryOptions = $derived(
		subcategorySelectOptions(subcategories.filter((s) => !usedSubcategoryIds.has(s.id)))
	);

	function currentMonthKey() {
		const now = new Date();
		return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
	}

	function shiftMonth(delta: number) {
		const [y, m] = month.split('-').map(Number);
		const d = new Date(y, m - 1 + delta, 1);
		const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
		void goto(resolve(`/budget?month=${key}`), { replaceState: true, keepFocus: true });
	}

	function progressClass(status: string) {
		if (status === 'exceeded') return 'bg-red-500';
		if (status === 'warning') return 'bg-amber-500';
		return 'bg-emerald-500';
	}

	function resetForm() {
		editId = null;
		name = '';
		scope = 'category';
		categoryId = expenseCategories[0]?.id ?? '';
		subcategoryId = '';
		accountId = '';
		amount = '';
		alertAtPercent = '90';
		isActive = true;
		copyForward = false;
		void loadSubcategories();
	}

	function fillForm(item: BudgetSummaryItem) {
		editId = item.id;
		name = item.name;
		scope = item.scope;
		categoryId = item.category_id ?? '';
		subcategoryId = item.subcategory_id ?? '';
		accountId = item.account_id ?? '';
		amount = formatMoneyForInput(item.planned_display);
		alertAtPercent = String(item.alert_at_percent);
		isActive = item.is_active ?? true;
		copyForward = item.copy_forward ?? false;
		formOpen = false;
		void loadSubcategories();
	}

	async function loadSubcategories() {
		if (!categoryId) {
			subcategories = [];
			subcategoryId = '';
			return;
		}
		try {
			subcategories = await listSubcategories(categoryId);
			if (subcategoryId && !subcategories.some((s) => s.id === subcategoryId)) {
				subcategoryId = '';
			}
		} catch {
			subcategories = [];
			subcategoryId = '';
		}
	}

	async function loadAll() {
		loading = true;
		error = '';
		try {
			const [summary, meta] = await Promise.all([getBudgetSummary(month), getUIMeta()]);
			items = summary.items;
			canCopyFromPrevious = summary.can_copy_from_previous;
			accounts = meta.accounts.filter((a) => a.status === 'active');
			banks = meta.banks;
			categories = meta.expense_categories;
			if (!categoryId && expenseCategories.length > 0) {
				categoryId = expenseCategories[0].id;
			}
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}

	async function submit(e: Event) {
		e.preventDefault();
		saving = true;
		error = '';
		const payload = {
			name: name.trim(),
			scope,
			category_id: scope === 'category' || scope === 'subcategory' ? categoryId : undefined,
			subcategory_id: scope === 'subcategory' ? subcategoryId : undefined,
			account_id: accountId || undefined,
			amount: toAPIAmount(amount),
			alert_at_percent: Number(alertAtPercent) || 0,
			is_active: isActive,
			copy_forward: copyForward
		};
		try {
			if (editId) {
				await updateBudget(editId, payload, month);
			} else {
				await createBudget(payload, month);
			}
			toast($_('common.saved'));
			formOpen = false;
			resetForm();
			await loadAll();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			saving = false;
		}
	}

	async function removeItem(item: BudgetSummaryItem) {
		const ok = await confirm({
			message: $_('budget.confirmDelete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteBudget(item.id);
			toast($_('common.deleted'));
			if (editId === item.id) {
				formOpen = false;
				resetForm();
			}
			await loadAll();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		}
	}

	async function copyFromPreviousMonth() {
		copying = true;
		error = '';
		try {
			await copyBudgetsFromPreviousMonth(month);
			toast($_('common.saved'));
			await loadAll();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			copying = false;
		}
	}

	async function copyItemToNextMonth(item: BudgetSummaryItem) {
		copying = true;
		error = '';
		try {
			await copyBudgetToNextMonth(item.id);
			toast($_('common.saved'));
			await loadAll();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			copying = false;
		}
	}

	function rowActions(item: BudgetSummaryItem): RowAction[] {
		return [
			{
				icon: 'edit',
				label: $_('common.edit'),
				onclick: () => fillForm(item)
			},
			{
				icon: 'repeat',
				label: $_('budget.action.copy_next'),
				onclick: () => void copyItemToNextMonth(item)
			},
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void removeItem(item)
			}
		];
	}

	function toggleForm() {
		if (formOpen && !editId) {
			formOpen = false;
			return;
		}
		formOpen = true;
		if (editId) resetForm();
	}

	onMount(() => {
		void loadAll();
	});

	$effect(() => {
		if (!month) return;
		void loadAll();
	});

	$effect(() => {
		if (scope === 'category' || scope === 'subcategory') {
			void loadSubcategories();
		}
	});

	$effect(() => {
		if (hasAllExpenseBudget && scope === 'all_expense' && !editId) {
			scope = 'category';
		}
		if (scope === 'category' && categoryId && usedCategoryIds.has(categoryId)) {
			categoryId = categoryOptions[0]?.value ?? '';
		}
		if (scope === 'subcategory' && subcategoryId && usedSubcategoryIds.has(subcategoryId)) {
			subcategoryId = subcategoryOptions[0]?.value ?? '';
		}
	});
</script>

{#snippet budgetForm(formPrefix: 'create' | 'edit')}
	<form class="space-y-4" onsubmit={submit}>
		<div class="grid items-end gap-3 md:grid-cols-2">
			<div>
				<label class="field-label" for="budget-name-{formPrefix}">
					{$_('budget.field.name')}
				</label>
				<input id="budget-name-{formPrefix}" class="input w-full" bind:value={name} required />
			</div>
			<div class="hidden md:block">
				<Select
					id="budget-scope-{formPrefix}"
					label={$_('budget.field.scope')}
					bind:value={scope}
					options={scopeOptions}
					usePortal
				/>
			</div>
		</div>
		<div class="md:hidden">
			<Select
				id="budget-scope-{formPrefix}-mobile"
				label={$_('budget.field.scope')}
				bind:value={scope}
				options={scopeOptions}
				usePortal
			/>
		</div>
		{#if scope === 'all_expense'}
			<div>
				<label class="field-label" for="budget-amount-{formPrefix}">
					{$_('budget.field.amount')}
				</label>
				<MoneyInput id="budget-amount-{formPrefix}" bind:value={amount} required />
			</div>
		{:else if scope === 'category'}
			<div class="grid items-end gap-3 md:grid-cols-2">
				<Select
					id="budget-category-{formPrefix}"
					label={$_('budget.field.category')}
					bind:value={categoryId}
					options={categoryOptions}
					onchange={() => void loadSubcategories()}
					usePortal
				/>
				<div>
					<label class="field-label" for="budget-amount-{formPrefix}">
						{$_('budget.field.amount')}
					</label>
					<MoneyInput id="budget-amount-{formPrefix}" bind:value={amount} required />
				</div>
			</div>
		{:else}
			<div class="grid items-end gap-3 md:grid-cols-3">
				<Select
					id="budget-category-{formPrefix}"
					label={$_('budget.field.category')}
					bind:value={categoryId}
					options={categoryOptions}
					onchange={() => void loadSubcategories()}
					usePortal
				/>
				<Select
					id="budget-subcategory-{formPrefix}"
					label={$_('budget.field.subcategory')}
					bind:value={subcategoryId}
					options={subcategoryOptions}
					disabled={subcategoryOptions.length === 0}
					usePortal
				/>
				<div>
					<label class="field-label" for="budget-amount-{formPrefix}">
						{$_('budget.field.amount')}
					</label>
					<MoneyInput id="budget-amount-{formPrefix}" bind:value={amount} required />
				</div>
			</div>
		{/if}
		<div class="grid items-end gap-3 md:grid-cols-3">
			<Select
				id="budget-account-{formPrefix}"
				label={$_('budget.field.account')}
				bind:value={accountId}
				options={accountOptions}
				usePortal
			/>
			<div>
				<label class="field-label" for="budget-alert-{formPrefix}">
					{$_('budget.field.alert')}
				</label>
				<IntegerInput
					id="budget-alert-{formPrefix}"
					class="input w-full tabular-nums"
					value={alertAtPercent === '' ? NaN : Number(alertAtPercent)}
					min={0}
					max={100}
					onchange={(v) => {
						alertAtPercent = Number.isFinite(v) ? String(v) : '';
					}}
				/>
			</div>
			<div>
				<span class="field-label">
					{$_('budget.field.active')}
				</span>
				<div class="flex h-11 items-center">
					<ToggleSwitch
						checked={isActive}
						label={$_('budget.field.active')}
						onchange={() => (isActive = !isActive)}
					/>
				</div>
			</div>
		</div>
		<div class="flex items-center gap-2">
			<ToggleSwitch
				checked={copyForward}
				label={$_('budget.field.copy_forward')}
				onchange={() => (copyForward = !copyForward)}
			/>
			<div>
				<span class="text-sm">{$_('budget.field.copy_forward')}</span>
				<p class="text-xs" style:color="var(--text-muted)">
					{$_('budget.field.copy_forward_hint')}
				</p>
			</div>
		</div>
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
		{#if error}
			<p class="text-sm" style:color="var(--danger)">{error}</p>
		{/if}
	</form>
{/snippet}

<svelte:head>
	<title>{$_('budget.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-5">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/budget', label: $_('budget.title') }
		]}
	/>

	<div class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-semibold">{$_('budget.title')}</h1>

		<div class="flex flex-wrap items-center gap-2">
			{#if canCopyFromPrevious}
				<button
					type="button"
					class="btn-ghost shrink-0"
					disabled={copying}
					onclick={() => void copyFromPreviousMonth()}
				>
					{$_('budget.action.copy_from_previous')}
				</button>
			{/if}
			<div class="flex items-center gap-2">
				<button type="button" class="btn-primary shrink-0" onclick={toggleForm} disabled={copying}>
					{formOpen && !editId ? $_('common.cancel') : $_('budget.add')}
				</button>
				<div class="flex shrink-0 items-center gap-1">
					<button
						type="button"
						class="btn-ghost shrink-0 max-sm:flex max-sm:h-11 max-sm:w-11 max-sm:items-center max-sm:justify-center max-sm:p-0"
						onclick={() => shiftMonth(-1)}
						aria-label={$_('budget.month.prev')}
					>
						←
					</button>
					<span
						class="inline-flex h-11 w-[11rem] shrink-0 items-center justify-center rounded-xl border px-2 text-center text-sm font-medium capitalize sm:h-auto sm:px-3 sm:py-2"
						style:border-color="var(--border)"
					>
						{monthLabel}
					</span>
					<button
						type="button"
						class="btn-ghost shrink-0 max-sm:flex max-sm:h-11 max-sm:w-11 max-sm:items-center max-sm:justify-center max-sm:p-0"
						onclick={() => shiftMonth(1)}
						aria-label={$_('budget.month.next')}
					>
						→
					</button>
				</div>
			</div>
		</div>
	</div>

	{#if formOpen && !editId}
		<div class="card">
			{@render budgetForm('create')}
		</div>
	{/if}

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if items.length === 0 && !formOpen}
		<EmptyStateCard message={$_('budget.empty')} />
	{:else if items.length > 0}
		<div class="grid gap-4 md:grid-cols-2">
			{#each items as item (item.id)}
				<article class="card space-y-3">
					<div class="flex items-start justify-between gap-2">
						<div class="min-w-0">
							<h2 class="font-medium">{item.name}</h2>
							<p class="text-sm" style:color="var(--text-muted)">
								{tr('budget.progress', {
									values: {
										spent: formatMoneyDisplay(item.spent_display),
										planned: formatMoneyDisplay(item.planned_display)
									}
								})}
							</p>
						</div>
						<RowActionsMenu actions={rowActions(item)} />
					</div>
					<div
						class="h-2 overflow-hidden rounded-full"
						style:background-color="color-mix(in srgb, var(--border) 80%, transparent)"
					>
						<div
							class="h-full transition-all {progressClass(item.status)}"
							style="width: {Math.min(item.percent, 100)}%"
						></div>
					</div>
					<p class="text-sm tabular-nums" style:color="var(--text-muted)">
						{budgetStatusLine(item)}
					</p>
					<p class="text-sm" style:color="var(--text-muted)">
						{tr('budget.copy_status', {
							values: {
								value: item.copy_forward
									? $_('budget.copy_status.yes')
									: $_('budget.copy_status.no')
							}
						})}
					</p>
					{#if item.scope === 'all_expense' && item.children_planned_display}
						<p class="text-sm tabular-nums" style:color="var(--text-muted)">
							{tr('budget.children', {
								values: {
									spent: formatMoneyDisplay(item.children_spent_display ?? '0.00'),
									planned: formatMoneyDisplay(item.children_planned_display)
								}
							})}
						</p>
					{/if}
					{#if editId === item.id}
						<div class="border-t pt-3" style:border-color="var(--border)">
							{@render budgetForm('edit')}
						</div>
					{/if}
				</article>
			{/each}
		</div>
	{/if}
</div>
