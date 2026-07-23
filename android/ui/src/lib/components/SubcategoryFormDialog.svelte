<script lang="ts">
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import CategoryIconPicker from '$lib/components/CategoryIconPicker.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import type { CategoryKind } from '$lib/category-icons';

	let {
		open = $bindable(false),
		mode = 'create',
		categoryType = 'expense',
		initialName = '',
		initialIcon = 'default',
		saving = false,
		onsave,
		onclose
	}: {
		open?: boolean;
		mode?: 'create' | 'edit';
		categoryType?: CategoryKind;
		initialName?: string;
		initialIcon?: string;
		saving?: boolean;
		onsave: (payload: { name: string; icon: string }) => void | Promise<void>;
		onclose: () => void;
	} = $props();

	let name = $state('');
	let icon = $state('default');
	let seededForOpen = $state(false);

	const title = $derived.by(() => {
		void $locale;
		return mode === 'edit' ? tr('categories.sub.editTitle') : tr('categories.sub.createTitle');
	});

	$effect(() => {
		if (!open) {
			seededForOpen = false;
			return;
		}
		if (seededForOpen) return;
		name = initialName;
		icon = initialIcon;
		seededForOpen = true;
	});

	function close() {
		onclose();
	}

	async function save() {
		const trimmed = name.trim();
		if (!trimmed || saving) return;
		await onsave({ name: trimmed, icon });
	}
</script>

<ModalShell bind:open {title} maxWidth="max-w-md" onclose={close}>
	<div class="space-y-4">
		<label class="block space-y-1.5">
			<span class="text-sm font-medium">{$_('categories.sub.fieldName')}</span>
			<input
				class="input w-full"
				placeholder={$_('categories.sub.fieldName')}
				bind:value={name}
				disabled={saving}
				onkeydown={(e) => {
					if (e.key === 'Enter') {
						e.preventDefault();
						void save();
					}
				}}
			/>
		</label>
		<div class="space-y-1.5">
			<span class="text-sm font-medium">{$_('categories.icons.title')}</span>
			<div class="flex items-center gap-3">
				<CategoryIconPicker
					bind:value={icon}
					bind:categoryName={name}
					{categoryType}
					lockName={mode === 'edit'}
					variant="button"
				/>
				<span class="text-sm" style:color="var(--text-muted)">{$_('categories.icons.change')}</span>
			</div>
		</div>
	</div>
	{#snippet footer()}
		<button type="button" class="btn-ghost" disabled={saving} onclick={close}>
			{$_('common.cancel')}
		</button>
		<button
			type="button"
			class="btn-primary"
			disabled={saving || !name.trim()}
			onclick={() => void save()}
		>
			{saving ? $_('common.loading') : $_('common.save')}
		</button>
	{/snippet}
</ModalShell>
