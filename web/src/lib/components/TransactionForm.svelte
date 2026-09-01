<script lang="ts">
	import { untrack } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		createTransaction,
		updateTransaction,
		listAccounts,
		listCategories,
		listSubcategories,
		listMerchants,
		listTags,
		type Account,
		type Category,
		type Subcategory,
		type Merchant,
		type Tag,
		type TagRef,
		type Transaction
	} from '$lib/api/client';
	import { fromDatetimeLocalValue, nowDatetimeLocal, toDatetimeLocalValue } from '$lib/dates';
	import { buildDatetimeLocal, parseDatetimeLocal } from '$lib/datetime-picker';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import {
		operationDatetimePickerCreate,
		operationDatetimePickerEdit
	} from '$lib/datetime-picker-standards';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import Combobox from '$lib/components/Combobox.svelte';
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
	let subcategoryQuery = $state('');
	let merchantId = $state('');
	let merchantQuery = $state('');
	let selectedTags = $state<TagRef[]>([]);
	let tagInput = $state('');
	let description = $state('');
	let dateTimeValue = $state('');
	let accounts = $state<Account[]>([]);
	let categories = $state<Category[]>([]);
	let subcategories = $state<Subcategory[]>([]);
	let merchants = $state<Merchant[]>([]);
	let allTags = $state<Tag[]>([]);
	let saving = $state(false);
	let optionalDetailsOpen = $state(false);
	let timeExpanded = $state(false);
	let timeInput = $state('');

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const editing = $derived(!!transaction);

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
	const subcategoryOptions = $derived(subcategorySelectOptions(subcategories));
	const merchantOptions = $derived(
		merchants.map((m) => ({
			value: m.id,
			label: m.name,
			icon: { type: 'category' as const, icon: m.icon || 'default' }
		}))
	);
	const newSubcategoryName = $derived(subcategoryId ? '' : subcategoryQuery.trim());
	const newMerchantName = $derived(merchantId ? '' : merchantQuery.trim());

	$effect(() => {
		const p = parseDatetimeLocal(dateTimeValue);
		if (!p) return;
		const next = `${String(p.hour).padStart(2, '0')}:${String(p.minute).padStart(2, '0')}`;
		if (untrack(() => timeInput) !== next) timeInput = next;
	});

	const tagSuggestions = $derived.by(() => {
		const q = tagInput.trim().toLowerCase();
		const selected = new Set(selectedTags.map((t) => t.name.toLowerCase()));
		return allTags
			.filter((t) => !selected.has(t.name.toLowerCase()))
			.filter((t) => !q || t.name.toLowerCase().includes(q))
			.slice(0, 8);
	});

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
		if (!open) return;
		// Capture props as deps; run init untracked so its $state writes / tz reads
		// cannot re-enter this effect (effect_update_depth_exceeded on edit/repeat).
		const editSource = transaction;
		const repeatSource = repeatFrom;
		const createType = defaultType;
		void untrack(() => init(editSource, repeatSource, createType));
	});

	async function init(
		editSource: Transaction | null,
		repeatSource: Transaction | null,
		createType: 'expense' | 'income'
	) {
		if (editSource) {
			txType = editSource.type === 'income' ? 'income' : 'expense';
			amount = formatMoneyForInput(editSource.amount_display);
			selectedAccount = editSource.account_id;
			categoryId = editSource.category_id ?? '';
			subcategoryId = editSource.subcategory_id ?? '';
			subcategoryQuery = editSource.subcategory_name ?? '';
			merchantId = editSource.merchant_id ?? '';
			merchantQuery = editSource.merchant_name ?? '';
			selectedTags = [...(editSource.tags ?? [])];
			tagInput = '';
			description = editSource.description ?? '';
			dateTimeValue = toDatetimeLocalValue(
				editSource.transaction_date,
				untrack(() => tz)
			);
			timeExpanded = false;
			optionalDetailsOpen = Boolean(merchantId || selectedTags.length || description.trim());
		} else if (repeatSource) {
			txType = repeatSource.type === 'income' ? 'income' : 'expense';
			amount = formatMoneyForInput(repeatSource.amount_display);
			selectedAccount = repeatSource.account_id;
			categoryId = repeatSource.category_id ?? '';
			subcategoryId = repeatSource.subcategory_id ?? '';
			subcategoryQuery = repeatSource.subcategory_name ?? '';
			merchantId = repeatSource.merchant_id ?? '';
			merchantQuery = repeatSource.merchant_name ?? '';
			selectedTags = [...(repeatSource.tags ?? [])];
			tagInput = '';
			description = repeatSource.description ?? '';
			dateTimeValue = nowDatetimeLocal(untrack(() => tz));
			timeExpanded = false;
			optionalDetailsOpen = Boolean(merchantId || selectedTags.length || description.trim());
		} else {
			txType = createType;
			amount = '';
			selectedAccount = '';
			categoryId = '';
			subcategoryId = '';
			subcategoryQuery = '';
			merchantId = '';
			merchantQuery = '';
			timeExpanded = false;
			optionalDetailsOpen = false;
			selectedTags = [];
			tagInput = '';
			description = '';
			dateTimeValue = nowDatetimeLocal(untrack(() => tz));
		}
		accounts = (await listAccounts('active').catch(() => [] as Account[])) ?? [];
		if (!editSource && !repeatSource) {
			selectedAccount = defaultAccountId(accounts, accountId);
		}
		const [merchantsRes, tagsRes] = await Promise.allSettled([listMerchants(), listTags()]);
		merchants = merchantsRes.status === 'fulfilled' ? merchantsRes.value : [];
		allTags = tagsRes.status === 'fulfilled' ? tagsRes.value : [];
		try {
			await loadCategories();
		} catch {
			categories = [];
			subcategories = [];
		}
	}

	async function loadCategories() {
		try {
			categories = await listCategories(txType);
		} catch {
			categories = [];
		}
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
		subcategoryQuery = '';
		subcategories = categoryId ? await listSubcategories(categoryId) : [];
	}

	function applyOperationTime() {
		timeExpanded = true;
		const p = parseDatetimeLocal(dateTimeValue);
		if (!p) return;
		const [h, m] = (timeInput || '00:00').split(':').map(Number);
		dateTimeValue = buildDatetimeLocal(p.year, p.month, p.day, h || 0, m || 0);
	}

	async function save(e: Event) {
		e.preventDefault();
		saving = true;
		try {
			const tagIds = selectedTags.filter((t) => t.id).map((t) => t.id);
			const tagNames = selectedTags.filter((t) => !t.id).map((t) => t.name);
			const payload = {
				account_id: selectedAccount,
				type: txType,
				amount: toAPIAmount(amount),
				description: description || undefined,
				category_id: categoryId || undefined,
				subcategory_id: newSubcategoryName ? undefined : subcategoryId || undefined,
				subcategory_name: newSubcategoryName || undefined,
				merchant_id: newMerchantName ? undefined : merchantId || undefined,
				merchant_name: newMerchantName || undefined,
				tag_ids: tagIds,
				tag_names: tagNames,
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
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}

	function close() {
		open = false;
		onclose();
	}

	function addTag(name: string, id = '') {
		const trimmed = name.trim();
		if (!trimmed) return;
		if (selectedTags.some((t) => t.name.toLowerCase() === trimmed.toLowerCase())) {
			tagInput = '';
			return;
		}
		selectedTags = [...selectedTags, { id, name: trimmed }];
		tagInput = '';
	}

	function removeTag(name: string) {
		selectedTags = selectedTags.filter((t) => t.name !== name);
	}

	function onTagKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			const match = tagSuggestions.find(
				(t) => t.name.toLowerCase() === tagInput.trim().toLowerCase()
			);
			if (match) addTag(match.name, match.id);
			else addTag(tagInput);
		}
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

		<Combobox
			id="tx-sub"
			label={$_('transactions.field.subcategory')}
			bind:value={subcategoryId}
			bind:query={subcategoryQuery}
			options={subcategoryOptions}
			usePortal
			allowCreate
			placeholder={$_('transactions.field.newSubcategory')}
			createLabel={$_('transactions.field.createNamed', {
				values: { name: subcategoryQuery.trim() || '…' }
			})}
			emptyLabel={$_('common.notFound')}
		/>

		<div>
			<label class="mb-1 block text-sm font-medium" for="tx-amount"
				>{$_('transactions.field.amount')}</label
			>
			<MoneyInput id="tx-amount" bind:value={amount} required />
		</div>

		<DateTimePicker
			id="tx-date"
			label={$_('transactions.field.dateOnly')}
			bind:value={dateTimeValue}
			bind:timeExpanded
			showOptionalTimeUI={false}
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

		<details bind:open={optionalDetailsOpen}>
			<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
				{$_('transactions.field.optionalDetails')}
			</summary>
			<div class="mt-3 space-y-4">
				<div>
					<label class="mb-1 block text-sm font-medium" for="tx-time"
						>{$_('transactions.field.timeOptional')}</label
					>
					<input
						id="tx-time"
						type="time"
						class="input w-full"
						bind:value={timeInput}
						onchange={applyOperationTime}
					/>
					<FieldHint text={$_('transactions.field.timeHint')} />
				</div>

				<div>
					<label class="mb-1 block text-sm font-medium" for="tx-desc"
						>{$_('transactions.field.description')}</label
					>
					<input id="tx-desc" class="input w-full" bind:value={description} />
				</div>

				<Combobox
					id="tx-merchant"
					label={$_('transactions.field.merchant')}
					bind:value={merchantId}
					bind:query={merchantQuery}
					options={merchantOptions}
					usePortal
					allowCreate
					placeholder={$_('transactions.field.newMerchant')}
					createLabel={$_('transactions.field.createNamed', {
						values: { name: merchantQuery.trim() || '…' }
					})}
					emptyLabel={$_('common.notFound')}
				/>

				<div>
					<span class="mb-1 block text-sm font-medium">{$_('transactions.field.tags')}</span>
					{#if selectedTags.length}
						<div class="mb-2 flex flex-wrap gap-2">
							{#each selectedTags as t (t.name)}
								<button
									type="button"
									class="rounded-md px-2 py-1 text-sm"
									style:background="var(--surface-2)"
									onclick={() => removeTag(t.name)}
								>
									{t.name} ×
								</button>
							{/each}
						</div>
					{/if}
					<input
						id="tx-tags"
						class="input w-full"
						placeholder={$_('transactions.field.tagsHint')}
						bind:value={tagInput}
						onkeydown={onTagKeydown}
					/>
					{#if tagInput.trim() && tagSuggestions.length}
						<ul
							class="mt-1 max-h-40 overflow-auto rounded-md border text-sm"
							style:border-color="var(--border)"
						>
							{#each tagSuggestions as sug (sug.id)}
								<li>
									<button
										type="button"
										class="block w-full px-3 py-2 text-left hover:opacity-80"
										onclick={() => addTag(sug.name, sug.id)}
									>
										{sug.name}
									</button>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</div>
		</details>

		{#if creditCardNegativeWarning}
			<p class="text-sm" style:color="var(--warning)">
				{$_('accounts.creditCard.negativeBalance')}
			</p>
		{/if}
	</form>

	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={close}>{$_('common.cancel')}</button>
		<button type="submit" form="tx-form" class="btn-primary" disabled={saving}>
			{saving ? $_('common.loading') : $_('common.save')}
		</button>
	{/snippet}
</ModalShell>
