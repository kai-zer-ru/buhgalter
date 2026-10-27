<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		createCategoryRule,
		deleteCategoryRule,
		listAccounts,
		listCategories,
		listCategoryRules,
		listSubcategories,
		reorderCategoryRules,
		updateCategoryRule,
		type Account,
		type Category,
		type CategoryRule,
		type CategoryRuleKind,
		type Subcategory
	} from '$lib/api/client';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
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
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { matchRefByName, refsForAccount } from '$lib/household/refs-for-account';

	let rules = $state<CategoryRule[]>([]);
	let accounts = $state<Account[]>([]);
	let expenseCategories = $state<Category[]>([]);
	let incomeCategories = $state<Category[]>([]);
	let subcategories = $state<Subcategory[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let formOpen = $state(false);
	let editingId = $state<string | null>(null);
	let saving = $state(false);

	let kind = $state<CategoryRuleKind>('category_account');
	let categoryId = $state('');
	let subcategoryId = $state('');
	let defaultAccountId = $state('');
	let pattern = $state('');
	let ruleCategoryType = $state<'expense' | 'income'>('expense');

	let dragGhost = $state<DragGhostView | null>(null);
	let draggingId = $state<string | null>(null);
	let overId = $state<string | null>(null);

	const accountOptions = $derived(accountSelectOptions(accounts));
	const selectedAcc = $derived(accounts.find((acc) => acc.id === defaultAccountId));
	const scopedRuleCategories = $derived(
		kind === 'category_account'
			? refsForAccount(
					selectedAcc,
					ruleCategoryType === 'income' ? incomeCategories : expenseCategories
				)
			: ruleCategoryType === 'income'
				? incomeCategories
				: expenseCategories
	);
	const categoryOptions = $derived(
		categorySelectOptions(
			scopedRuleCategories.filter((c) => kind !== 'text_category' || !c.is_system)
		)
	);
	const subcategoryOptions = $derived(subcategorySelectOptions(subcategories));

	const kindOptions = $derived([
		{ value: 'category_account', label: $_('categoryRules.kind.categoryAccount') },
		{ value: 'text_category', label: $_('categoryRules.kind.textCategory') }
	]);

	onMount(() => void load());

	async function load() {
		loading = true;
		loadError = null;
		try {
			const [r, acc, exp, inc] = await Promise.all([
				listCategoryRules(),
				listAccounts('active'),
				listCategories('expense'),
				listCategories('income')
			]);
			rules = r ?? [];
			accounts = acc ?? [];
			expenseCategories = exp ?? [];
			incomeCategories = inc ?? [];
		} catch (e) {
			loadError = reportPageLoadFailure(e);
		} finally {
			loading = false;
		}
	}

	function closeForm() {
		formOpen = false;
	}

	function openCreate() {
		editingId = null;
		kind = 'category_account';
		ruleCategoryType = 'expense';
		categoryId = '';
		subcategoryId = '';
		defaultAccountId = '';
		pattern = '';
		subcategories = [];
		formOpen = true;
	}

	async function openEdit(rule: CategoryRule) {
		editingId = rule.id;
		kind = rule.kind;
		categoryId = rule.category_id;
		subcategoryId = rule.subcategory_id ?? '';
		defaultAccountId = rule.default_account_id ?? '';
		pattern = rule.pattern ?? '';
		const cat =
			expenseCategories.find((c) => c.id === rule.category_id) ??
			incomeCategories.find((c) => c.id === rule.category_id);
		ruleCategoryType = cat?.type === 'income' ? 'income' : 'expense';
		subcategories = categoryId ? await listSubcategories(categoryId).catch(() => []) : [];
		formOpen = true;
	}

	async function onCategoryChange() {
		subcategoryId = '';
		subcategories = categoryId ? await listSubcategories(categoryId).catch(() => []) : [];
	}

	function onRuleAccountChange() {
		if (kind !== 'category_account') return;
		const current = (ruleCategoryType === 'income' ? incomeCategories : expenseCategories).find(
			(c) => c.id === categoryId
		);
		const matched = matchRefByName(current, scopedRuleCategories);
		if (current && !matched) {
			categoryId = scopedRuleCategories[0]?.id ?? '';
			void onCategoryChange();
			toast($_('refs.reset.category'));
		} else if (matched && matched.id !== categoryId) {
			categoryId = matched.id;
			void onCategoryChange();
		}
	}

	async function save(e: Event) {
		e.preventDefault();
		saving = true;
		try {
			const payload = {
				kind,
				category_id: categoryId,
				subcategory_id: subcategoryId || undefined,
				default_account_id: kind === 'category_account' ? defaultAccountId || undefined : undefined,
				pattern: kind === 'text_category' ? pattern.trim() || undefined : undefined
			};
			if (editingId) {
				await updateCategoryRule(editingId, payload);
			} else {
				await createCategoryRule(payload);
			}
			formOpen = false;
			toast.success($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}

	async function removeRule(rule: CategoryRule) {
		const ok = await confirm({
			message: $_('categoryRules.deleteConfirm'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteCategoryRule(rule.id);
			toast.success($_('common.deleted'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	function ruleLabel(rule: CategoryRule): string {
		const cats = [...expenseCategories, ...incomeCategories];
		const cat = cats.find((c) => c.id === rule.category_id);
		const catName = cat?.name ?? rule.category_id;
		if (rule.kind === 'text_category') {
			return `${rule.pattern ?? ''} → ${catName}`;
		}
		const acc = accounts.find((a) => a.id === rule.default_account_id);
		const accName = acc?.name ?? $_('categoryRules.accountShort');
		return `${catName}${rule.subcategory_id ? ' / …' : ''} → ${accName}`;
	}

	function rowActions(rule: CategoryRule): RowAction[] {
		return [
			{ icon: 'edit', label: $_('common.edit'), onclick: () => void openEdit(rule) },
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void removeRule(rule)
			}
		];
	}

	function startDrag(e: PointerEvent, rule: CategoryRule, rowEl: HTMLElement) {
		beginPointerDrag({
			e,
			id: rule.id,
			rowEl,
			dragKind: 'category_rule',
			isDisabled: () => formOpen,
			setGhost: (g) => (dragGhost = g),
			setDraggingId: (id) => (draggingId = id),
			setOverId: (id) => (overId = id),
			onDrop: (from, to) => void dropRule(from, to)
		});
	}

	async function dropRule(fromId: string, toId: string) {
		const ids = moveId(
			rules.map((r) => r.id),
			fromId,
			toId
		);
		if (!ids) return;
		try {
			rules = await reorderCategoryRules(ids);
		} catch (err) {
			toast.fromError(err);
		}
	}
</script>

<div class="space-y-4">
	<div class="flex justify-end">
		<button type="button" class="btn btn-primary" onclick={openCreate}>
			{$_('categoryRules.add')}
		</button>
	</div>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if rules.length === 0}
			<EmptyStateCard message={$_('categoryRules.empty')} />
		{:else}
			<div class="space-y-2">
				{#each rules as rule (rule.id)}
					<div
						class="card transition-opacity"
						class:opacity-30={draggingId === rule.id}
						class:border-t-2={overId === rule.id && draggingId !== null && draggingId !== rule.id}
						data-drag-id={rule.id}
						data-drag-kind="category_rule"
						style:border-color={overId === rule.id ? 'var(--primary)' : undefined}
					>
						<div class="flex items-center gap-1" data-drag-row>
							<span
								class="btn-icon btn-ghost cursor-grab touch-none text-lg leading-none select-none active:cursor-grabbing"
								role="button"
								tabindex="-1"
								aria-label={$_('categoryRules.drag.handle')}
								onpointerdown={(e) =>
									startDrag(e, rule, e.currentTarget.closest('[data-drag-id]') as HTMLElement)}
							>
								⠿
							</span>
							<div class="min-w-0 flex-1">
								<div class="truncate font-medium">{ruleLabel(rule)}</div>
								<div class="text-xs" style:color="var(--text-muted)">
									{rule.kind === 'category_account'
										? $_('categoryRules.kind.categoryAccount')
										: $_('categoryRules.kind.textCategory')}
									· {$_('categoryRules.priority')}
									{rule.priority}
								</div>
							</div>
							<RowActionsMenu actions={rowActions(rule)} />
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

<ModalShell
	bind:open={formOpen}
	title={editingId ? $_('categoryRules.edit') : $_('categoryRules.add')}
	onclose={closeForm}
>
	<form class="space-y-4" onsubmit={save}>
		<Select
			label={$_('categoryRules.field.kind')}
			bind:value={kind}
			options={kindOptions}
			usePortal
		/>
		<Select
			label={$_('categoryRules.field.categoryType')}
			bind:value={ruleCategoryType}
			options={[
				{ value: 'expense', label: $_('transactions.type.expense') },
				{ value: 'income', label: $_('transactions.type.income') }
			]}
			usePortal
			onchange={() => {
				categoryId = '';
				void onCategoryChange();
			}}
		/>
		<Select
			label={$_('transactions.field.category')}
			bind:value={categoryId}
			options={categoryOptions}
			usePortal
			onchange={() => void onCategoryChange()}
		/>
		<Select
			label={$_('transactions.field.subcategory')}
			bind:value={subcategoryId}
			options={[{ value: '', label: '—' }, ...subcategoryOptions]}
			usePortal
		/>
		{#if kind === 'category_account'}
			<Select
				label={$_('transactions.field.account')}
				bind:value={defaultAccountId}
				options={accountOptions}
				onchange={() => onRuleAccountChange()}
				usePortal
			/>
		{:else}
			<label class="block space-y-1">
				<span class="text-sm font-medium">{$_('categoryRules.field.pattern')}</span>
				<input class="input w-full" bind:value={pattern} maxlength={200} required />
			</label>
		{/if}
		<div class="flex justify-end gap-2">
			<button type="button" class="btn-secondary" onclick={closeForm}>
				{$_('common.cancel')}
			</button>
			<button type="submit" class="btn-primary" disabled={saving}>
				{$_('common.save')}
			</button>
		</div>
	</form>
</ModalShell>
