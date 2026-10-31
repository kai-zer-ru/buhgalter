<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		createTransactionTemplate,
		deleteTransactionTemplate,
		listAccounts,
		listCategories,
		listMerchants,
		listSubcategories,
		listTags,
		listTransactionTemplates,
		reorderTransactionTemplates,
		updateTransactionTemplate,
		type Account,
		type Category,
		type Merchant,
		type Subcategory,
		type Tag,
		type Transaction,
		type TransactionTemplate,
		type TransactionTemplateUpsert
	} from '$lib/api/client';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import ReorderDragGhost from '$lib/components/ReorderDragGhost.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import Select from '$lib/components/Select.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import TransactionForm from '$lib/components/TransactionForm.svelte';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import {
		accountSelectOptions,
		categorySelectOptions,
		subcategorySelectOptions
	} from '$lib/select-options';
	import { beginPointerDrag, moveId, type DragGhostView } from '$lib/drag-reorder';
	import { confirm } from '$lib/confirm';
	import { fromCents, toCents } from '$lib/money';
	import { reportPageLoadFailure } from '$lib/page-load';
	import {
		hasTemplatePrefillWarnings,
		templateToRepeatFrom
	} from '$lib/transaction-template-prefill';
	import { toast } from '$lib/toast';
	import { matchRefByName, refsForAccount, remapRefIds } from '$lib/household/refs-for-account';

	let templates = $state<TransactionTemplate[]>([]);
	let accounts = $state<Account[]>([]);
	let expenseCategories = $state<Category[]>([]);
	let incomeCategories = $state<Category[]>([]);
	let merchants = $state<Merchant[]>([]);
	let allTags = $state<Tag[]>([]);
	let subcategories = $state<Subcategory[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let saving = $state(false);

	let formOpen = $state(false);
	let editingId = $state<string | null>(null);
	let txOpen = $state(false);
	let transferOpen = $state(false);
	let repeatTx = $state<Transaction | null>(null);
	let repeatTransfer = $state<Transaction | null>(null);
	let newTxType = $state<'expense' | 'income'>('expense');
	let name = $state('');
	let tplType = $state<'expense' | 'income' | 'transfer'>('expense');
	let accountId = $state('');
	let toAccountId = $state('');
	let categoryId = $state('');
	let subcategoryId = $state('');
	let merchantId = $state('');
	let selectedTagIds = $state<string[]>([]);
	let description = $state('');
	let amountFixed = $state(true);
	let amount = $state('');

	let dragGhost = $state<DragGhostView | null>(null);
	let draggingId = $state<string | null>(null);
	let overId = $state<string | null>(null);

	const selectedAcc = $derived(accounts.find((acc) => acc.id === accountId));
	const accountOptions = $derived(accountSelectOptions(accounts));
	const scopedCategories = $derived(
		refsForAccount(selectedAcc, tplType === 'income' ? incomeCategories : expenseCategories)
	);
	const categoryOptions = $derived(categorySelectOptions(scopedCategories));
	const subcategoryOptions = $derived(subcategorySelectOptions(subcategories));
	const scopedMerchants = $derived(refsForAccount(selectedAcc, merchants));
	const merchantOptions = $derived([
		{ value: '', label: '—' },
		...scopedMerchants.map((m) => ({ value: m.id, label: m.name }))
	]);
	const scopedTags = $derived(refsForAccount(selectedAcc, allTags));
	const typeOptions = $derived([
		{ value: 'expense', label: $_('transactions.type.expense') },
		{ value: 'income', label: $_('transactions.type.income') },
		{ value: 'transfer', label: $_('transactions.transfer') }
	]);

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		try {
			const [tpls, accs, exp, inc, merch, tags] = await Promise.all([
				listTransactionTemplates(),
				listAccounts('active'),
				listCategories('expense'),
				listCategories('income'),
				listMerchants(),
				listTags()
			]);
			templates = tpls;
			accounts = accs;
			expenseCategories = exp.filter((c) => !c.is_system);
			incomeCategories = inc.filter((c) => !c.is_system);
			merchants = merch;
			allTags = tags;
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: templates.length > 0 });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	async function loadSubs(catId: string) {
		if (!catId) {
			subcategories = [];
			return;
		}
		try {
			subcategories = await listSubcategories(catId);
		} catch {
			subcategories = [];
		}
	}

	function openCreate() {
		editingId = null;
		name = '';
		tplType = 'expense';
		accountId = accounts.find((a) => a.is_primary)?.id ?? accounts[0]?.id ?? '';
		toAccountId = '';
		categoryId = '';
		subcategoryId = '';
		merchantId = '';
		selectedTagIds = [];
		description = '';
		amountFixed = true;
		amount = '';
		subcategories = [];
		formOpen = true;
	}

	async function openEdit(t: TransactionTemplate) {
		editingId = t.id;
		name = t.name;
		tplType = t.type;
		accountId = t.account_id ?? '';
		toAccountId = t.to_account_id ?? '';
		categoryId = t.category_id ?? '';
		subcategoryId = t.subcategory_id ?? '';
		merchantId = t.merchant_id ?? '';
		selectedTagIds = (t.tags ?? []).map((x) => x.id);
		description = t.description ?? '';
		amountFixed = t.amount != null;
		amount = t.amount != null ? fromCents(t.amount) : '';
		await loadSubs(categoryId);
		formOpen = true;
	}

	function closeForm() {
		formOpen = false;
		editingId = null;
	}

	function onTemplateAccountChange() {
		const currentCat = (tplType === 'income' ? incomeCategories : expenseCategories).find(
			(c) => c.id === categoryId
		);
		const matchedCat = matchRefByName(currentCat, scopedCategories);
		if (currentCat && !matchedCat) {
			categoryId = scopedCategories[0]?.id ?? '';
			subcategoryId = '';
			void loadSubs(categoryId);
			toast($_('refs.reset.category'));
		} else if (matchedCat && matchedCat.id !== categoryId) {
			categoryId = matchedCat.id;
			void loadSubs(categoryId);
		}
		const currentMerchant = merchants.find((m) => m.id === merchantId);
		const matchedMerchant = matchRefByName(currentMerchant, scopedMerchants);
		if (currentMerchant && !matchedMerchant) {
			merchantId = '';
			toast($_('refs.reset.merchant'));
		} else if (matchedMerchant) {
			merchantId = matchedMerchant.id;
		}
		const selected = allTags.filter((t) => selectedTagIds.includes(t.id));
		const remapped = remapRefIds(selected, scopedTags);
		if (remapped.dropped) toast($_('refs.reset.tags'));
		selectedTagIds = remapped.next.map((t) => t.id);
	}

	function toggleTag(id: string) {
		if (selectedTagIds.includes(id)) {
			selectedTagIds = selectedTagIds.filter((x) => x !== id);
		} else {
			selectedTagIds = [...selectedTagIds, id];
		}
	}

	function buildPayload(): TransactionTemplateUpsert | null {
		const n = name.trim();
		if (!n) {
			toast($_('templates.err.name'));
			return null;
		}
		let amountCents: number | null = null;
		if (amountFixed) {
			try {
				amountCents = toCents(amount);
			} catch {
				toast($_('templates.err.amount'));
				return null;
			}
			if (amountCents <= 0) {
				toast($_('templates.err.amount'));
				return null;
			}
		}
		if (tplType === 'transfer') {
			if (!accountId || !toAccountId || accountId === toAccountId) {
				toast($_('templates.err.accounts'));
				return null;
			}
			return {
				name: n,
				type: 'transfer',
				account_id: accountId,
				to_account_id: toAccountId,
				amount: amountCents,
				description: description.trim() || null
			};
		}
		return {
			name: n,
			type: tplType,
			account_id: accountId || null,
			category_id: categoryId || null,
			subcategory_id: subcategoryId || null,
			merchant_id: merchantId || null,
			amount: amountCents,
			description: description.trim() || null,
			tag_ids: selectedTagIds
		};
	}

	async function saveForm(e: Event) {
		e.preventDefault();
		const payload = buildPayload();
		if (!payload) return;
		saving = true;
		try {
			if (editingId) {
				await updateTransactionTemplate(editingId, payload);
			} else {
				await createTransactionTemplate(payload);
			}
			toast($_('common.saved'));
			closeForm();
			await load();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}

	async function removeTemplate(t: TransactionTemplate) {
		const ok = await confirm({
			message: $_('templates.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteTransactionTemplate(t.id);
			toast($_('common.deleted'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	function createFromTemplate(tpl: TransactionTemplate) {
		const { tx, warnings } = templateToRepeatFrom(tpl, {
			activeAccountIds: new Set(accounts.map((a) => a.id)),
			categoryIds: new Set([...expenseCategories, ...incomeCategories].map((c) => c.id)),
			merchantIds: new Set(merchants.map((m) => m.id)),
			tagIds: new Set(allTags.map((t) => t.id))
		});
		if (hasTemplatePrefillWarnings(warnings)) {
			toast($_('templates.prefill.missing'));
		}
		if (tx.type === 'transfer') {
			repeatTx = null;
			repeatTransfer = tx;
			transferOpen = true;
			return;
		}
		repeatTransfer = null;
		repeatTx = tx;
		newTxType = tx.type === 'income' ? 'income' : 'expense';
		txOpen = true;
	}

	function rowActions(t: TransactionTemplate): RowAction[] {
		return [
			{
				icon: 'create',
				label: $_('templates.createTransaction'),
				onclick: () => void createFromTemplate(t)
			},
			{ icon: 'edit', label: $_('common.edit'), onclick: () => void openEdit(t) },
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void removeTemplate(t)
			}
		];
	}

	function typeLabel(type: TransactionTemplate['type']) {
		if (type === 'income') return $_('transactions.type.income');
		if (type === 'transfer') return $_('transactions.transfer');
		return $_('transactions.type.expense');
	}

	/** Account, category and merchant names for the row subtitle; unknown refs are skipped. */
	function templateDetails(t: TransactionTemplate): string {
		const accName = (id: string | null | undefined) => accounts.find((a) => a.id === id)?.name;
		if (t.type === 'transfer') {
			const from = accName(t.account_id);
			const to = accName(t.to_account_id);
			return from && to ? `${from} → ${to}` : (from ?? to ?? '');
		}
		const cats = t.type === 'income' ? incomeCategories : expenseCategories;
		return [
			accName(t.account_id),
			cats.find((c) => c.id === t.category_id)?.name,
			merchants.find((m) => m.id === t.merchant_id)?.name
		]
			.filter(Boolean)
			.join(' · ');
	}

	function templateCurrency(t: TransactionTemplate): string | undefined {
		return accounts.find((a) => a.id === t.account_id)?.currency;
	}

	function startDrag(e: PointerEvent, t: TransactionTemplate, rowEl: HTMLElement) {
		beginPointerDrag({
			e,
			id: t.id,
			rowEl,
			dragKind: 'template',
			isDisabled: () => formOpen,
			setGhost: (g: DragGhostView | null) => (dragGhost = g),
			setDraggingId: (id: string | null) => (draggingId = id),
			setOverId: (id: string | null) => (overId = id),
			onDrop: (from, to) => void dropTemplate(from, to)
		});
	}

	async function dropTemplate(fromId: string, toId: string) {
		const ids = moveId(
			templates.map((t) => t.id),
			fromId,
			toId
		);
		if (!ids) return;
		try {
			templates = await reorderTransactionTemplates(ids);
		} catch (err) {
			toast.fromError(err);
		}
	}

	$effect(() => {
		if (!formOpen || tplType === 'transfer') return;
		void loadSubs(categoryId);
	});
</script>

<div class="space-y-8">
	<SectionHeader title={$_('settings.tab.templates')}>
		{#snippet actions()}
			<button type="button" class="btn-primary" onclick={openCreate}>
				{$_('templates.add')}
			</button>
		{/snippet}
	</SectionHeader>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if templates.length === 0}
			<EmptyStateCard message={$_('templates.empty')} />
		{:else}
			<section class="space-y-3">
				<GroupHeading
					title={$_('templates.group.all', { values: { count: templates.length } })}
					hint={$_('templates.group.all_hint')}
				/>
				<div class="card list-divided p-0 sm:p-0">
					{#each templates as t (t.id)}
						{@const details = templateDetails(t)}
						<div
							class="transition-opacity"
							class:opacity-30={draggingId === t.id}
							data-drag-id={t.id}
							data-drag-kind="template"
							style:box-shadow={overId === t.id && draggingId !== null && draggingId !== t.id
								? 'inset 0 2px 0 var(--primary)'
								: undefined}
						>
							<div class="flex items-center gap-1 px-1.5 py-1.5 sm:px-2" data-drag-row>
								<span
									class="list-drag-handle"
									role="button"
									tabindex="-1"
									aria-label={$_('templates.drag.handle')}
									onpointerdown={(e) =>
										startDrag(e, t, e.currentTarget.closest('[data-drag-id]') as HTMLElement)}
								>
									⠿
								</span>
								<button
									type="button"
									class="list-row-button"
									title={$_('templates.createTransaction')}
									onclick={() => createFromTemplate(t)}
								>
									<span class="type-glyph type-{t.type}" aria-label={typeLabel(t.type)}>
										<svg
											aria-hidden="true"
											class="h-5 w-5"
											viewBox="0 0 24 24"
											fill="none"
											stroke="currentColor"
											stroke-width="2"
											stroke-linecap="round"
											stroke-linejoin="round"
										>
											{#if t.type === 'income'}
												<path d="M12 5v14M19 12l-7 7-7-7" />
											{:else if t.type === 'transfer'}
												<path d="M4 8h14l-4-4M20 16H6l4 4" />
											{:else}
												<path d="M12 19V5M5 12l7-7 7 7" />
											{/if}
										</svg>
									</span>
									<span class="min-w-0 flex-1">
										<span class="block truncate font-medium">{t.name}</span>
										<span class="block truncate text-xs" style:color="var(--text-muted)">
											{typeLabel(t.type)}{details ? ` · ${details}` : ''}
										</span>
									</span>
									<span class="shrink-0 text-right text-sm">
										{#if t.amount != null}
											<MoneyDisplay
												cents={t.amount}
												currency={templateCurrency(t)}
												class="font-medium tabular-nums"
											/>
										{:else}
											<span class="text-xs" style:color="var(--text-muted)"
												>{$_('templates.amount.open')}</span
											>
										{/if}
									</span>
								</button>
								<RowActionsMenu actions={rowActions(t)} />
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/if}
	</PageLoadGate>
</div>

{#if dragGhost}
	<ReorderDragGhost ghost={dragGhost} />
{/if}

{#if formOpen}
	<ModalShell
		open={true}
		title={editingId ? $_('templates.edit') : $_('templates.add')}
		onclose={closeForm}
		maxWidth="max-w-md"
	>
		<form class="space-y-3" onsubmit={saveForm}>
			<label class="block">
				<span class="field-label">{$_('templates.field.name')}</span>
				<input class="input w-full" bind:value={name} maxlength={80} required />
			</label>
			<div class="space-y-1">
				<span class="field-label">{$_('templates.field.type')}</span>
				<div class="segmented" role="radiogroup" aria-label={$_('templates.field.type')}>
					{#each typeOptions as opt (opt.value)}
						<button
							type="button"
							role="radio"
							aria-checked={tplType === opt.value}
							onclick={() => {
								if (tplType === opt.value) return;
								tplType = opt.value as 'expense' | 'income' | 'transfer';
								categoryId = '';
								subcategoryId = '';
								merchantId = '';
								selectedTagIds = [];
							}}
						>
							{opt.label}
						</button>
					{/each}
				</div>
			</div>
			{#if tplType === 'transfer'}
				<label class="block">
					<span class="field-label">{$_('transactions.field.from')}</span>
					<Select
						options={accountOptions}
						bind:value={accountId}
						onchange={() => onTemplateAccountChange()}
					/>
				</label>
				<label class="block">
					<span class="field-label">{$_('transactions.field.to')}</span>
					<Select options={accountOptions} bind:value={toAccountId} />
				</label>
			{:else}
				<label class="block">
					<span class="field-label">{$_('transactions.field.account')}</span>
					<Select
						options={accountOptions}
						bind:value={accountId}
						onchange={() => onTemplateAccountChange()}
					/>
				</label>
				<label class="block">
					<span class="field-label">{$_('transactions.field.category')}</span>
					<Select
						options={[{ value: '', label: '—' }, ...categoryOptions]}
						bind:value={categoryId}
						onchange={() => {
							subcategoryId = '';
						}}
					/>
				</label>
				<label class="block">
					<span class="field-label">{$_('transactions.field.subcategory')}</span>
					<Select
						options={[{ value: '', label: '—' }, ...subcategoryOptions]}
						bind:value={subcategoryId}
						disabled={!categoryId}
					/>
				</label>
				<label class="block">
					<span class="field-label">{$_('transactions.field.merchant')}</span>
					<Select options={merchantOptions} bind:value={merchantId} />
				</label>
				{#if scopedTags.length > 0}
					<div class="space-y-1">
						<span class="text-sm" style:color="var(--text-muted)"
							>{$_('transactions.field.tags')}</span
						>
						<div class="flex flex-wrap gap-1">
							{#each scopedTags as tag (tag.id)}
								<button
									type="button"
									class="rounded-full border px-2 py-0.5 text-xs"
									style:border-color={selectedTagIds.includes(tag.id)
										? 'var(--primary)'
										: 'var(--border)'}
									style:background={selectedTagIds.includes(tag.id)
										? 'color-mix(in srgb, var(--primary) 15%, transparent)'
										: 'transparent'}
									onclick={() => toggleTag(tag.id)}
								>
									{tag.name}
								</button>
							{/each}
						</div>
					</div>
				{/if}
			{/if}
			<div
				class="flex items-center justify-between gap-4 border-t pt-3"
				style:border-color="var(--border)"
			>
				<span class="text-sm">{$_('templates.field.amountFixed')}</span>
				<ToggleSwitch
					checked={amountFixed}
					label={$_('templates.field.amountFixed')}
					onchange={() => (amountFixed = !amountFixed)}
				/>
			</div>
			{#if amountFixed}
				<label class="block">
					<span class="field-label">{$_('transactions.field.amount')}</span>
					<MoneyInput bind:value={amount} />
				</label>
			{/if}
			<label class="block">
				<span class="field-label">{$_('transactions.field.description')}</span>
				<input class="input w-full" bind:value={description} maxlength={500} />
			</label>
			<div class="flex justify-end gap-2 pt-2">
				<button type="button" class="btn-ghost" onclick={closeForm}>{$_('common.cancel')}</button>
				<button type="submit" class="btn-primary" disabled={saving}>
					{$_('common.save')}
				</button>
			</div>
		</form>
	</ModalShell>
{/if}

<TransactionForm
	bind:open={txOpen}
	defaultType={newTxType}
	repeatFrom={repeatTx}
	onclose={() => {
		txOpen = false;
		repeatTx = null;
	}}
	onsaved={() => {
		txOpen = false;
		repeatTx = null;
		toast($_('common.saved'));
	}}
/>
<TransferForm
	bind:open={transferOpen}
	repeatFrom={repeatTransfer}
	onclose={() => {
		transferOpen = false;
		repeatTransfer = null;
	}}
	onsaved={() => {
		transferOpen = false;
		repeatTransfer = null;
		toast($_('common.saved'));
	}}
/>

<style>
	.type-glyph {
		display: inline-flex;
		height: 2.5rem;
		width: 2.5rem;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		border-radius: 0.75rem;
		color: var(--text-muted);
		background-color: color-mix(in srgb, var(--border) 45%, transparent);
	}

	.type-glyph.type-expense {
		color: var(--danger);
		background-color: color-mix(in srgb, var(--danger) 12%, transparent);
	}

	.type-glyph.type-income {
		color: var(--primary);
		background-color: color-mix(in srgb, var(--primary) 12%, transparent);
	}
</style>
