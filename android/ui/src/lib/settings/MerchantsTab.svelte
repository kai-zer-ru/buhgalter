<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		createMerchant,
		deleteMerchant,
		listMerchants,
		updateMerchant,
		type Merchant
	} from '$lib/api/client';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import CategoryIconPicker from '$lib/components/CategoryIconPicker.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import { defaultIconForKind } from '$lib/category-icons';
	import { confirm } from '$lib/confirm';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';

	const iconSize = 40;

	let merchants = $state<Merchant[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let newName = $state('');
	let newIcon = $state(defaultIconForKind('expense'));

	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editIcon = $state('default');
	let listQuery = $state('');

	const filteredMerchants = $derived.by(() => {
		const q = listQuery.trim().toLowerCase();
		if (!q) return merchants;
		return merchants.filter((m) => m.name.toLowerCase().includes(q));
	});

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		try {
			merchants = await listMerchants();
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: merchants.length > 0 });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	async function addMerchant() {
		const name = newName.trim();
		if (!name) return;
		try {
			await createMerchant(name, newIcon);
			newName = '';
			newIcon = defaultIconForKind('expense');
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	function startEdit(m: Merchant) {
		editingId = m.id;
		editName = m.name;
		editIcon = m.icon || 'default';
	}

	async function saveEdit() {
		if (!editingId) return;
		const name = editName.trim();
		const icon = editIcon.trim() || 'default';
		if (!name) return;
		try {
			await updateMerchant(editingId, name, icon);
			editingId = null;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function removeMerchant(m: Merchant) {
		const ok = await confirm({
			message: $_('merchants.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteMerchant(m.id);
			toast($_('common.deleted'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	function rowActions(m: Merchant): RowAction[] {
		return [
			{
				icon: 'edit',
				label: $_('common.edit'),
				onclick: () => startEdit(m)
			},
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void removeMerchant(m)
			}
		];
	}
</script>

<div class="space-y-6">
	<div class="card space-y-3">
		<h2 class="font-medium">{$_('merchants.add')}</h2>
		<div class="flex items-center gap-2">
			<CategoryIconPicker
				bind:value={newIcon}
				bind:categoryName={newName}
				categoryType="expense"
				variant="button"
				quickSize={iconSize}
				{iconSize}
			/>
			<input
				class="input min-w-0 flex-1"
				placeholder={$_('merchants.field.name')}
				bind:value={newName}
			/>
			<button type="button" class="btn-primary shrink-0 sm:px-4" onclick={() => void addMerchant()}>
				{$_('common.create')}
			</button>
		</div>
	</div>

	{#if merchants.length > 0}
		<input
			class="input w-full"
			type="search"
			placeholder={$_('common.searchPlaceholder')}
			bind:value={listQuery}
		/>
	{/if}

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if merchants.length === 0}
			<EmptyStateCard message={$_('merchants.empty')} />
		{:else if filteredMerchants.length === 0}
			<EmptyStateCard message={$_('common.notFound')} />
		{:else}
			<div class="space-y-2">
				{#each filteredMerchants as m (m.id)}
					<div class="card">
						<div class="flex flex-wrap items-center gap-1 sm:flex-nowrap">
							{#if editingId !== m.id}
								<span class="inline-flex shrink-0 items-center justify-center min-h-11 min-w-11">
									<CategoryIcon icon={m.icon || 'default'} size={iconSize} />
								</span>
								<span class="min-w-0 flex-1 truncate font-medium">{m.name}</span>
								<RowActionsMenu actions={rowActions(m)} />
							{:else}
								{#key editingId}
									<CategoryIconPicker
										bind:value={editIcon}
										bind:categoryName={editName}
										categoryType="expense"
										lockName={true}
										variant="button"
										quickSize={iconSize}
										{iconSize}
									/>
								{/key}
								<input class="input min-w-0 flex-1" bind:value={editName} />
								<IconButton
									icon="save"
									label={$_('common.save')}
									variant="primary"
									onclick={() => void saveEdit()}
								/>
								<IconButton
									icon="cancel"
									label={$_('common.cancel')}
									onclick={() => (editingId = null)}
								/>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</PageLoadGate>
</div>
