<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { get } from 'svelte/store';
	import { replaceState } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		createCategory,
		createSubcategory,
		deleteCategory,
		deleteSubcategory,
		listCategories,
		listSubcategories,
		reorderCategories,
		reorderSubcategories,
		setPrimaryCategory,
		updateCategory,
		updateSubcategory,
		type Category,
		type Subcategory
	} from '$lib/api/client';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import CategoryIconPicker from '$lib/components/CategoryIconPicker.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import ReorderDragGhost from '$lib/components/ReorderDragGhost.svelte';
	import { defaultIconForKind } from '$lib/category-icons';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import { beginPointerDrag, moveId, type DragGhostView } from '$lib/drag-reorder';

	type Tab = 'expense' | 'income';
	const categoryIconSize = 40;

	function tabFromSearchParams(params: URLSearchParams): Tab {
		return params.get('type') === 'income' ? 'income' : 'expense';
	}

	function subInputId(categoryId: string) {
		return `sub-input-${categoryId}`;
	}

	function defaultSubIconFor(cat: Category): string {
		return cat.icon || defaultIconForKind(tab);
	}

	let tab = $state<Tab>(tabFromSearchParams(get(page).url.searchParams));
	let categories = $state<Category[]>([]);
	let subs = $state<Record<string, Subcategory[]>>({});
	let expanded = $state<Record<string, boolean>>({});
	let loading = $state(true);

	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editIcon = $state('default');

	let newName = $state('');
	let newIcon = $state(defaultIconForKind('expense'));

	let newSubName = $state<Record<string, string>>({});
	let newSubIcon = $state<Record<string, string>>({});
	let editingSubId = $state<string | null>(null);
	let editSubName = $state('');
	let editSubIcon = $state('default');

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
				() => editingSubId !== null,
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

	async function load() {
		loading = true;
		try {
			categories = await listCategories(tab);
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	function selectTab(next: Tab) {
		tab = next;
		newIcon = defaultIconForKind(next);
		expanded = {};
		const url = new URL(get(page).url);
		url.searchParams.set('tab', 'categories');
		if (next === 'expense') {
			url.searchParams.delete('type');
		} else {
			url.searchParams.set('type', next);
		}
		const search = url.searchParams.toString();
		const categoriesUrl = search ? `${resolve('/settings')}?${search}` : resolve('/settings');
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path
		replaceState(categoriesUrl, {});
		void load();
	}

	async function toggleExpand(cat: Category) {
		const opening = !expanded[cat.id];
		expanded = { ...expanded, [cat.id]: opening };
		if (!opening) return;

		newSubName = { ...newSubName, [cat.id]: newSubName[cat.id] ?? '' };
		newSubIcon = { ...newSubIcon, [cat.id]: newSubIcon[cat.id] ?? defaultSubIconFor(cat) };

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
			await createCategory({ name: newName.trim(), type: tab, icon: newIcon });
			newName = '';
			newIcon = defaultIconForKind(tab);
			toast($_('common.saved'));
			await load();
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
			await updateCategory(editingId, { name: editName, icon: editIcon });
			editingId = null;
			toast($_('common.saved'));
			await load();
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
			await deleteCategory(id);
			toast($_('common.deleted'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function addSub(categoryId: string) {
		const name = (newSubName[categoryId] ?? '').trim();
		if (!name) return;
		const parent = categories.find((c) => c.id === categoryId);
		const icon =
			newSubIcon[categoryId] ?? (parent ? defaultSubIconFor(parent) : defaultIconForKind(tab));
		try {
			const sub = await createSubcategory(categoryId, { name, icon });
			subs = { ...subs, [categoryId]: [...(subs[categoryId] ?? []), sub] };
			newSubName = { ...newSubName, [categoryId]: '' };
			newSubIcon = {
				...newSubIcon,
				[categoryId]: parent ? defaultSubIconFor(parent) : defaultIconForKind(tab)
			};
			toast($_('common.saved'));
			await tick();
			document.getElementById(subInputId(categoryId))?.focus();
		} catch (err) {
			toast.fromError(err);
		}
	}

	function startEditSub(sub: Subcategory) {
		editingSubId = sub.id;
		editSubName = sub.name;
		editSubIcon = sub.icon || 'default';
	}

	async function saveSubEdit(categoryId: string) {
		if (!editingSubId) return;
		try {
			const updated = await updateSubcategory(editingSubId, {
				name: editSubName,
				icon: editSubIcon
			});
			subs = {
				...subs,
				[categoryId]: (subs[categoryId] ?? []).map((s) => (s.id === updated.id ? updated : s))
			};
			editingSubId = null;
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function removeSub(categoryId: string, subId: string) {
		const ok = await confirm({
			message: $_('categories.confirm.deleteSub'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteSubcategory(subId);
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
		const list = subs[categoryId] ?? [];
		const ids = moveId(
			list.map((s) => s.id),
			fromId,
			toId
		);
		if (!ids) return;
		try {
			subs = { ...subs, [categoryId]: await reorderSubcategories(categoryId, ids) };
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function makePrimary(id: string) {
		if (categories.find((c) => c.id === id)?.is_primary) return;
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

<div class="space-y-6">
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

	<div class="card space-y-3">
		<h2 class="font-medium">{$_('categories.add')}</h2>
		<div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
			<input
				class="input min-w-[12rem] flex-1"
				placeholder={$_('categories.field.name')}
				bind:value={newName}
			/>
			<CategoryIconPicker bind:value={newIcon} bind:categoryName={newName} categoryType={tab} />
			<button
				type="button"
				class="btn-primary btn-icon sm:min-w-[auto] sm:px-4"
				onclick={addCategory}
			>
				<span class="sr-only">{$_('common.create')}</span>
				<svg
					aria-hidden="true"
					class="h-5 w-5 sm:hidden"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
				>
					<path d="M12 5v14M5 12h14" />
				</svg>
				<span class="hidden sm:inline">{$_('common.create')}</span>
			</button>
		</div>
	</div>

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else}
		<div class="space-y-2">
			{#each categories as cat (cat.id)}
				<div
					class="card transition-opacity"
					class:opacity-30={draggingId === cat.id}
					class:border-t-2={overId === cat.id && draggingId !== null && draggingId !== cat.id}
					data-drag-id={cat.id}
					data-drag-kind="category"
					style:border-color={overId === cat.id ? 'var(--primary)' : undefined}
				>
					<div class="flex flex-wrap items-center gap-1 sm:flex-nowrap" data-drag-row>
						{#if editingId !== cat.id}
							{#if !cat.is_system}
								<span
									class="btn-icon btn-ghost cursor-grab touch-none text-lg leading-none select-none active:cursor-grabbing"
									role="button"
									tabindex="-1"
									aria-label={$_('categories.drag.handle')}
									onpointerdown={(e) =>
										startCategoryDrag(
											e,
											cat,
											e.currentTarget.closest('[data-drag-id]') as HTMLElement
										)}
								>
									⠿
								</span>
							{:else}
								<span class="btn-icon shrink-0" aria-hidden="true"></span>
							{/if}
							<span class="inline-flex shrink-0 items-center justify-center min-h-11 min-w-11">
								<CategoryIcon icon={cat.icon} size={categoryIconSize} />
							</span>
							{#if !cat.is_system}
								<button
									type="button"
									class="min-w-0 flex-1 truncate text-left font-medium inline-flex items-center gap-1.5"
									aria-expanded={expanded[cat.id] ?? false}
									onclick={() => toggleExpand(cat)}
								>
									<span class="inline-flex min-w-0 items-center gap-1 truncate">
										<span class="truncate">{cat.name}</span>
										{#if cat.is_primary}
											<span
												class="shrink-0"
												style:color="var(--primary)"
												title={$_('categories.primary.badge')}
												aria-label={$_('categories.primary.badge')}
											>
												<svg
													aria-hidden="true"
													class="h-4 w-4"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
												>
													<path d="M20 6 9 17l-5-5" />
												</svg>
											</span>
										{/if}
									</span>
									<span class="shrink-0 text-xs leading-none" aria-hidden="true">
										{expanded[cat.id] ? '▼' : '▶'}
									</span>
								</button>
							{:else}
								<span class="min-w-0 flex-1 truncate font-medium">
									{cat.name}
									<span class="ml-1 text-xs" style:color="var(--text-muted)"
										>({$_('categories.system.badge')})</span
									>
								</span>
							{/if}
							{#if !cat.is_system}
								<RowActionsMenu actions={categoryActions(cat)} />
							{/if}
						{:else}
							<span class="inline-flex shrink-0 items-center justify-center min-h-11 min-w-11">
								<CategoryIcon icon={cat.icon} size={categoryIconSize} />
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
								<div class="flex flex-wrap gap-2">
									<IconButton
										icon="save"
										label={$_('common.save')}
										variant="primary"
										onclick={saveEdit}
									/>
									<IconButton
										icon="cancel"
										label={$_('common.cancel')}
										onclick={() => (editingId = null)}
									/>
								</div>
							</div>
						{/if}
					</div>

					{#if expanded[cat.id] && !cat.is_system}
						<div
							class="mt-3 space-y-3 border-t pt-3 pl-4 sm:pl-10"
							style:border-color="var(--border)"
						>
							{#each subs[cat.id] ?? [] as sub (sub.id)}
								<div
									class="flex flex-wrap items-center gap-2 rounded-lg transition-opacity"
									class:opacity-30={draggingId === sub.id}
									class:border-t-2={overId === sub.id &&
										draggingId !== null &&
										draggingId !== sub.id}
									data-drag-id={sub.id}
									data-drag-kind="sub"
									style:border-color={overId === sub.id ? 'var(--primary)' : undefined}
								>
									{#if editingSubId === sub.id}
										<div
											class="flex min-w-0 w-full flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center"
										>
											<CategoryIconPicker
												bind:value={editSubIcon}
												bind:categoryName={editSubName}
												categoryType={tab}
												lockName={true}
												quickSize={categoryIconSize}
												iconSize={categoryIconSize}
											/>
											<input
												class="input min-w-[10rem] flex-1"
												bind:value={editSubName}
												onkeydown={(e) => {
													if (e.key === 'Enter') {
														e.preventDefault();
														void saveSubEdit(cat.id);
													}
												}}
											/>
											<div class="flex shrink-0 gap-1">
												<IconButton
													icon="save"
													label={$_('common.save')}
													variant="primary"
													onclick={() => saveSubEdit(cat.id)}
												/>
												<IconButton
													icon="cancel"
													label={$_('common.cancel')}
													onclick={() => (editingSubId = null)}
												/>
											</div>
										</div>
									{:else}
										<div
											class="flex min-w-0 flex-1 items-center gap-1 overflow-hidden"
											data-drag-row
										>
											<span
												class="btn-icon btn-ghost cursor-grab touch-none text-base leading-none select-none active:cursor-grabbing"
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
											<span
												class="inline-flex shrink-0 items-center justify-center min-h-11 min-w-11"
											>
												<CategoryIcon icon={sub.icon || 'default'} size={categoryIconSize} />
											</span>
											<span class="min-w-0 flex-1 truncate">{sub.name}</span>
											<RowActionsMenu
												actions={[
													{
														icon: 'edit',
														label: $_('accounts.action.edit'),
														onclick: () => startEditSub(sub)
													},
													{
														icon: 'delete',
														label: $_('common.delete'),
														variant: 'danger',
														onclick: () => removeSub(cat.id, sub.id)
													}
												]}
											/>
										</div>
									{/if}
								</div>
							{/each}
							<div class="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
								<CategoryIconPicker
									bind:value={newSubIcon[cat.id]}
									bind:categoryName={newSubName[cat.id]}
									categoryType={tab}
									quickSize={categoryIconSize}
									iconSize={categoryIconSize}
								/>
								<input
									id={subInputId(cat.id)}
									class="input min-w-[10rem] flex-1"
									placeholder={$_('categories.sub.add')}
									bind:value={newSubName[cat.id]}
									onkeydown={(e) => {
										if (e.key === 'Enter') {
											e.preventDefault();
											void addSub(cat.id);
										}
									}}
								/>
								<IconButton
									icon="create"
									label={$_('common.create')}
									onclick={() => addSub(cat.id)}
								/>
							</div>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>
