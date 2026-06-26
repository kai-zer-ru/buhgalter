<script lang="ts">
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import {
		defaultCategoryNameForIcon,
		quickIconsDisplay,
		searchCategoryIcons,
		type CategoryKind
	} from '$lib/category-icons';

	let {
		value = $bindable('default'),
		categoryName = $bindable(''),
		categoryType = 'expense',
		lockName = false,
		iconSize = 40,
		quickSize = 40
	}: {
		value?: string;
		categoryName?: string;
		categoryType?: CategoryKind;
		/** When true, icon changes never overwrite the name (edit existing category). */
		lockName?: boolean;
		iconSize?: number;
		quickSize?: number;
	} = $props();

	let open = $state(false);
	let search = $state('');
	let nameLocked = $state(false);
	let lastAutoName = $state('');
	let pinnedQuickIcon = $state<string | null>(null);

	const displayQuick = $derived(
		quickIconsDisplay(categoryType, { pinned: pinnedQuickIcon, selected: value })
	);
	const filtered = $derived(searchCategoryIcons(search, categoryType));

	const modalTitle = $derived.by(() => {
		void $locale;
		return categoryType === 'income'
			? tr('categories.icons.titleIncome')
			: tr('categories.icons.titleExpense');
	});

	function closeModal() {
		open = false;
		search = '';
	}

	$effect(() => {
		void categoryType;
		pinnedQuickIcon = null;
	});

	$effect(() => {
		if (lockName) {
			nameLocked = true;
		}
	});

	$effect(() => {
		const name = categoryName;
		if (!name.trim()) {
			if (!lockName) {
				nameLocked = false;
			}
			lastAutoName = '';
			return;
		}
		if (nameLocked) return;
		if (!lastAutoName) {
			nameLocked = true;
			return;
		}
		if (name !== lastAutoName) {
			nameLocked = true;
		}
	});

	function pickFromQuick(id: string) {
		pinnedQuickIcon = null;
		pickIcon(id);
	}

	function pickIcon(id: string) {
		value = id;
		if (nameLocked) return;
		const trimmed = categoryName.trim();
		if (trimmed && trimmed !== lastAutoName) {
			nameLocked = true;
			return;
		}
		const next = defaultCategoryNameForIcon(id, categoryType);
		categoryName = next;
		lastAutoName = next;
	}

	function select(id: string) {
		pinnedQuickIcon = id;
		pickIcon(id);
		closeModal();
	}
</script>

<div class="flex flex-wrap items-center gap-1.5">
	{#each displayQuick as iconId (iconId)}
		<button
			type="button"
			class="btn-icon btn-ghost"
			style:background-color={value === iconId
				? 'color-mix(in srgb, var(--primary) 15%, transparent)'
				: 'transparent'}
			title={iconId}
			onclick={() => pickFromQuick(iconId)}
		>
			<CategoryIcon icon={iconId} size={quickSize} />
		</button>
	{/each}
	<button
		type="button"
		class="btn-icon btn-ghost"
		style:background-color="color-mix(in srgb, var(--border) 40%, transparent)"
		style:color="var(--text-muted)"
		aria-label={$_('categories.icons.more')}
		onclick={() => (open = true)}
	>
		<svg aria-hidden="true" class="mx-auto h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
			<circle cx="5" cy="12" r="1.75" />
			<circle cx="12" cy="12" r="1.75" />
			<circle cx="19" cy="12" r="1.75" />
		</svg>
	</button>
</div>

<ModalShell bind:open title={modalTitle} maxWidth="max-w-lg" onclose={closeModal}>
	<div class="space-y-3">
		<input
			class="input"
			type="search"
			placeholder={$_('categories.icons.search')}
			bind:value={search}
		/>

		{#if filtered.length === 0}
			<p class="py-8 text-center text-sm" style:color="var(--text-muted)">
				{$_('categories.icons.empty')}
			</p>
		{:else}
			<div class="grid grid-cols-6 gap-1.5 sm:grid-cols-8">
				{#each filtered as icon (icon.id)}
					<button
						type="button"
						class="btn-icon btn-ghost"
						style:background-color={value === icon.id
							? 'color-mix(in srgb, var(--primary) 15%, transparent)'
							: 'transparent'}
						title={icon.tags[0] ?? icon.id}
						onclick={() => select(icon.id)}
					>
						<CategoryIcon icon={icon.id} size={iconSize} />
					</button>
				{/each}
			</div>
		{/if}
	</div>
	{#snippet footer()}
		<div class="mr-auto flex items-center gap-2 text-sm" style:color="var(--text-muted)">
			<span>{$_('categories.icons.selected')}:</span>
			<span class="btn-icon">
				<CategoryIcon icon={value} size={iconSize} />
			</span>
		</div>
		<IconButton icon="save" label={$_('common.save')} variant="primary" onclick={closeModal} />
	{/snippet}
</ModalShell>
