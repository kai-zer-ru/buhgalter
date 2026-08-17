<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
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
		type TransactionTemplate,
		type TransactionTemplateUpsert
	} from '$lib/api/client';
	import { transactionNewPath, transferNewPath } from '$lib/android/form-routes';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import ReorderDragGhost from '$lib/components/ReorderDragGhost.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import Select from '$lib/components/Select.svelte';
	import {
		accountSelectOptions,
		categorySelectOptions,
		subcategorySelectOptions
	} from '$lib/select-options';
	import { beginPointerDrag, moveId, type DragGhostView } from '$lib/drag-reorder';
	import { confirm } from '$lib/confirm';
	import { fromCents, toCents } from '$lib/money';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';

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

	const accountOptions = $derived(accountSelectOptions(accounts));
	const categoryOptions = $derived(
		categorySelectOptions(tplType === 'income' ? incomeCategories : expenseCategories)
	);
	const subcategoryOptions = $derived(subcategorySelectOptions(subcategories));
	const merchantOptions = $derived([
		{ value: '', label: '—' },
		...merchants.map((m) => ({ value: m.id, label: m.name }))
	]);
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
		const from = '/settings/transaction-templates';
		if (tpl.type === 'transfer') {
			void goto(resolve(transferNewPath({ templateId: tpl.id, from })));
			return;
		}
		void goto(
			resolve(
				transactionNewPath({
					type: tpl.type === 'income' ? 'income' : 'expense',
					templateId: tpl.id,
					from
				})
			)
		);
	}

	function rowActions(t: TransactionTemplate): RowAction[] {
		return [
			{
				icon: 'create',
				label: $_('templates.createTransaction'),
				onclick: () => createFromTemplate(t)
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

	function amountLabel(t: TransactionTemplate) {
		if (t.amount == null) return $_('templates.amount.open');
		return fromCents(t.amount);
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

<div class="space-y-4">
	<div class="flex justify-end">
		<button type="button" class="btn btn-primary" onclick={openCreate}>
			{$_('templates.add')}
		</button>
	</div>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if templates.length === 0}
			<EmptyStateCard message={$_('templates.empty')} />
		{:else}
			<div class="space-y-2">
				{#each templates as t (t.id)}
					<div
						class="card transition-opacity"
						class:opacity-30={draggingId === t.id}
						class:border-t-2={overId === t.id && draggingId !== null && draggingId !== t.id}
						data-drag-id={t.id}
						data-drag-kind="template"
						style:border-color={overId === t.id ? 'var(--primary)' : undefined}
					>
						<div class="flex items-center gap-1" data-drag-row>
							<span
								class="btn-icon btn-ghost cursor-grab touch-none text-lg leading-none select-none active:cursor-grabbing"
								role="button"
								tabindex="-1"
								aria-label={$_('templates.drag.handle')}
								onpointerdown={(e) =>
									startDrag(e, t, e.currentTarget.closest('[data-drag-id]') as HTMLElement)}
							>
								⠿
							</span>
							<div class="min-w-0 flex-1">
								<div class="truncate font-medium">{t.name}</div>
								<div class="text-xs" style:color="var(--text-muted)">
									{typeLabel(t.type)} · {amountLabel(t)}
								</div>
							</div>
							<RowActionsMenu actions={rowActions(t)} />
						</div>
					</div>
				{/each}
			</div>
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
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('templates.field.name')}</span>
				<input class="input w-full" bind:value={name} maxlength={80} required />
			</label>
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('templates.field.type')}</span>
				<Select
					options={typeOptions}
					value={tplType}
					onchange={(v) => {
						tplType = v as 'expense' | 'income' | 'transfer';
						categoryId = '';
						subcategoryId = '';
						merchantId = '';
						selectedTagIds = [];
					}}
				/>
			</label>
			{#if tplType === 'transfer'}
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('transactions.field.from')}</span
					>
					<Select options={accountOptions} bind:value={accountId} />
				</label>
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)">{$_('transactions.field.to')}</span>
					<Select options={accountOptions} bind:value={toAccountId} />
				</label>
			{:else}
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('transactions.field.account')}</span
					>
					<Select options={accountOptions} bind:value={accountId} />
				</label>
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('transactions.field.category')}</span
					>
					<Select
						options={[{ value: '', label: '—' }, ...categoryOptions]}
						bind:value={categoryId}
						onchange={() => {
							subcategoryId = '';
						}}
					/>
				</label>
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('transactions.field.subcategory')}</span
					>
					<Select
						options={[{ value: '', label: '—' }, ...subcategoryOptions]}
						bind:value={subcategoryId}
						disabled={!categoryId}
					/>
				</label>
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('transactions.field.merchant')}</span
					>
					<Select options={merchantOptions} bind:value={merchantId} />
				</label>
				{#if allTags.length > 0}
					<div class="space-y-1">
						<span class="text-sm" style:color="var(--text-muted)"
							>{$_('transactions.field.tags')}</span
						>
						<div class="flex flex-wrap gap-1">
							{#each allTags as tag (tag.id)}
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
			<label class="flex items-center gap-2 text-sm">
				<input type="checkbox" bind:checked={amountFixed} />
				{$_('templates.field.amountFixed')}
			</label>
			{#if amountFixed}
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('transactions.field.amount')}</span
					>
					<MoneyInput bind:value={amount} />
				</label>
			{/if}
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)"
					>{$_('transactions.field.description')}</span
				>
				<input class="input w-full" bind:value={description} maxlength={500} />
			</label>
			<div class="flex justify-end gap-2 pt-2">
				<button type="button" class="btn btn-ghost" onclick={closeForm}
					>{$_('common.cancel')}</button
				>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{$_('common.save')}
				</button>
			</div>
		</form>
	</ModalShell>
{/if}
