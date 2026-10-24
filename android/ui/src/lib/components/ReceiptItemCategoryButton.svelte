<script lang="ts">
	import { _ } from 'svelte-i18n';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import ReceiptInlineSubcategoryField from '$lib/components/ReceiptInlineSubcategoryField.svelte';
	import Select from '$lib/components/Select.svelte';
	import type { Subcategory } from '$lib/api/client';
	import type { SelectOption } from '$lib/select-options';

	type Props = {
		buttonId: string;
		categorySelectId: string;
		subcategoryFieldId: string;
		itemName: string;
		categoryId?: string;
		subcategoryId?: string;
		categoryOptions: SelectOption[];
		subcategories: Subcategory[];
		onCategoryChange: (categoryId: string) => void;
		onSubcategoriesChange: (subs: Subcategory[]) => void;
	};

	let {
		buttonId,
		categorySelectId,
		subcategoryFieldId,
		itemName,
		categoryId = $bindable(''),
		subcategoryId = $bindable(''),
		categoryOptions,
		subcategories,
		onCategoryChange,
		onSubcategoriesChange
	}: Props = $props();

	let open = $state(false);

	const categoryName = $derived(
		categoryOptions.find((option) => option.value === categoryId)?.label ?? ''
	);
	const subcategoryName = $derived(
		subcategories.find((sub) => sub.id === subcategoryId)?.name ?? ''
	);
	const buttonLabel = $derived.by(() => {
		if (categoryName && subcategoryName) return `${categoryName} · ${subcategoryName}`;
		if (categoryName) return categoryName;
		return $_('receipt.selectCategory');
	});
	const buttonAriaLabel = $derived(
		itemName.trim() ? `${buttonLabel}. ${itemName.trim()}` : buttonLabel
	);

	function close() {
		open = false;
	}
</script>

<button
	type="button"
	id={buttonId}
	class="btn-ghost min-h-11 w-full min-w-0 whitespace-normal break-words text-left"
	style:color={categoryName ? 'var(--text)' : undefined}
	aria-haspopup="dialog"
	aria-expanded={open}
	aria-label={buttonAriaLabel}
	onclick={() => (open = true)}
>
	{buttonLabel}
</button>

<ModalShell bind:open title={itemName.trim() || '—'} clip={false} onclose={close}>
	<div class="space-y-4">
		<Select
			id={categorySelectId}
			label={$_('transactions.field.category')}
			placeholder={$_('receipt.selectCategory')}
			bind:value={categoryId}
			options={categoryOptions}
			onchange={(next) => onCategoryChange(next)}
		/>
		<div>
			<label class="mb-1 block text-sm font-medium" for={subcategoryFieldId}
				>{$_('transactions.field.subcategory')}</label
			>
			<ReceiptInlineSubcategoryField
				id={subcategoryFieldId}
				{categoryId}
				bind:value={subcategoryId}
				{subcategories}
				{onSubcategoriesChange}
			/>
		</div>
	</div>
	{#snippet footer()}
		<button type="button" class="btn-ghost min-h-11" onclick={close}>
			{$_('common.close')}
		</button>
	{/snippet}
</ModalShell>
