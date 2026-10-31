<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { createTag, deleteTag, listTags, updateTag, type Tag } from '$lib/api/client';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import { confirm } from '$lib/confirm';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import CatalogScopeTabs from '$lib/components/CatalogScopeTabs.svelte';
	import { householdMe, inHousehold } from '$lib/household/membership';

	let tags = $state<Tag[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let newName = $state('');
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let listQuery = $state('');
	let catalogScope = $state<'personal' | 'household'>('personal');

	const filteredTags = $derived.by(() => {
		const q = listQuery.trim().toLowerCase();
		if (!q) return tags;
		return tags.filter((t) => t.name.toLowerCase().includes(q));
	});

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		try {
			tags = await listTags(undefined, inHousehold($householdMe) ? catalogScope : undefined);
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: tags.length > 0 });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	async function addTag() {
		const name = newName.trim();
		if (!name) return;
		try {
			await createTag(name, inHousehold($householdMe) ? catalogScope : undefined);
			newName = '';
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	function startEdit(t: Tag) {
		editingId = t.id;
		editName = t.name;
	}

	function cancelEdit() {
		editingId = null;
	}

	async function saveTag() {
		if (!editingId) return;
		try {
			await updateTag(editingId, editName);
			editingId = null;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function removeTag(t: Tag) {
		const ok = await confirm({
			message: $_('tags.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteTag(t.id);
			toast($_('common.deleted'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	function rowActions(t: Tag): RowAction[] {
		return [
			{
				icon: 'edit',
				label: $_('common.edit'),
				onclick: () => startEdit(t)
			},
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void removeTag(t)
			}
		];
	}
</script>

<div class="space-y-8">
	<CatalogScopeTabs
		mode="catalog"
		value={catalogScope}
		onchange={(next) => {
			catalogScope = next === 'household' ? 'household' : 'personal';
			void load();
		}}
	/>
	<section class="card space-y-3">
		<div class="space-y-1">
			<h2 class="font-semibold">{$_('tags.add')}</h2>
			<p class="text-sm" style:color="var(--text-muted)">{$_('tags.add_hint')}</p>
		</div>
		<form
			class="flex flex-col gap-3 sm:flex-row sm:items-center"
			onsubmit={(e) => {
				e.preventDefault();
				void addTag();
			}}
		>
			<input
				class="input min-w-[12rem] flex-1"
				placeholder={$_('tags.field.name')}
				bind:value={newName}
			/>
			<button type="submit" class="btn-primary sm:px-4">{$_('common.create')}</button>
		</form>
	</section>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if tags.length === 0}
			<EmptyStateCard message={$_('tags.empty')} />
		{:else}
			<section class="space-y-3">
				<GroupHeading
					title={$_('tags.group.all', { values: { count: tags.length } })}
					hint={$_('tags.group.all_hint')}
				/>
				<div class="card list-divided p-0 sm:p-0">
					<div class="p-2 sm:p-3">
						<input
							class="input w-full"
							type="search"
							placeholder={$_('common.searchPlaceholder')}
							bind:value={listQuery}
						/>
					</div>
					{#if filteredTags.length === 0}
						<p class="px-4 py-6 text-center text-sm" style:color="var(--text-muted)">
							{$_('common.notFound')}
						</p>
					{/if}
					{#each filteredTags as t (t.id)}
						<div data-testid="tag-row">
							{#if editingId === t.id}
								<form
									class="flex flex-col gap-3 p-3 sm:flex-row sm:items-center sm:p-4"
									onsubmit={(e) => {
										e.preventDefault();
										void saveTag();
									}}
								>
									<span class="tag-glyph hidden sm:inline-flex" aria-hidden="true">#</span>
									<input
										class="input min-w-[12rem] flex-1"
										bind:value={editName}
										onkeydown={(e) => {
											if (e.key === 'Escape') cancelEdit();
										}}
									/>
									<div class="flex justify-end gap-2">
										<button type="button" class="btn-ghost" onclick={cancelEdit}
											>{$_('common.cancel')}</button
										>
										<button type="submit" class="btn-primary">{$_('common.save')}</button>
									</div>
								</form>
							{:else}
								<div class="flex items-center gap-3 py-1.5 pr-1.5 pl-3 sm:pr-2 sm:pl-4">
									<span class="tag-glyph" aria-hidden="true">#</span>
									<span class="min-w-0 flex-1 truncate font-medium">{t.name}</span>
									<RowActionsMenu actions={rowActions(t)} />
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</section>
		{/if}
	</PageLoadGate>
</div>

<style>
	.tag-glyph {
		display: inline-flex;
		height: 2.25rem;
		width: 2.25rem;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		border-radius: 0.75rem;
		font-weight: 600;
		color: var(--primary);
		background-color: color-mix(in srgb, var(--primary) 12%, transparent);
	}
</style>
