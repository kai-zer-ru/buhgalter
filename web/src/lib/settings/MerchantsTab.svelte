<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		createMerchant,
		deleteMerchant,
		listMerchants,
		mergeMerchants,
		updateMerchant,
		type Merchant
	} from '$lib/api/client';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import CategoryIconPicker from '$lib/components/CategoryIconPicker.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import { defaultIconForKind } from '$lib/category-icons';
	import { confirm } from '$lib/confirm';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { writeMerchantsRefSnapshot } from '$lib/ref-cache';
	import { toast } from '$lib/toast';

	const iconSize = 40;

	let merchants = $state<Merchant[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let newName = $state('');
	let newAliases = $state('');
	let newIcon = $state(defaultIconForKind('expense'));

	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editAliases = $state('');
	let editIcon = $state('default');
	let listQuery = $state('');

	let mergeTarget = $state<Merchant | null>(null);
	let mergeSelected = $state<string[]>([]);

	const filteredMerchants = $derived.by(() => {
		const q = listQuery.trim().toLowerCase();
		if (!q) return merchants;
		return merchants.filter(
			(m) =>
				m.name.toLowerCase().includes(q) ||
				(m.aliases ?? []).some((alias) => alias.toLowerCase().includes(q))
		);
	});

	const mergeOthers = $derived.by(() => {
		const target = mergeTarget;
		if (!target) return [];
		return merchants.filter((m) => m.id !== target.id);
	});

	onMount(() => {
		void load();
	});

	function parseAliases(raw: string): string[] {
		const out: string[] = [];
		for (const part of raw.split(',')) {
			const name = part.trim();
			if (!name) continue;
			if (out.some((item) => item.toLocaleLowerCase() === name.toLocaleLowerCase())) continue;
			out.push(name);
		}
		return out;
	}

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
			await createMerchant(name, newIcon, parseAliases(newAliases));
			newName = '';
			newAliases = '';
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
		editAliases = (m.aliases ?? []).join(', ');
		editIcon = m.icon || 'default';
	}

	async function saveEdit() {
		if (!editingId) return;
		const name = editName.trim();
		const icon = editIcon.trim() || 'default';
		if (!name) return;
		try {
			await updateMerchant(editingId, name, icon, parseAliases(editAliases));
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
			if (editingId === m.id) editingId = null;
			if (mergeTarget?.id === m.id) {
				mergeTarget = null;
				mergeSelected = [];
			} else {
				mergeSelected = mergeSelected.filter((id) => id !== m.id);
			}
			const remaining = merchants.filter((row) => row.id !== m.id);
			merchants = remaining;
			writeMerchantsRefSnapshot(remaining);
			toast($_('common.deleted'));
		} catch (err) {
			toast.fromError(err);
		}
	}

	function startMerge(m: Merchant) {
		mergeTarget = m;
		mergeSelected = [];
	}

	function toggleMerge(id: string, checked: boolean) {
		if (checked) {
			if (!mergeSelected.includes(id)) mergeSelected = [...mergeSelected, id];
			return;
		}
		mergeSelected = mergeSelected.filter((item) => item !== id);
	}

	async function submitMerge() {
		const target = mergeTarget;
		if (!target || mergeSelected.length === 0) return;
		const ok = await confirm({
			message: $_('merchants.confirm.merge', { values: { name: target.name } }),
			confirmLabel: $_('merchants.action.merge')
		});
		if (!ok) return;
		try {
			await mergeMerchants(target.id, mergeSelected);
			mergeTarget = null;
			mergeSelected = [];
			toast($_('common.saved'));
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
				icon: 'transfer',
				label: $_('merchants.action.merge'),
				onclick: () => startMerge(m)
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
		<input
			id="merchant-new-aliases"
			class="input w-full"
			placeholder={$_('merchants.field.aliasesPlaceholder')}
			aria-label={$_('merchants.field.aliases')}
			bind:value={newAliases}
		/>
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
					<div class="card space-y-2">
						<div class="flex flex-wrap items-center gap-1 sm:flex-nowrap">
							{#if editingId !== m.id}
								<span class="inline-flex shrink-0 items-center justify-center min-h-11 min-w-11">
									<CategoryIcon icon={m.icon || 'default'} size={iconSize} />
								</span>
								<div class="min-w-0 flex-1">
									<div class="truncate font-medium">{m.name}</div>
									{#if m.aliases?.length}
										<div class="truncate text-sm" style:color="var(--text-muted)">
											{m.aliases.join(', ')}
										</div>
									{/if}
								</div>
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
						{#if editingId === m.id}
							<input
								id="merchant-edit-aliases"
								class="input w-full"
								placeholder={$_('merchants.field.aliasesPlaceholder')}
								aria-label={$_('merchants.field.aliases')}
								bind:value={editAliases}
							/>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</PageLoadGate>
</div>

<ModalShell
	open={mergeTarget !== null}
	title={$_('merchants.merge.title')}
	onclose={() => (mergeTarget = null)}
>
	<p class="text-sm" style:color="var(--text-muted)">
		{$_('merchants.merge.hint', { values: { name: mergeTarget?.name ?? '' } })}
	</p>
	{#if mergeOthers.length === 0}
		<p class="mt-3 text-sm">{$_('merchants.merge.empty')}</p>
	{:else}
		<ul class="mt-3 space-y-2">
			{#each mergeOthers as m (m.id)}
				<li>
					<label class="flex items-center gap-2">
						<input
							type="checkbox"
							checked={mergeSelected.includes(m.id)}
							onchange={(event) =>
								toggleMerge(m.id, (event.currentTarget as HTMLInputElement).checked)}
						/>
						<span>{m.name}</span>
					</label>
				</li>
			{/each}
		</ul>
	{/if}
	{#snippet footer()}
		<button type="button" class="btn-secondary" onclick={() => (mergeTarget = null)}>
			{$_('common.cancel')}
		</button>
		<button
			type="button"
			class="btn-primary"
			disabled={mergeSelected.length === 0}
			onclick={() => void submitMerge()}
		>
			{$_('merchants.action.merge')}
		</button>
	{/snippet}
</ModalShell>
