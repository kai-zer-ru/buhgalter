<script lang="ts">
	import type { Snippet } from 'svelte';
	import { _ } from 'svelte-i18n';
	import type { Transaction } from '$lib/api/client';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import TransactionAccountCell from '$lib/components/TransactionAccountCell.svelte';
	import { formatAPIOperationDateTimeForDisplay } from '$lib/dates';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import {
		transactionAmountSign,
		transferCommissionDisplay,
		canEditTransaction,
		canRepeatTransaction,
		canDeleteTransaction
	} from '$lib/transaction-display';
	import TransactionCategoryCell from '$lib/components/TransactionCategoryCell.svelte';
	import TransactionMerchantTags from '$lib/components/TransactionMerchantTags.svelte';

	let {
		transactions,
		siblings = [],
		tz,
		emptyMessage,
		showDelete = false,
		showEdit = false,
		showDescription = false,
		showAmountSign = false,
		showCategory = true,
		singleAccount = false,
		ondelete,
		onedit,
		onrepeat,
		onmakeRecurring,
		onmakeSubscription,
		onattachSubscription,
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
		showCategory?: boolean;
		singleAccount?: boolean;
		ondelete?: (tx: Transaction) => void;
		onedit?: (tx: Transaction) => void;
		onrepeat?: (tx: Transaction) => void;
		onmakeRecurring?: (tx: Transaction) => void;
		onmakeSubscription?: (tx: Transaction) => void;
		onattachSubscription?: (tx: Transaction) => void;
		descriptionExtra?: Snippet<[Transaction]>;
	} = $props();

	const showActions = $derived(
		Boolean(
			(showDelete && ondelete) ||
			(showEdit && onedit) ||
			onrepeat ||
			onmakeRecurring ||
			onmakeSubscription ||
			onattachSubscription
		)
	);

	function canMakeRecurring(tx: Transaction): boolean {
		return Boolean(onmakeRecurring && tx.type !== 'transfer' && !tx.category_is_system);
	}

	function canMakeSubscription(tx: Transaction): boolean {
		return Boolean(onmakeSubscription && tx.type === 'expense');
	}

	function canAttachSubscription(tx: Transaction): boolean {
		return Boolean(onattachSubscription && tx.type === 'expense');
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
		if (canMakeSubscription(tx)) {
			actions.push({
				icon: 'pay',
				label: $_('subscriptions.fromTransaction'),
				onclick: () => onmakeSubscription?.(tx)
			});
		}
		if (canAttachSubscription(tx)) {
			actions.push({
				icon: 'add',
				label: $_('subscriptions.attachToSubscription'),
				onclick: () => onattachSubscription?.(tx)
			});
		}
		if (showEdit && onedit && canEditTransaction(tx)) {
			actions.push({
				icon: 'edit',
				label: $_('common.edit'),
				onclick: () => onedit(tx)
			});
		}
		if (showDelete && ondelete && canDeleteTransaction(tx)) {
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
					{#if showCategory}
						<th class="p-3">{$_('transactions.col.category')}</th>
					{/if}
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
					{@const commissionDisplay = transferCommissionDisplay(tx, siblings)}
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
						{#if showCategory}
							<td class="p-3 align-middle whitespace-nowrap">
								<TransactionCategoryCell
									categoryName={tx.category_name}
									categoryIcon={tx.category_icon}
									subcategoryName={tx.subcategory_name}
									subcategoryIcon={tx.subcategory_icon}
									typeFallback={tx.type}
								/>
							</td>
						{/if}
						<td class="p-3 align-middle whitespace-nowrap tabular-nums font-medium">
							<div>
								{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}<MoneyDisplay
									value={tx.amount_display}
									class=""
								/>
							</div>
							{#if commissionDisplay}
								<div class="text-xs font-normal" style:color="var(--text-muted)">
									{$_('transactions.commission')}:
									<MoneyDisplay value={commissionDisplay} class="" />
								</div>
							{/if}
						</td>
						{#if showDescription}
							<td class="p-3 align-middle" style:color="var(--text-muted)">
								<div class="flex flex-wrap items-center gap-x-2 gap-y-1">
									<TransactionMerchantTags
										merchantName={tx.merchant_name}
										merchantIcon={tx.merchant_icon}
										tags={tx.tags}
									/>
									{#if tx.description || descriptionExtra}
										<span>
											{tx.description ?? ''}
											{#if descriptionExtra}
												{@render descriptionExtra(tx)}
											{/if}
										</span>
									{/if}
								</div>
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
			{@const commissionDisplay = transferCommissionDisplay(tx, siblings)}
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
						<div class="text-right">
							<p class="text-sm font-semibold tabular-nums">
								{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}<MoneyDisplay
									value={tx.amount_display}
									class=""
								/>
							</p>
							{#if commissionDisplay}
								<p class="text-xs tabular-nums" style:color="var(--text-muted)">
									{$_('transactions.commission')}:
									<MoneyDisplay value={commissionDisplay} class="" />
								</p>
							{/if}
						</div>
						{#if showActions}
							<RowActionsMenu actions={rowActions(tx)} />
						{/if}
					</div>
				</div>
				{#if showCategory}
					<p class="mt-2 text-sm">
						<TransactionCategoryCell
							categoryName={tx.category_name}
							categoryIcon={tx.category_icon}
							subcategoryName={tx.subcategory_name}
							subcategoryIcon={tx.subcategory_icon}
							typeFallback={tx.type}
						/>
					</p>
				{/if}
				{#if showDescription && (tx.description || descriptionExtra || tx.merchant_name || (tx.tags && tx.tags.length > 0))}
					<div
						class="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm"
						style:color="var(--text-muted)"
					>
						<TransactionMerchantTags
							merchantName={tx.merchant_name}
							merchantIcon={tx.merchant_icon}
							tags={tx.tags}
						/>
						{#if tx.description || descriptionExtra}
							<span>
								{tx.description ?? ''}
								{#if descriptionExtra}
									{@render descriptionExtra(tx)}
								{/if}
							</span>
						{/if}
					</div>
				{/if}
			</article>
		{/each}
	</div>
{/if}
