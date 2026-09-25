<script lang="ts">
	import { untrack } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		listAccounts,
		listCategories,
		listCredits,
		listMerchants,
		listSubcategories,
		listTags,
		type Account,
		type Category,
		type Credit,
		type Merchant,
		type Subcategory,
		type Tag,
		type TagRef,
		type Transaction
	} from '$lib/api/client';
	import { createTransaction, updateTransaction } from '$lib/offline/transactions-api';
	import { addCreditPayment } from '$lib/offline/credits-api';
	import { applyOutboxToAccounts } from '$lib/offline/local-state';
	import { outboxTick } from '$lib/offline/store';
	import { invalidateRefCache, subcategoriesRefPath } from '$lib/offline/ref-cache';
	import { fromDatetimeLocalValue, nowDatetimeLocal, toDatetimeLocalValue } from '$lib/dates';
	import { buildDatetimeLocal, parseDatetimeLocal } from '$lib/datetime-picker';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import {
		operationDatetimePickerCreate,
		operationDatetimePickerEdit
	} from '$lib/datetime-picker-standards';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import Combobox from '$lib/components/Combobox.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { creditCardExpenseWarning, isCreditCard } from '$lib/credit-card';
	import { formatMoneyForInput, fromCents, toAPIAmount, toCents } from '$lib/money';
	import {
		accountSelectOptions,
		categorySelectOptions,
		subcategorySelectOptions
	} from '$lib/select-options';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import { suggestCategoryFromMerchant } from '$lib/android/notification-intercept/category-suggest';

	function isCreditsExpenseCategory(cat: Category | undefined): boolean {
		return Boolean(cat?.is_system && cat.name === 'Кредиты');
	}

	type CreatePrefill = {
		description?: string;
		amount?: string;
		accountId?: string;
		merchantId?: string;
		merchantName?: string;
		categoryId?: string;
		subcategoryId?: string;
		/** ISO datetime */
		occurredAt?: string;
	};

	type Props = {
		variant?: 'modal' | 'page';
		open?: boolean;
		backHref?: string;
		accountId?: string;
		defaultType?: 'expense' | 'income';
		transaction?: Transaction | null;
		repeatFrom?: Transaction | null;
		initialDescription?: string;
		/** Richer create prefill (bank notification intercept, etc.). */
		createPrefill?: CreatePrefill | null;
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
		createPrefill = null,
		onclose,
		onsaved
	}: Props = $props();

	let txType = $state<'expense' | 'income'>('expense');
	let amount = $state('');
	let selectedAccount = $state('');
	let categoryId = $state('');
	let subcategoryId = $state('');
	let subcategoryQuery = $state('');
	let creditId = $state('');
	let activeCredits = $state<Credit[]>([]);
	let merchantId = $state('');
	let merchantQuery = $state('');
	let selectedTags = $state<TagRef[]>([]);
	let tagInput = $state('');
	let description = $state('');
	let dateTimeValue = $state('');
	let accounts = $state<Account[]>([]);
	let accountsBase = $state<Account[]>([]);
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

	$effect(() => {
		void $outboxTick;
		accounts = applyOutboxToAccounts(accountsBase, tz);
	});

	const accountOptions = $derived(accountSelectOptions(accounts));
	const pickableCategories = $derived.by(() => {
		const userCats = categories.filter((cat) => !cat.is_system);
		const creditsCat = categories.find((cat) => isCreditsExpenseCategory(cat));
		let list = userCats;
		if (txType === 'expense' && creditsCat && !editing) {
			list = [...userCats, creditsCat];
		}
		if ((!editing && !repeatFrom) || !categoryId) return list;
		const current = categories.find((cat) => cat.id === categoryId);
		if (current?.is_system && !list.some((cat) => cat.id === categoryId)) {
			return [...list, current];
		}
		return list;
	});
	const categoryOptions = $derived(categorySelectOptions(pickableCategories));
	const selectedCategory = $derived(categories.find((c) => c.id === categoryId));
	const isCreditPaymentMode = $derived(
		!editing && txType === 'expense' && isCreditsExpenseCategory(selectedCategory)
	);
	const subcategoryOptions = $derived(subcategorySelectOptions(subcategories));
	const creditOptions = $derived(
		activeCredits.map((c) => ({
			value: c.id,
			label: c.name?.trim() || $_('credits.unnamed')
		}))
	);
	const merchantOptions = $derived(
		merchants.map((m) => ({
			value: m.id,
			label: m.name,
			icon: { type: 'category' as const, icon: m.icon || 'default' }
		}))
	);
	const newSubcategoryName = $derived.by(() => {
		if (subcategoryId) return '';
		const trimmed = subcategoryQuery.trim();
		if (!trimmed || isLikelyUuid(trimmed)) return '';
		return trimmed;
	});
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

	const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

	function isLikelyUuid(value: string): boolean {
		return UUID_RE.test(value.trim());
	}

	async function loadSubcategoriesForCategory(
		categoryId: string,
		requiredSubId = ''
	): Promise<Subcategory[]> {
		let list = await listSubcategories(categoryId);
		if (requiredSubId && !list.some((s) => s.id === requiredSubId)) {
			invalidateRefCache(subcategoriesRefPath(categoryId));
			list = await listSubcategories(categoryId);
		}
		return list;
	}

	$effect(() => {
		if (variant === 'modal' && !open) return;
		const editSource = transaction;
		const repeatSource = repeatFrom;
		const createType = defaultType;
		const desc = initialDescription;
		const prefill = createPrefill;
		void untrack(() => init(editSource, repeatSource, createType, desc, prefill));
	});

	async function init(
		editSource: Transaction | null,
		repeatSource: Transaction | null,
		createType: 'expense' | 'income',
		createDescription: string,
		prefill: CreatePrefill | null
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
			dateTimeValue = toDatetimeLocalValue(editSource.transaction_date, tz);
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
			dateTimeValue = nowDatetimeLocal(tz);
			timeExpanded = false;
			optionalDetailsOpen = Boolean(merchantId || selectedTags.length || description.trim());
		} else {
			txType = createType;
			amount = prefill?.amount ? formatMoneyForInput(prefill.amount) : '';
			selectedAccount = '';
			categoryId = prefill?.categoryId ?? '';
			subcategoryId = prefill?.subcategoryId ?? '';
			subcategoryQuery = '';
			merchantId = prefill?.merchantId ?? '';
			merchantQuery = prefill?.merchantName ?? '';
			selectedTags = [];
			tagInput = '';
			description = prefill?.description ?? createDescription;
			if (prefill?.occurredAt) {
				try {
					dateTimeValue = toDatetimeLocalValue(prefill.occurredAt, tz);
				} catch {
					dateTimeValue = nowDatetimeLocal(tz);
				}
			} else {
				dateTimeValue = nowDatetimeLocal(tz);
			}
			timeExpanded = Boolean(prefill?.occurredAt);
			optionalDetailsOpen = Boolean(
				merchantId || merchantQuery.trim() || description.trim() || selectedTags.length
			);
			if (prefill?.merchantId && !prefill.categoryId) {
				const suggest = await suggestCategoryFromMerchant(prefill.merchantId, createType);
				if (suggest) {
					categoryId = suggest.categoryId;
					subcategoryId = suggest.subcategoryId ?? '';
				}
			}
		}
		accountsBase = (await listAccounts('active').catch(() => [] as Account[])) ?? [];
		accounts = applyOutboxToAccounts(accountsBase, tz);
		if (!editSource && !repeatSource) {
			const preferred = prefill?.accountId || accountId;
			selectedAccount = defaultAccountId(accounts, preferred);
		}
		const [merchantsRes, tagsRes] = await Promise.allSettled([listMerchants(), listTags()]);
		merchants = merchantsRes.status === 'fulfilled' ? merchantsRes.value : [];
		allTags = tagsRes.status === 'fulfilled' ? tagsRes.value : [];
		await loadCategories();
	}

	async function loadCategories() {
		try {
			categories = await listCategories(txType);
		} catch {
			categories = [];
		}
		const selectable = pickableCategories;
		if (!categoryId && selectable.length) {
			categoryId = selectable.find((c) => c.is_primary)?.id ?? selectable[0].id;
		}
		if (categoryId && !categories.some((c) => c.id === categoryId)) {
			categoryId = selectable.find((c) => c.is_primary)?.id ?? selectable[0]?.id ?? '';
		}
		const cat = categories.find((c) => c.id === categoryId);
		if (!editing && txType === 'expense' && isCreditsExpenseCategory(cat)) {
			subcategories = [];
			creditId = '';
			try {
				activeCredits = await listCredits({ status: 'active' });
			} catch {
				activeCredits = [];
			}
			return;
		}
		activeCredits = [];
		creditId = '';
		if (categoryId) {
			try {
				subcategories = await loadSubcategoriesForCategory(categoryId, subcategoryId);
			} catch {
				subcategories = [];
			}
		} else {
			subcategories = [];
		}
		if (subcategoryId && !subcategories.some((s) => s.id === subcategoryId)) {
			const nameFromTx = subcategoryQuery.trim();
			subcategoryId = '';
			if (!nameFromTx || isLikelyUuid(nameFromTx)) {
				subcategoryQuery = '';
			}
		}
	}

	function clearOptionalDetails() {
		merchantId = '';
		merchantQuery = '';
		selectedTags = [];
		tagInput = '';
		description = '';
		optionalDetailsOpen = false;
	}

	async function onCategoryChange(nextCategoryId: string) {
		creditId = '';
		activeCredits = [];
		subcategoryId = '';
		subcategoryQuery = '';
		const cat = categories.find((c) => c.id === nextCategoryId);
		if (!editing && txType === 'expense' && isCreditsExpenseCategory(cat)) {
			subcategories = [];
			clearOptionalDetails();
			try {
				activeCredits = await listCredits({ status: 'active' });
			} catch {
				activeCredits = [];
			}
			return;
		}
		if (!nextCategoryId) {
			subcategories = [];
			return;
		}
		try {
			subcategories = await loadSubcategoriesForCategory(nextCategoryId);
		} catch {
			subcategories = [];
		}
	}

	function onCreditChange(nextCreditId: string) {
		const credit = activeCredits.find((c) => c.id === nextCreditId);
		if (!credit) return;
		if (credit.next_payment_amount != null && credit.next_payment_amount > 0) {
			amount = formatMoneyForInput(fromCents(credit.next_payment_amount));
		} else if (credit.monthly_payment > 0) {
			amount = formatMoneyForInput(fromCents(credit.monthly_payment));
		}
		if (credit.next_payment_date) {
			dateTimeValue = toDatetimeLocalValue(credit.next_payment_date, tz);
		}
		if (credit.debit_account_id && accounts.some((a) => a.id === credit.debit_account_id)) {
			selectedAccount = credit.debit_account_id;
		}
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
			if (isCreditPaymentMode) {
				if (!creditId) {
					toast.error($_('credits.error.pickCredit'));
					return;
				}
				await addCreditPayment(creditId, {
					amount: toAPIAmount(amount),
					payment_date: fromDatetimeLocalValue(dateTimeValue, tz),
					account_id: selectedAccount || undefined
				});
				if (variant === 'modal') open = false;
				toast($_('common.saved'));
				onsaved();
				return;
			}
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
			if (variant === 'modal') open = false;
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
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

		{#if isCreditPaymentMode}
			<Select
				id="tx-credit"
				label={$_('credits.field.credit')}
				bind:value={creditId}
				options={creditOptions}
				usePortal
				onchange={(next) => onCreditChange(next)}
			/>
			{#if activeCredits.length === 0}
				<p class="text-sm" style:color="var(--text-muted)">{$_('credits.empty')}</p>
			{/if}
		{:else}
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
		{/if}

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

		{#snippet timeField()}
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
		{/snippet}

		{#if isCreditPaymentMode}
			{@render timeField()}
		{:else}
			<details bind:open={optionalDetailsOpen}>
				<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
					{$_('transactions.field.optionalDetails')}
				</summary>
				<div class="mt-3 space-y-4">
					{@render timeField()}

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
