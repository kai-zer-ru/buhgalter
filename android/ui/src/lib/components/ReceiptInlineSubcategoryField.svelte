<script lang="ts">
	import { _ } from 'svelte-i18n';
	import Select from '$lib/components/Select.svelte';
	import SubcategoryFormDialog from '$lib/components/SubcategoryFormDialog.svelte';
	import { createSubcategory, type Subcategory } from '$lib/api/client';
	import { subcategorySelectOptions } from '$lib/select-options';
	import { toast } from '$lib/toast';

	type Props = {
		id: string;
		categoryId: string;
		value?: string;
		subcategories: Subcategory[];
		onSubcategoriesChange: (subs: Subcategory[]) => void;
		onValueChange?: (subcategoryId: string) => void;
	};

	let {
		id,
		categoryId,
		value = $bindable(''),
		subcategories,
		onSubcategoriesChange,
		onValueChange
	}: Props = $props();

	let dialogOpen = $state(false);
	let dialogSaving = $state(false);

	const options = $derived(subcategorySelectOptions(subcategories));

	async function handleCreate(payload: { name: string; icon: string }) {
		if (!categoryId || dialogSaving) return;
		dialogSaving = true;
		try {
			const sub = await createSubcategory(categoryId, payload);
			const next = [...subcategories, sub];
			onSubcategoriesChange(next);
			value = sub.id;
			onValueChange?.(sub.id);
			dialogOpen = false;
			toast.success($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			dialogSaving = false;
		}
	}
</script>

<div class="flex min-w-[10rem] items-end gap-1">
	<div class="min-w-0 flex-1">
		<Select
			{id}
			placeholder={$_('receipt.selectSubcategory')}
			bind:value
			{options}
			disabled={!categoryId}
			onchange={() => onValueChange?.(value)}
		/>
	</div>
	<button
		type="button"
		class="btn-ghost min-h-11 shrink-0 px-3"
		disabled={!categoryId || dialogSaving}
		title={$_('receipt.addSubcategory')}
		aria-label={$_('receipt.addSubcategory')}
		onclick={() => (dialogOpen = true)}
	>
		+
	</button>
</div>

<SubcategoryFormDialog
	bind:open={dialogOpen}
	categoryType="expense"
	saving={dialogSaving}
	onsave={handleCreate}
	onclose={() => (dialogOpen = false)}
/>
