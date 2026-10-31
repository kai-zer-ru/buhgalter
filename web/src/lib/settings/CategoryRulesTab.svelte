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
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import ReorderDragGhost from '$lib/components/ReorderDragGhost.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
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

	/** Subcategory names for row titles; rules store only `subcategory_id`. */
	let subNames = $state<Record<string, string>>({});

	async function loadSubNames(list: CategoryRule[]) {
		const catIds = [...new Set(list.filter((r) => r.subcategory_id).map((r) => r.category_id))];
		if (catIds.length === 0) return;
		const groups = await Promise.all(catIds.map((id) => listSubcategories(id).catch(() => [])));
		subNames = Object.fromEntries(groups.flat().map((sub) => [sub.id, sub.name]));
	}

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
			void loadSubNames(rules);
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

	const ruleGroups = $derived(
		(['category_account', 'text_category'] as const)
			.map((k) => ({ kind: k, rules: rules.filter((r) => r.kind === k) }))
			.filter((g) => g.rules.length > 0)
	);

	function ruleCategory(rule: CategoryRule): Category | undefined {
		return (
			expenseCategories.find((c) => c.id === rule.category_id) ??
			incomeCategories.find((c) => c.id === rule.category_id)
		);
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
		const kindOf = rules.find((r) => r.id === fromId)?.kind;
		if (!kindOf || rules.find((r) => r.id === toId)?.kind !== kindOf) return;
		// Kinds resolve separately, so a drag only reorders its own group.
		const moved = moveId(
			rules.filter((r) => r.kind === kindOf).map((r) => r.id),
			fromId,
			toId
		);
		if (!moved) return;
		const ids = [...moved, ...rules.filter((r) => r.kind !== kindOf).map((r) => r.id)];
		try {
			rules = await reorderCategoryRules(ids);
		} catch (err) {
			toast.fromError(err);
		}
	}
</script>

<div class="space-y-8">
	<SectionHeader title={$_('categoryRules.title')}>
		{#snippet actions()}
			<button type="button" class="btn-primary" onclick={openCreate}>
				{$_('categoryRules.add')}
			</button>
		{/snippet}
	</SectionHeader>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if rules.length === 0}
			<EmptyStateCard message={$_('categoryRules.empty')} />
		{:else}
			<div class="space-y-8">
				{#each ruleGroups as group (group.kind)}
					<section class="space-y-3">
						<GroupHeading
							title={group.kind === 'category_account'
								? $_('categoryRules.kind.categoryAccount')
								: $_('categoryRules.kind.textCategory')}
							hint={group.kind === 'category_account'
								? $_('categoryRules.group.categoryAccount_hint')
								: $_('categoryRules.group.textCategory_hint')}
						/>
						<div class="card list-divided p-0 sm:p-0">
							{#each group.rules as rule, i (rule.id)}
								{@const cat = ruleCategory(rule)}
								<div
									class="transition-opacity"
									class:opacity-30={draggingId === rule.id}
									data-drag-id={rule.id}
									data-drag-kind="category_rule"
									style:box-shadow={overId === rule.id &&
									draggingId !== null &&
									draggingId !== rule.id
										? 'inset 0 2px 0 var(--primary)'
										: undefined}
								>
									<div class="flex items-center gap-1 px-1.5 py-1.5 sm:px-2" data-drag-row>
										<span
											class="list-drag-handle"
											role="button"
											tabindex="-1"
											aria-label={$_('categoryRules.drag.handle')}
											onpointerdown={(e) =>
												startDrag(
													e,
													rule,
													e.currentTarget.closest('[data-drag-id]') as HTMLElement
												)}
										>
											⠿
										</span>
										<span
											class="w-6 shrink-0 text-center text-xs tabular-nums"
											style:color="var(--text-muted)"
											title={$_('categoryRules.priority')}>{i + 1}</span
										>
										<div class="flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1 py-1.5">
											{#if rule.kind === 'text_category'}
												<span class="rule-pattern">«{rule.pattern ?? ''}»</span>
												{@render arrow()}
												{@render categoryChip(cat, rule)}
											{:else}
												{@render categoryChip(cat, rule)}
											{/if}
										</div>
										<RowActionsMenu actions={rowActions(rule)} />
									</div>
								</div>
							{/each}
						</div>
					</section>
				{/each}
			</div>
		{/if}
	</PageLoadGate>
</div>

{#snippet arrow()}
	<svg
		aria-hidden="true"
		class="h-4 w-4 shrink-0"
		style:color="var(--text-muted)"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
	>
		<path d="M5 12h14M13 6l6 6-6 6" />
	</svg>
{/snippet}

{#snippet categoryChip(cat: Category | undefined, rule: CategoryRule)}
	<span class="inline-flex min-w-0 items-center gap-2">
		<CategoryIcon icon={cat?.icon || 'default'} size={28} />
		<span class="truncate font-medium">
			{cat?.name ?? rule.category_id}{#if rule.subcategory_id && subNames[rule.subcategory_id]}
				<span style:color="var(--text-muted)">&nbsp;›&nbsp;</span>{subNames[
					rule.subcategory_id
				]}{/if}
		</span>
	</span>
{/snippet}

{#if dragGhost}
	<ReorderDragGhost ghost={dragGhost} />
{/if}

<ModalShell
	bind:open={formOpen}
	title={editingId ? $_('categoryRules.edit') : $_('categoryRules.add')}
	onclose={closeForm}
>
	<form class="space-y-4" onsubmit={save}>
		<div class="space-y-1">
			<span class="text-sm font-medium">{$_('categoryRules.field.kind')}</span>
			<div class="segmented" role="radiogroup" aria-label={$_('categoryRules.field.kind')}>
				{#each kindOptions as opt (opt.value)}
					<button
						type="button"
						role="radio"
						aria-checked={kind === opt.value}
						onclick={() => (kind = opt.value as CategoryRuleKind)}
					>
						{opt.label}
					</button>
				{/each}
			</div>
			<p class="text-xs" style:color="var(--text-muted)">
				{kind === 'category_account'
					? $_('categoryRules.group.categoryAccount_hint')
					: $_('categoryRules.group.textCategory_hint')}
			</p>
		</div>
		<div class="space-y-1">
			<span class="text-sm font-medium">{$_('categoryRules.field.categoryType')}</span>
			<div class="segmented" role="radiogroup" aria-label={$_('categoryRules.field.categoryType')}>
				{#each ['expense', 'income'] as const as type (type)}
					<button
						type="button"
						role="radio"
						aria-checked={ruleCategoryType === type}
						onclick={() => {
							if (ruleCategoryType === type) return;
							ruleCategoryType = type;
							categoryId = '';
							void onCategoryChange();
						}}
					>
						{$_(`transactions.type.${type}`)}
					</button>
				{/each}
			</div>
		</div>
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
		<div class="flex justify-end gap-2 border-t pt-4" style:border-color="var(--border)">
			<button type="button" class="btn-secondary" onclick={closeForm}>
				{$_('common.cancel')}
			</button>
			<button type="submit" class="btn-primary" disabled={saving}>
				{$_('common.save')}
			</button>
		</div>
	</form>
</ModalShell>

<style>
	.rule-pattern {
		max-width: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		border-radius: 0.5rem;
		padding: 0.125rem 0.5rem;
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		font-size: 0.8125rem;
		background-color: color-mix(in srgb, var(--border) 45%, transparent);
	}
</style>
