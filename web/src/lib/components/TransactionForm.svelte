<script lang="ts">
	import { _ } from 'svelte-i18n';
	import {
		createTransaction,
		updateTransaction,
		listAccounts,
		listCategories,
		listSubcategories,
		type Account,
		type Category,
		type Subcategory,
		type Transaction
	} from '$lib/api/client';
	import { ApiError } from '$lib/api/client';
	import { fromDatetimeLocalValue, nowDatetimeLocal, toDatetimeLocalValue } from '$lib/dates';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { formatMoneyDisplay, toAPIAmount } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	type Props = {
		open: boolean;
		accountId?: string;
		defaultType?: 'expense' | 'income';
		transaction?: Transaction | null;
		repeatFrom?: Transaction | null;
		onclose: () => void;
		onsaved: () => void;
	};

	let {
		open = $bindable(),
		accountId = '',
		defaultType = 'expense',
		transaction = null,
		repeatFrom = null,
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
	let categories = $state<Category[]>([]);
	let subcategories = $state<Subcategory[]>([]);
	let saving = $state(false);
	let error = $state('');

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const editing = $derived(!!transaction);

	const accountOptions = $derived(accounts.map((acc) => ({ value: acc.id, label: acc.name })));
	const pickableCategories = $derived.by(() => {
		const userCats = categories.filter((cat) => !cat.is_system);
		if ((!editing && !repeatFrom) || !categoryId) return userCats;
		const current = categories.find((cat) => cat.id === categoryId);
		if (current?.is_system && !userCats.some((cat) => cat.id === categoryId)) {
			return [...userCats, current];
		}
		return userCats;
	});
	const categoryOptions = $derived(
		pickableCategories.map((cat) => ({ value: cat.id, label: cat.name }))
	);
	const subcategoryOptions = $derived([
		{ value: '', label: '—' },
		...subcategories.map((sub) => ({ value: sub.id, label: sub.name }))
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

	$effect(() => {
		if (!open) return;
		void init(transaction, repeatFrom, defaultType);
	});

	async function init(
		editSource: Transaction | null,
		repeatSource: Transaction | null,
		createType: 'expense' | 'income'
	) {
		error = '';
		if (editSource) {
			txType = editSource.type === 'income' ? 'income' : 'expense';
			amount = formatMoneyDisplay(editSource.amount_display);
			selectedAccount = editSource.account_id;
			categoryId = editSource.category_id ?? '';
			subcategoryId = editSource.subcategory_id ?? '';
			newSubcategory = '';
			description = editSource.description ?? '';
			dateTimeValue = toDatetimeLocalValue(editSource.transaction_date, tz);
		} else if (repeatSource) {
			txType = repeatSource.type === 'income' ? 'income' : 'expense';
			amount = formatMoneyDisplay(repeatSource.amount_display);
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
			description = '';
			dateTimeValue = nowDatetimeLocal(tz);
		}
		accounts = await listAccounts('active');
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

	async function onCategoryChange() {
		subcategoryId = '';
		newSubcategory = '';
		subcategories = categoryId ? await listSubcategories(categoryId) : [];
	}

	async function save(e: Event) {
		e.preventDefault();
		saving = true;
		error = '';
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
			open = false;
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			saving = false;
		}
	}

	function close() {
		open = false;
		onclose();
	}
</script>

<ModalShell
	bind:open
	title={editing
		? $_('transactions.edit')
		: txType === 'expense'
			? $_('transactions.type.expense')
			: $_('transactions.type.income')}
	onclose={close}
>
	<form id="tx-form" class="space-y-4" onsubmit={save}>
		{#if editing}
			<p class="text-sm font-medium">
				{txType === 'expense' ? $_('transactions.type.expense') : $_('transactions.type.income')}
			</p>
		{/if}

		<div>
			<label class="mb-1 block text-sm font-medium" for="tx-amount"
				>{$_('transactions.field.amount')}</label
			>
			<MoneyInput id="tx-amount" bind:value={amount} required />
		</div>

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
			onchange={() => void onCategoryChange()}
		/>

		<div>
			<Select
				id="tx-sub"
				label={$_('transactions.field.subcategory')}
				bind:value={subcategoryId}
				options={subcategoryOptions}
				usePortal
			/>
			<input
				class="input mt-2 w-full"
				placeholder={$_('transactions.field.newSubcategory')}
				bind:value={newSubcategory}
			/>
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
			timeMode="optional"
			defaultTime={editing ? 'preserve' : 'now'}
			usePortal
			required
		/>
		{#if isFuture}
			<div class="space-y-1">
				<p class="text-sm" style:color="var(--primary)">📅 {$_('transactions.planned')}</p>
				<FieldHint text={$_('transactions.field.plannedHint')} />
			</div>
		{/if}

		<FormFeedback {error} />
	</form>

	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={close}>{$_('common.cancel')}</button>
		<button type="submit" form="tx-form" class="btn-primary" disabled={saving}>
			{saving ? $_('common.loading') : $_('common.save')}
		</button>
	{/snippet}
</ModalShell>
