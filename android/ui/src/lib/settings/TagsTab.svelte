<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { createTag, deleteTag, listTags, updateTag, type Tag } from '$lib/api/client';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import { confirm } from '$lib/confirm';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';

	let tags = $state<Tag[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let newName = $state('');
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let listQuery = $state('');

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
			tags = await listTags();
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
			await createTag(name);
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

<div class="space-y-6">
	<div class="card space-y-3">
		<h2 class="font-medium">{$_('tags.add')}</h2>
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
	</div>

	{#if tags.length > 0}
		<input
			class="input w-full"
			type="search"
			placeholder={$_('common.searchPlaceholder')}
			bind:value={listQuery}
		/>
	{/if}

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if tags.length === 0}
			<EmptyStateCard message={$_('tags.empty')} />
		{:else if filteredTags.length === 0}
			<EmptyStateCard message={$_('common.notFound')} />
		{:else}
			<div class="space-y-2">
				{#each filteredTags as t (t.id)}
					<div class="card">
						{#if editingId === t.id}
							<div class="flex flex-col gap-3 sm:flex-row sm:items-center">
								<input class="input min-w-[12rem] flex-1" bind:value={editName} />
								<button type="button" class="btn-primary" onclick={() => void saveTag()}
									>{$_('common.save')}</button
								>
								<button type="button" class="btn-ghost" onclick={cancelEdit}
									>{$_('common.cancel')}</button
								>
							</div>
						{:else}
							<div class="flex items-center gap-2">
								<span class="min-w-0 flex-1 truncate font-medium">{t.name}</span>
								<RowActionsMenu actions={rowActions(t)} />
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</PageLoadGate>
</div>
