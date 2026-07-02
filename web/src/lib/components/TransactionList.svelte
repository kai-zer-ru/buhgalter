<script lang="ts">
	import type { Snippet } from 'svelte';
	import { _ } from 'svelte-i18n';
	import type { Transaction } from '$lib/api/client';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import TransactionAccountCell from '$lib/components/TransactionAccountCell.svelte';
	import { formatAPIOperationDateTimeForDisplay } from '$lib/dates';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import {
		transactionAmountSign,
		canEditTransaction,
		canRepeatTransaction
	} from '$lib/transaction-display';

	let {
		transactions,
		siblings = [],
		tz,
		emptyMessage,
		showDelete = false,
		showEdit = false,
		showDescription = false,
		showAmountSign = false,
		singleAccount = false,
		ondelete,
		onedit,
		onrepeat,
		onmakeRecurring,
		descriptionExtra
	}: {
		transactions: Transaction[];
		siblings?: Transaction[];
		tz: string;
		emptyMessage: string;
		showDelete?: boolean;
		showEdit?: boolean;
		showDescription?: boolean;
		showAmountSign?: boolean;
		singleAccount?: boolean;
		ondelete?: (tx: Transaction) => void;
		onedit?: (tx: Transaction) => void;
		onrepeat?: (tx: Transaction) => void;
		onmakeRecurring?: (tx: Transaction) => void;
		descriptionExtra?: Snippet<[Transaction]>;
	} = $props();

	const showActions = $derived(
		Boolean((showDelete && ondelete) || (showEdit && onedit) || onrepeat || onmakeRecurring)
	);

	function canMakeRecurring(tx: Transaction): boolean {
		return Boolean(onmakeRecurring && tx.type !== 'transfer' && !tx.category_is_system);
	}

	function rowActions(tx: Transaction): RowAction[] {
		const actions: RowAction[] = [];
		if (onrepeat && canRepeatTransaction(tx)) {
			actions.push({
				icon: 'create',
				label: $_('transactions.repeat'),
				onclick: () => onrepeat(tx)
			});
		}
		if (canMakeRecurring(tx)) {
			actions.push({
				icon: 'repeat',
				label: $_('recurring.fromTransaction'),
				onclick: () => onmakeRecurring?.(tx)
			});
		}
		if (showEdit && onedit && canEditTransaction(tx)) {
			actions.push({
				icon: 'edit',
				label: $_('common.edit'),
				onclick: () => onedit(tx)
			});
		}
		if (showDelete && ondelete) {
			actions.push({
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => ondelete(tx)
			});
		}
		return actions;
	}
</script>

{#if transactions.length === 0}
	<EmptyStateCard message={emptyMessage} />
{:else}
	<div class="hidden md:block">
		<table class="w-full text-left text-sm">
			<thead>
				<tr style:color="var(--text-muted)">
					<th class="p-3">{$_('transactions.col.date')}</th>
					<th class="p-3">{$_('transactions.col.account')}</th>
					<th class="p-3">{$_('transactions.col.category')}</th>
					<th class="p-3">{$_('transactions.col.amount')}</th>
					{#if showDescription}
						<th class="p-3">{$_('transactions.col.description')}</th>
					{/if}
					{#if showActions}
						<th class="p-3"></th>
					{/if}
				</tr>
			</thead>
			<tbody>
				{#each transactions as tx (tx.id)}
					<tr class="border-t" style:border-color="var(--border)">
						<td class="p-3 align-middle whitespace-nowrap">
							{formatAPIOperationDateTimeForDisplay(tx.transaction_date, tz)}
							{#if tx.kind === 'future'}
								<span title={$_('transactions.planned')}> 📅</span>
							{/if}
						</td>
						<td class="p-3 align-middle whitespace-nowrap">
							<TransactionAccountCell {tx} {siblings} mode="prefix" />
						</td>
						<td class="p-3 align-middle whitespace-nowrap">
							{#if tx.category_icon}
								<span class="inline-flex items-center gap-1 align-middle">
									<CategoryIcon icon={tx.category_icon} size={24} />
									<span class="leading-none">{tx.category_name ?? tx.type}</span>
								</span>
							{:else}
								{tx.category_name ?? tx.type}
							{/if}
						</td>
						<td class="p-3 align-middle whitespace-nowrap tabular-nums font-medium">
							{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}<MoneyDisplay
								value={tx.amount_display}
								class=""
							/>
						</td>
						{#if showDescription}
							<td class="p-3 align-middle" style:color="var(--text-muted)">
								{tx.description ?? ''}
								{#if descriptionExtra}
									{@render descriptionExtra(tx)}
								{/if}
							</td>
						{/if}
						{#if showActions}
							<td class="p-3 align-middle text-right whitespace-nowrap">
								<RowActionsMenu actions={rowActions(tx)} />
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	<div class="space-y-3 md:hidden">
		{#each transactions as tx (tx.id)}
			<article class="rounded-xl border p-4" style:border-color="var(--border)">
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0">
						<p class="text-sm" style:color="var(--text-muted)">
							{formatAPIOperationDateTimeForDisplay(tx.transaction_date, tz)}
							{#if tx.kind === 'future'}
								<span title={$_('transactions.planned')}> 📅</span>
							{/if}
						</p>
						<p class="mt-1 text-sm font-medium">
							<TransactionAccountCell {tx} {siblings} mode="prefix" />
						</p>
					</div>
					<div class="flex shrink-0 items-start gap-2">
						<p class="text-sm font-semibold tabular-nums">
							{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}<MoneyDisplay
								value={tx.amount_display}
								class=""
							/>
						</p>
						{#if showActions}
							<RowActionsMenu actions={rowActions(tx)} />
						{/if}
					</div>
				</div>
				<p class="mt-2 text-sm">
					{#if tx.category_icon}
						<span class="inline-flex items-center gap-1 align-middle">
							<CategoryIcon icon={tx.category_icon} size={24} />
							<span class="leading-none">{tx.category_name ?? tx.type}</span>
						</span>
					{:else}
						{tx.category_name ?? tx.type}
					{/if}
				</p>
				{#if showDescription && (tx.description || descriptionExtra)}
					<p class="mt-2 text-sm" style:color="var(--text-muted)">
						{tx.description ?? ''}
						{#if descriptionExtra}
							{@render descriptionExtra(tx)}
						{/if}
					</p>
				{/if}
			</article>
		{/each}
	</div>
{/if}
