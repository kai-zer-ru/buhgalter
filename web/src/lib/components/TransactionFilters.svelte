<script lang="ts">
	import { _ } from 'svelte-i18n';
	import type { Account, Category } from '$lib/api/client';
	import { categorySelectLabel } from '$lib/category-label';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import Select from '$lib/components/Select.svelte';

	type Props = {
		fromLocal: string;
		toLocal: string;
		type: string;
		categoryId: string;
		accountId: string;
		kind: string;
		search: string;
		accounts?: Account[];
		categories?: Category[];
		showAccount?: boolean;
		showCategory?: boolean;
		showType?: boolean;
		showKind?: boolean;
		expandSearchToEnd?: boolean;
		onreset: () => void;
	};

	let {
		fromLocal = $bindable(),
		toLocal = $bindable(),
		type = $bindable(),
		categoryId = $bindable(),
		accountId = $bindable(),
		kind = $bindable(),
		search = $bindable(),
		accounts = [],
		categories = [],
		showAccount = true,
		showCategory = true,
		showType = true,
		showKind = true,
		expandSearchToEnd = false,
		onreset
	}: Props = $props();

	const typeOptions = $derived([
		{ value: '', label: $_('transactions.filter.all') },
		{ value: 'expense', label: $_('transactions.type.expense') },
		{ value: 'income', label: $_('transactions.type.income') }
	]);

	const accountOptions = $derived([
		{ value: '', label: $_('import.export.all_accounts') },
		...accounts.map((acc) => ({ value: acc.id, label: acc.name }))
	]);

	const filteredCategories = $derived(
		type === 'income' || type === 'expense'
			? categories.filter((cat) => cat.type === type)
			: categories
	);

	const categoryOptions = $derived([
		{ value: '', label: $_('import.export.all_categories') },
		...filteredCategories.map((cat) => ({
			value: cat.id,
			label: categorySelectLabel(cat, categories)
		}))
	]);

	$effect(() => {
		if (type !== 'income' && type !== 'expense') return;
		if (!categoryId) return;
		const selected = categories.find((cat) => cat.id === categoryId);
		if (selected && selected.type !== type) {
			categoryId = '';
		}
	});

	const kindOptions = $derived([
		{ value: '', label: $_('transactions.filter.all') },
		{ value: 'manual', label: $_('transactions.filters.actual') },
		{ value: 'future', label: $_('transactions.filter.planned') }
	]);
</script>

<details class="filter-panel card" open>
	<summary class="md:hidden">{$_('transactions.filters.toggle')}</summary>
	<div class="grid items-end gap-3 sm:grid-cols-2 lg:grid-cols-4 md:mt-0 mt-3">
		<DateTimePicker
			id="tx-filter-from"
			label={$_('transactions.filters.from')}
			bind:value={fromLocal}
			timeMode="hidden"
			usePortal
		/>
		<DateTimePicker
			id="tx-filter-to"
			label={$_('transactions.filters.to')}
			bind:value={toLocal}
			timeMode="hidden"
			usePortal
		/>

		{#if showType}
			<Select
				id="tx-filter-type"
				label={$_('transactions.filters.type')}
				bind:value={type}
				options={typeOptions}
				usePortal
			/>
		{/if}

		{#if showAccount}
			<Select
				id="tx-filter-account"
				label={$_('transactions.filters.account')}
				bind:value={accountId}
				options={accountOptions}
				usePortal
			/>
		{/if}

		{#if showCategory}
			<Select
				id="tx-filter-category"
				label={$_('transactions.filters.category')}
				bind:value={categoryId}
				options={categoryOptions}
				usePortal
			/>
		{/if}

		{#if showKind}
			<Select
				id="tx-filter-kind"
				label={$_('transactions.filters.kind')}
				bind:value={kind}
				options={kindOptions}
				usePortal
			/>
		{/if}

		<label
			class={`space-y-1 sm:col-span-2 ${expandSearchToEnd ? 'lg:col-span-3' : 'lg:col-span-2'}`}
		>
			<span class="text-xs" style:color="var(--text-muted)"
				>{$_('transactions.filters.search')}</span
			>
			<input class="input w-full min-h-11" bind:value={search} />
		</label>

		<div class="flex items-end gap-2 sm:col-span-2 lg:col-span-2">
			<button type="button" class="btn-ghost min-h-11" onclick={onreset}
				>{$_('transactions.filters.reset')}</button
			>
		</div>
	</div>
</details>
