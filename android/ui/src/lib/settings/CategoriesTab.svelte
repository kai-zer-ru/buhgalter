<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { replaceState } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _, locale } from 'svelte-i18n';
	import {
		listCategories,
		listSubcategories,
		reorderCategories,
		reorderSubcategories,
		setPrimaryCategory,
		createSubcategory,
		deleteSubcategory,
		updateSubcategory,
		type Category,
		type Subcategory
	} from '$lib/api/client';
	import { createCategory, deleteCategory, updateCategory } from '$lib/offline/categories-api';
	import { requireOnline } from '$lib/offline/require-online';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import CategoryIconPicker from '$lib/components/CategoryIconPicker.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import ReorderDragGhost from '$lib/components/ReorderDragGhost.svelte';
	import SubcategoryFormDialog from '$lib/components/SubcategoryFormDialog.svelte';
	import { defaultIconForKind } from '$lib/category-icons';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { categoriesRefPath, refCacheUpdate } from '$lib/offline/ref-cache';
	import {
		onSubcategoriesReordered,
		onSubcategoryCreated,
		onSubcategoryDeleted,
		onSubcategoryUpdated
	} from '$lib/offline/ref-cache-mutations';
	import { refCachePathMatches } from '$lib/offline/ref-cache-watch';
	import { beginPointerDrag, moveId, type DragGhostView } from '$lib/drag-reorder';
	import CatalogScopeTabs from '$lib/components/CatalogScopeTabs.svelte';
	import { householdMe, inHousehold } from '$lib/household/membership';

	type Tab = 'expense' | 'income';
	type SubDialogState =
		| { mode: 'create'; categoryId: string; name: string; icon: string }
		| { mode: 'edit'; categoryId: string; subId: string; name: string; icon: string };

	const categoryIconSize = 40;
	const subIconSize = 32;

	function tabFromSearchParams(params: URLSearchParams): Tab {
		return params.get('type') === 'income' ? 'income' : 'expense';
	}

	function defaultSubIconFor(cat: Category): string {
		return cat.icon || defaultIconForKind(tab);
	}

	let tab = $state<Tab>(tabFromSearchParams(get(page).url.searchParams));
	let catalogScope = $state<'personal' | 'household'>('personal');
	let categories = $state<Category[]>([]);
	let subs = $state<Record<string, Subcategory[]>>({});
	let expanded = $state<Record<string, boolean>>({});
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editIcon = $state('default');

	let newName = $state('');
	let newIcon = $state(defaultIconForKind('expense'));

	let subDialog = $state<SubDialogState | null>(null);
	let subDialogSaving = $state(false);

	let dragGhost = $state<DragGhostView | null>(null);
	let draggingId = $state<string | null>(null);
	let overId = $state<string | null>(null);

	function dragBindings(isDisabled: () => boolean, onDrop: (from: string, to: string) => void) {
		return {
			isDisabled,
			setGhost: (g: DragGhostView | null) => (dragGhost = g),
			setDraggingId: (id: string | null) => (draggingId = id),
			setOverId: (id: string | null) => (overId = id),
			onDrop
		};
	}

	function startCategoryDrag(e: PointerEvent, cat: Category, rowEl: HTMLElement) {
		beginPointerDrag({
			e,
			id: cat.id,
			rowEl,
			dragKind: 'category',
			...dragBindings(
				() => editingId !== null,
				(from, to) => void dropCategory(from, to)
			)
		});
	}

	function startSubDrag(e: PointerEvent, categoryId: string, sub: Subcategory, rowEl: HTMLElement) {
		beginPointerDrag({
			e,
			id: sub.id,
			rowEl,
			dragKind: 'sub',
			...dragBindings(
				() => subDialog !== null,
				(from, to) => void dropSub(categoryId, from, to)
			)
		});
	}

	onMount(() => {
		tab = tabFromSearchParams(get(page).url.searchParams);
		newIcon = defaultIconForKind(tab);
		const syncTabFromLocation = () => {
			const next = tabFromSearchParams(new URL(window.location.href).searchParams);
			if (next === tab) return;
			tab = next;
			newIcon = defaultIconForKind(next);
			expanded = {};
			void load();
		};
		window.addEventListener('popstate', syncTabFromLocation);
		void load();
		return () => window.removeEventListener('popstate', syncTabFromLocation);
	});

	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || loading) return;
		if (refCachePathMatches(update.path, [categoriesRefPath(tab), '/api/v1/ui/meta'])) {
			void load({ background: true });
		}
	});

	async function load(opts: { background?: boolean } = {}) {
		if (!opts.background) loading = true;
		try {
			categories = await listCategories(tab, inHousehold($householdMe) ? catalogScope : undefined);
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: categories.length > 0 });
			if (msg) loadError = msg;
		} finally {
			if (!opts.background) loading = false;
		}
	}

	function selectTab(next: Tab) {
		tab = next;
		newIcon = defaultIconForKind(next);
		expanded = {};
		const url = new URL(get(page).url);
		url.searchParams.delete('tab');
		if (next === 'expense') {
			url.searchParams.delete('type');
		} else {
			url.searchParams.set('type', next);
		}
		const search = url.searchParams.toString();
		const categoriesUrl = search ? `${resolve('/categories')}?${search}` : resolve('/categories');
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path
		replaceState(categoriesUrl, {});
		void load();
	}

	function isSubscriptionsSystemCategory(cat: Category): boolean {
		return cat.is_system && cat.name === 'Подписки';
	}

	function canExpandCategory(cat: Category): boolean {
		return !cat.is_system || isSubscriptionsSystemCategory(cat);
	}

	/** Loaded list wins over the server count: it reflects subs added or removed here. */
	function subCount(cat: Category): number {
		return subs[cat.id]?.length ?? cat.subcategory_count;
	}

	function subCountLabel(count: number): string {
		const form = new Intl.PluralRules($locale ?? 'ru').select(count);
		return $_(`categories.sub.count.${form}`, { values: { count } });
	}

	async function toggleExpand(cat: Category) {
		if (!canExpandCategory(cat)) return;
		const opening = !expanded[cat.id];
		expanded = { ...expanded, [cat.id]: opening };
		if (!opening) return;

		if (subs[cat.id]) return;
		try {
			subs = { ...subs, [cat.id]: await listSubcategories(cat.id) };
		} catch {
			subs = { ...subs, [cat.id]: [] };
		}
	}

	async function addCategory() {
		if (!newName.trim()) return;
		try {
			const category = await createCategory({
				name: newName.trim(),
				type: tab,
				icon: newIcon,
				scope: inHousehold($householdMe) ? catalogScope : undefined
			});
			categories = [...categories, category];
			newName = '';
			newIcon = defaultIconForKind(tab);
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		}
	}

	function startEdit(cat: Category) {
		editingId = cat.id;
		editName = cat.name;
		editIcon = cat.icon;
	}

	async function saveEdit() {
		if (!editingId) return;
		try {
			const updated = await updateCategory(editingId, {
				name: editName,
				icon: editIcon,
				type: tab
			});
			categories = categories.map((c) => (c.id === updated.id ? updated : c));
			editingId = null;
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function removeCategory(id: string) {
		const ok = await confirm({
			message: $_('categories.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteCategory(id, tab);
			categories = categories.filter((c) => c.id !== id);
			toast($_('common.deleted'));
		} catch (err) {
			toast.fromError(err);
		}
	}

	function openCreateSub(cat: Category) {
		if (!requireOnline('offline.onlineOnly.categoriesStructure')) return;
		subDialog = {
			mode: 'create',
			categoryId: cat.id,
			name: '',
			icon: defaultSubIconFor(cat)
		};
	}

	function openEditSub(categoryId: string, sub: Subcategory) {
		if (!requireOnline('offline.onlineOnly.categoriesStructure')) return;
		subDialog = {
			mode: 'edit',
			categoryId,
			subId: sub.id,
			name: sub.name,
			icon: sub.icon || 'default'
		};
	}

	function closeSubDialog() {
		if (subDialogSaving) return;
		subDialog = null;
	}

	async function saveSubDialog(payload: { name: string; icon: string }) {
		if (!subDialog || subDialogSaving) return;
		if (!requireOnline('offline.onlineOnly.categoriesStructure')) return;
		subDialogSaving = true;
		try {
			if (subDialog.mode === 'create') {
				const categoryId = subDialog.categoryId;
				const sub = await createSubcategory(categoryId, payload);
				onSubcategoryCreated(sub);
				subs = { ...subs, [categoryId]: [...(subs[categoryId] ?? []), sub] };
			} else {
				const { categoryId, subId } = subDialog;
				const updated = await updateSubcategory(subId, payload);
				onSubcategoryUpdated(updated);
				subs = {
					...subs,
					[categoryId]: (subs[categoryId] ?? []).map((s) => (s.id === updated.id ? updated : s))
				};
			}
			subDialog = null;
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			subDialogSaving = false;
		}
	}

	async function removeSub(categoryId: string, subId: string) {
		const ok = await confirm({
			message: $_('categories.confirm.deleteSub'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		if (!requireOnline('offline.onlineOnly.categoriesStructure')) return;
		try {
			await deleteSubcategory(subId);
			onSubcategoryDeleted(categoryId, subId);
			toast($_('common.deleted'));
			subs = {
				...subs,
				[categoryId]: (subs[categoryId] ?? []).filter((s) => s.id !== subId)
			};
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function dropCategory(fromId: string, toId: string) {
		if (!requireOnline('offline.onlineOnly.categoriesStructure')) return;
		const userCategories = categories.filter((c) => !c.is_system);
		const systemCategories = categories.filter((c) => c.is_system);
		const reordered = moveId(
			userCategories.map((c) => c.id),
			fromId,
			toId
		);
		if (!reordered) return;
		const ids = [...reordered, ...systemCategories.map((c) => c.id)];
		try {
			categories = await reorderCategories(tab, ids);
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function dropSub(categoryId: string, fromId: string, toId: string) {
		if (!requireOnline('offline.onlineOnly.categoriesStructure')) return;
		const list = subs[categoryId] ?? [];
		const ids = moveId(
			list.map((s) => s.id),
			fromId,
			toId
		);
		if (!ids) return;
		try {
			const reordered = await reorderSubcategories(categoryId, ids);
			onSubcategoriesReordered(categoryId, reordered);
			subs = { ...subs, [categoryId]: reordered };
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function makePrimary(id: string) {
		if (categories.find((c) => c.id === id)?.is_primary) return;
		if (!requireOnline('offline.onlineOnly.primary')) return;
		try {
			await setPrimaryCategory(id);
			categories = categories.map((c) => ({ ...c, is_primary: c.id === id }));
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		}
	}

	function categoryActions(cat: Category): RowAction[] {
		const actions: RowAction[] = [
			{
				icon: 'edit',
				label: $_('accounts.action.edit'),
				onclick: () => startEdit(cat)
			}
		];
		if (!cat.is_primary) {
			actions.push({
				icon: 'save',
				label: $_('categories.primary.set'),
				onclick: () => void makePrimary(cat.id)
			});
		}
		actions.push({
			icon: 'delete',
			label: $_('common.delete'),
			variant: 'danger',
			onclick: () => removeCategory(cat.id)
		});
		return actions;
	}
</script>

{#if dragGhost}
	<ReorderDragGhost ghost={dragGhost} />
{/if}

{#snippet chevron(open: boolean)}
	<svg
		class="h-5 w-5 shrink-0 transition-transform duration-150"
		class:rotate-180={open}
		style:color="var(--text-muted)"
		aria-hidden="true"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="m6 9 6 6 6-6" />
	</svg>
{/snippet}

{#snippet categoryRow(cat: Category)}
	{@const open = (expanded[cat.id] ?? false) && canExpandCategory(cat)}
	{@const count = subCount(cat)}
	<div
		class="category-row transition-opacity"
		class:opacity-30={draggingId === cat.id}
		data-drag-id={cat.id}
		data-drag-kind="category"
		style:box-shadow={overId === cat.id && draggingId !== null && draggingId !== cat.id
			? 'inset 0 2px 0 var(--primary)'
			: undefined}
	>
		{#if editingId === cat.id}
			<div class="flex items-start gap-3 p-3 sm:p-4">
				<span class="inline-flex min-h-11 min-w-11 shrink-0 items-center justify-center">
					<CategoryIcon icon={editIcon} size={categoryIconSize} />
				</span>
				<div class="flex min-w-0 flex-1 flex-col gap-3">
					<input class="input w-full" bind:value={editName} />
					<CategoryIconPicker
						bind:value={editIcon}
						bind:categoryName={editName}
						categoryType={tab}
						lockName={true}
						quickSize={categoryIconSize}
						iconSize={categoryIconSize}
					/>
					<div class="flex flex-wrap justify-end gap-2">
						<button type="button" class="btn-ghost" onclick={() => (editingId = null)}>
							{$_('common.cancel')}
						</button>
						<button type="button" class="btn-primary" onclick={saveEdit}>
							{$_('common.save')}
						</button>
					</div>
				</div>
			</div>
		{:else}
			<div class="flex items-center gap-1 px-1.5 py-1.5 sm:px-2" data-drag-row>
				{#if !cat.is_system}
					<span
						class="list-drag-handle"
						role="button"
						tabindex="-1"
						aria-label={$_('categories.drag.handle')}
						onpointerdown={(e) =>
							startCategoryDrag(e, cat, e.currentTarget.closest('[data-drag-id]') as HTMLElement)}
					>
						⠿
					</span>
				{:else}
					<span class="w-2 shrink-0" aria-hidden="true"></span>
				{/if}
				{#snippet rowBody()}
					<span class="inline-flex min-h-11 min-w-11 shrink-0 items-center justify-center">
						<CategoryIcon icon={cat.icon} size={categoryIconSize} />
					</span>
					<span class="min-w-0 flex-1">
						<span class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
							<span class="truncate font-medium">{cat.name}</span>
							{#if cat.is_primary}
								<span
									class="badge badge-success"
									title={$_('categories.primary.badge')}
									aria-label={$_('categories.primary.badge')}
								>
									{$_('categories.primary.short')}
								</span>
							{/if}
						</span>
						{#if count > 0}
							<span class="block text-xs" style:color="var(--text-muted)">
								{subCountLabel(count)}
							</span>
						{/if}
					</span>
				{/snippet}
				{#if canExpandCategory(cat)}
					<button
						type="button"
						class="list-row-button"
						aria-expanded={open}
						onclick={() => toggleExpand(cat)}
					>
						{@render rowBody()}
						{@render chevron(open)}
					</button>
				{:else}
					<div class="list-row-button cursor-default">
						{@render rowBody()}
					</div>
				{/if}
				{#if !cat.is_system}
					<RowActionsMenu actions={categoryActions(cat)} />
				{/if}
			</div>
		{/if}

		{#if open}
			{@const readOnlySubs = isSubscriptionsSystemCategory(cat)}
			<div class="sub-list">
				{#each subs[cat.id] ?? [] as sub (sub.id)}
					<div
						class="transition-opacity"
						class:opacity-30={!readOnlySubs && draggingId === sub.id}
						data-drag-id={sub.id}
						data-drag-kind="sub"
						style:box-shadow={!readOnlySubs &&
						overId === sub.id &&
						draggingId !== null &&
						draggingId !== sub.id
							? 'inset 0 2px 0 var(--primary)'
							: undefined}
					>
						<div class="flex min-w-0 items-center gap-1 py-1 pr-1.5 sm:pr-2" data-drag-row>
							{#if !readOnlySubs}
								<span
									class="list-drag-handle"
									role="button"
									tabindex="-1"
									aria-label={$_('categories.drag.handle')}
									onpointerdown={(e) =>
										startSubDrag(
											e,
											cat.id,
											sub,
											e.currentTarget.closest('[data-drag-id]') as HTMLElement
										)}
								>
									⠿
								</span>
							{:else}
								<span class="w-2 shrink-0" aria-hidden="true"></span>
							{/if}
							<span class="inline-flex min-h-10 min-w-10 shrink-0 items-center justify-center">
								<CategoryIcon icon={sub.icon || 'default'} size={subIconSize} />
							</span>
							<span class="min-w-0 flex-1 truncate pl-2 text-sm">{sub.name}</span>
							{#if !readOnlySubs}
								<RowActionsMenu
									actions={[
										{
											icon: 'edit',
											label: $_('accounts.action.edit'),
											onclick: () => openEditSub(cat.id, sub)
										},
										{
											icon: 'delete',
											label: $_('common.delete'),
											variant: 'danger',
											onclick: () => removeSub(cat.id, sub.id)
										}
									]}
								/>
							{/if}
						</div>
					</div>
				{:else}
					{#if subs[cat.id]}
						<p class="py-2 pl-4 text-sm" style:color="var(--text-muted)">
							{$_('categories.sub.empty')}
						</p>
					{/if}
				{/each}
				{#if !readOnlySubs}
					<div class="py-2 pr-2 pl-3">
						<button type="button" class="add-sub-btn" onclick={() => openCreateSub(cat)}>
							<svg
								aria-hidden="true"
								class="h-4 w-4"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
							>
								<path d="M12 5v14M5 12h14" />
							</svg>
							{$_('categories.sub.addButton')}
						</button>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/snippet}

<div class="space-y-8">
	<div class="space-y-4">
		<CatalogScopeTabs
			mode="catalog"
			value={catalogScope}
			onchange={(next) => {
				catalogScope = next === 'household' ? 'household' : 'personal';
				void load();
			}}
		/>
		<div class="page-tabs-scroll">
			<div class="page-tabs-row">
				<button
					type="button"
					class="tab shrink-0 {tab === 'expense' ? 'tab-active' : ''}"
					onclick={() => selectTab('expense')}
				>
					{$_('categories.tab.expense')}
				</button>
				<button
					type="button"
					class="tab shrink-0 {tab === 'income' ? 'tab-active' : ''}"
					onclick={() => selectTab('income')}
				>
					{$_('categories.tab.income')}
				</button>
			</div>
		</div>
	</div>

	<section class="card space-y-3">
		<div class="space-y-1">
			<h2 class="font-semibold">{$_('categories.add')}</h2>
			<p class="text-sm" style:color="var(--text-muted)">{$_('categories.add_hint')}</p>
		</div>
		<div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
			<input
				class="input min-w-[12rem] flex-1"
				placeholder={$_('categories.field.name')}
				bind:value={newName}
				onkeydown={(e) => {
					if (e.key === 'Enter') void addCategory();
				}}
			/>
			<CategoryIconPicker
				bind:value={newIcon}
				bind:categoryName={newName}
				categoryType={tab}
				quickLimit={5}
				fill
			/>
			<button type="button" class="btn-primary w-full sm:w-auto sm:px-4" onclick={addCategory}>
				{$_('common.create')}
			</button>
		</div>
	</section>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if categories.length === 0}
			<EmptyStateCard message={$_('categories.empty')} />
		{:else}
			{@const userCategories = categories.filter((c) => !c.is_system)}
			{@const systemCategories = categories.filter((c) => c.is_system)}
			<div class="space-y-8">
				{#if userCategories.length > 0}
					<section class="space-y-3">
						<GroupHeading
							title={$_('categories.group.mine', { values: { count: userCategories.length } })}
							hint={$_('categories.group.mine_hint')}
						/>
						<div class="card list-divided p-0 sm:p-0">
							{#each userCategories as cat (cat.id)}
								{@render categoryRow(cat)}
							{/each}
						</div>
					</section>
				{/if}
				{#if systemCategories.length > 0}
					<section class="space-y-3">
						<GroupHeading
							title={$_('categories.group.system')}
							hint={$_('categories.group.system_hint')}
						/>
						<div class="card list-divided p-0 sm:p-0">
							{#each systemCategories as cat (cat.id)}
								{@render categoryRow(cat)}
							{/each}
						</div>
					</section>
				{/if}
			</div>
		{/if}
	</PageLoadGate>
</div>

{#if subDialog}
	<SubcategoryFormDialog
		open={true}
		mode={subDialog.mode}
		categoryType={tab}
		initialName={subDialog.name}
		initialIcon={subDialog.icon}
		saving={subDialogSaving}
		onsave={saveSubDialog}
		onclose={closeSubDialog}
	/>
{/if}

<style>
	.sub-list {
		margin: 0 0.75rem 0.75rem 2.25rem;
		border-left: 2px solid var(--border);
		padding-left: 0.25rem;
	}

	.sub-list > :global([data-drag-kind='sub'] + [data-drag-kind='sub']) {
		border-top: 1px solid color-mix(in srgb, var(--border) 60%, transparent);
	}

	.add-sub-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		border: 1px dashed var(--border);
		border-radius: 0.75rem;
		padding: 0.375rem 0.75rem;
		font-size: 0.875rem;
		color: var(--text-muted);
		transition:
			color 120ms,
			border-color 120ms;
	}

	.add-sub-btn:hover {
		color: var(--primary);
		border-color: var(--primary);
	}

	@media (min-width: 640px) {
		.sub-list {
			margin-left: 3.25rem;
		}
	}
</style>
