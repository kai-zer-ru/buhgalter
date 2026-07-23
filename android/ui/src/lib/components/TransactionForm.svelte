<script lang="ts">
	import { _ } from 'svelte-i18n';
	import {
		listAccounts,
		listCategories,
		listSubcategories,
		type Account,
		type Category,
		type Subcategory,
		type Transaction
	} from '$lib/api/client';
	import { createTransaction, updateTransaction } from '$lib/offline/transactions-api';
	import { applyOutboxToAccounts } from '$lib/offline/local-state';
	import { outboxTick } from '$lib/offline/store';
	import { fromDatetimeLocalValue, nowDatetimeLocal, toDatetimeLocalValue } from '$lib/dates';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import {
		operationDatetimePickerCreate,
		operationDatetimePickerEdit
	} from '$lib/datetime-picker-standards';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { creditCardExpenseWarning, isCreditCard } from '$lib/credit-card';
	import { formatMoneyForInput, toAPIAmount, toCents } from '$lib/money';
	import {
		accountSelectOptions,
		categorySelectOptions,
		subcategorySelectOptions
	} from '$lib/select-options';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	type Props = {
		variant?: 'modal' | 'page';
		open?: boolean;
		backHref?: string;
		accountId?: string;
		defaultType?: 'expense' | 'income';
		transaction?: Transaction | null;
		repeatFrom?: Transaction | null;
		initialDescription?: string;
		onclose: () => void;
		onsaved: () => void;
	};

	let {
		variant = 'modal',
		open = $bindable(false),
		backHref = '/',
		accountId = '',
		defaultType = 'expense',
		transaction = null,
		repeatFrom = null,
		initialDescription = '',
		onclose,
		onsaved
	}: Props = $props();

	let txType = $state<'expense' | 'income'>('expense');
	let amount = $state('');
	let selectedAccount = $state('');
	let categoryId = $state('');
	let subcategoryId = $state('');
	let newSubcategory = $state('');
	let description = $state('');
	let dateTimeValue = $state('');
	let accounts = $state<Account[]>([]);
	let accountsBase = $state<Account[]>([]);
	let categories = $state<Category[]>([]);
	let subcategories = $state<Subcategory[]>([]);
	let saving = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const editing = $derived(!!transaction);

	$effect(() => {
		void $outboxTick;
		accounts = applyOutboxToAccounts(accountsBase, tz);
	});

	const accountOptions = $derived(accountSelectOptions(accounts));
	const pickableCategories = $derived.by(() => {
		const userCats = categories.filter((cat) => !cat.is_system);
		if ((!editing && !repeatFrom) || !categoryId) return userCats;
		const current = categories.find((cat) => cat.id === categoryId);
		if (current?.is_system && !userCats.some((cat) => cat.id === categoryId)) {
			return [...userCats, current];
		}
		return userCats;
	});
	const categoryOptions = $derived(categorySelectOptions(pickableCategories));
	const subcategoryOptions = $derived([
		{ value: '', label: '—' },
		...subcategorySelectOptions(subcategories)
	]);

	const isFuture = $derived.by(() => {
		if (!dateTimeValue) return false;
		try {
			return (
				fromDatetimeLocalValue(dateTimeValue, tz) > fromDatetimeLocalValue(nowDatetimeLocal(tz), tz)
			);
		} catch {
			return false;
		}
	});

	const selectedAccountRow = $derived(accounts.find((a) => a.id === selectedAccount));
	const creditCardNegativeWarning = $derived.by(() => {
		if (txType !== 'expense' || !selectedAccountRow || !isCreditCard(selectedAccountRow)) {
			return false;
		}
		const kopecks = toCents(amount);
		if (!kopecks || kopecks <= 0) return false;
		return creditCardExpenseWarning(selectedAccountRow.balance, kopecks);
	});

	$effect(() => {
		if (variant === 'modal' && !open) return;
		void init(transaction, repeatFrom, defaultType, initialDescription);
	});

	async function init(
		editSource: Transaction | null,
		repeatSource: Transaction | null,
		createType: 'expense' | 'income',
		createDescription: string
	) {
		if (editSource) {
			txType = editSource.type === 'income' ? 'income' : 'expense';
			amount = formatMoneyForInput(editSource.amount_display);
			selectedAccount = editSource.account_id;
			categoryId = editSource.category_id ?? '';
			subcategoryId = editSource.subcategory_id ?? '';
			newSubcategory = '';
			description = editSource.description ?? '';
			dateTimeValue = toDatetimeLocalValue(editSource.transaction_date, tz);
		} else if (repeatSource) {
			txType = repeatSource.type === 'income' ? 'income' : 'expense';
			amount = formatMoneyForInput(repeatSource.amount_display);
			selectedAccount = repeatSource.account_id;
			categoryId = repeatSource.category_id ?? '';
			subcategoryId = repeatSource.subcategory_id ?? '';
			newSubcategory = '';
			description = repeatSource.description ?? '';
			dateTimeValue = nowDatetimeLocal(tz);
		} else {
			txType = createType;
			amount = '';
			selectedAccount = '';
			categoryId = '';
			subcategoryId = '';
			newSubcategory = '';
			description = createDescription;
			dateTimeValue = nowDatetimeLocal(tz);
		}
		accountsBase = await listAccounts('active');
		accounts = applyOutboxToAccounts(accountsBase, tz);
		if (!editSource && !repeatSource) {
			selectedAccount = defaultAccountId(accounts, accountId);
		}
		await loadCategories();
	}

	async function loadCategories() {
		categories = await listCategories(txType);
		const selectable = categories.filter((c) => !c.is_system);
		if (!categoryId && selectable.length) {
			categoryId = selectable.find((c) => c.is_primary)?.id ?? selectable[0].id;
		}
		if (categoryId && !categories.some((c) => c.id === categoryId)) {
			categoryId = selectable.find((c) => c.is_primary)?.id ?? selectable[0]?.id ?? '';
		}
		if (categoryId) {
			subcategories = await listSubcategories(categoryId);
		} else {
			subcategories = [];
		}
	}

	async function onCategoryChange(nextCategoryId: string) {
		subcategoryId = '';
		newSubcategory = '';
		if (!nextCategoryId) {
			subcategories = [];
			return;
		}
		try {
			subcategories = await listSubcategories(nextCategoryId);
		} catch {
			subcategories = [];
		}
	}

	async function save(e: Event) {
		e.preventDefault();
		saving = true;
		try {
			const payload = {
				account_id: selectedAccount,
				type: txType,
				amount: toAPIAmount(amount),
				description: description || undefined,
				category_id: categoryId || undefined,
				subcategory_id: newSubcategory ? undefined : subcategoryId || undefined,
				subcategory_name: newSubcategory || undefined,
				transaction_date: fromDatetimeLocalValue(dateTimeValue, tz)
			};
			if (transaction) {
				await updateTransaction(transaction.id, payload);
			} else {
				await createTransaction(payload);
			}
			if (variant === 'modal') open = false;
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}

	function close() {
		if (variant === 'modal') open = false;
		onclose();
	}

	const pageTitle = $derived(
		editing
			? $_('transactions.edit')
			: txType === 'expense'
				? $_('transactions.type.expense')
				: $_('transactions.type.income')
	);
</script>

{#snippet formBody()}
	<form id="tx-form" class="space-y-4" onsubmit={save}>
		{#if editing}
			<p class="text-sm font-medium">
				{txType === 'expense' ? $_('transactions.type.expense') : $_('transactions.type.income')}
			</p>
		{/if}

		<Select
			id="tx-account"
			label={$_('transactions.field.account')}
			bind:value={selectedAccount}
			options={accountOptions}
			usePortal
		/>

		<Select
			id="tx-category"
			label={$_('transactions.field.category')}
			bind:value={categoryId}
			options={categoryOptions}
			usePortal
			onchange={(next) => void onCategoryChange(next)}
		/>

		<div>
			<Select
				id="tx-sub"
				label={$_('transactions.field.subcategory')}
				bind:value={subcategoryId}
				options={subcategoryOptions}
				usePortal
				onchange={() => {
					if (subcategoryId) newSubcategory = '';
				}}
			/>
			{#if !subcategoryId}
				<input
					class="input mt-2 w-full"
					placeholder={$_('transactions.field.newSubcategory')}
					bind:value={newSubcategory}
				/>
			{/if}
		</div>

		<div>
			<label class="mb-1 block text-sm font-medium" for="tx-amount"
				>{$_('transactions.field.amount')}</label
			>
			<MoneyInput id="tx-amount" bind:value={amount} required />
		</div>

		<div>
			<label class="mb-1 block text-sm font-medium" for="tx-desc"
				>{$_('transactions.field.description')}</label
			>
			<input id="tx-desc" class="input w-full" bind:value={description} />
		</div>

		<DateTimePicker
			id="tx-date"
			label={$_('transactions.field.dateOnly')}
			bind:value={dateTimeValue}
			{...editing ? operationDatetimePickerEdit : operationDatetimePickerCreate}
			usePortal
			required
		/>
		{#if isFuture}
			<div class="space-y-1">
				<p class="text-sm" style:color="var(--primary)">📅 {$_('transactions.planned')}</p>
				<FieldHint text={$_('transactions.field.plannedHint')} />
			</div>
		{/if}
		{#if creditCardNegativeWarning}
			<p class="text-sm" style:color="var(--warning)">
				{$_('accounts.creditCard.negativeBalance')}
			</p>
		{/if}
	</form>
{/snippet}

{#snippet formFooter()}
	<button type="button" class="btn-ghost" onclick={close}>{$_('common.cancel')}</button>
	<button type="submit" form="tx-form" class="btn-primary" disabled={saving}>
		{saving ? $_('common.loading') : $_('common.save')}
	</button>
{/snippet}

{#if variant === 'page'}
	<FormPageShell title={pageTitle} {backHref} onback={close}>
		{@render formBody()}
		{#snippet footer()}
			{@render formFooter()}
		{/snippet}
	</FormPageShell>
{:else}
	<ModalShell bind:open title={pageTitle} onclose={close}>
		{@render formBody()}
		{#snippet footer()}
			{@render formFooter()}
		{/snippet}
	</ModalShell>
{/if}
