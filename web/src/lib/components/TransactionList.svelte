<script lang="ts">
	import type { Snippet } from 'svelte';
	import { _ } from 'svelte-i18n';
	import type { Transaction } from '$lib/api/client';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import TransactionAccountCell from '$lib/components/TransactionAccountCell.svelte';
	import { formatAPIDateTimeForDisplay } from '$lib/dates';
	import { formatMoneyDisplay } from '$lib/money';
	import { transactionAmountSign } from '$lib/transaction-display';

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
		descriptionExtra?: Snippet<[Transaction]>;
	} = $props();

	const showActions = $derived(Boolean((showDelete && ondelete) || (showEdit && onedit)));
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
						<td class="p-3 whitespace-nowrap">
							{formatAPIDateTimeForDisplay(tx.transaction_date, tz)}
							{#if tx.kind === 'future'}
								<span title={$_('transactions.planned')}> 📅</span>
							{/if}
						</td>
						<td class="p-3">
							<TransactionAccountCell {tx} {siblings} mode="prefix" />
						</td>
						<td class="p-3">
							{#if tx.category_icon}
								<span class="inline-flex items-center gap-1">
									<CategoryIcon icon={tx.category_icon} size={16} />
									{tx.category_name ?? tx.type}
								</span>
							{:else}
								{tx.category_name ?? tx.type}
							{/if}
						</td>
						<td class="p-3 tabular-nums font-medium">
							{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}
							{formatMoneyDisplay(tx.amount_display)}
						</td>
						{#if showDescription}
							<td class="p-3" style:color="var(--text-muted)">
								{tx.description ?? ''}
								{#if descriptionExtra}
									{@render descriptionExtra(tx)}
								{/if}
							</td>
						{/if}
						{#if showActions}
							<td class="p-3 text-right whitespace-nowrap">
								<div class="flex items-center justify-end gap-0.5">
									{#if showEdit && onedit && tx.type !== 'transfer'}
										<IconButton icon="edit" label={$_('common.edit')} onclick={() => onedit(tx)} />
									{/if}
									{#if showDelete && ondelete}
										<IconButton
											icon="delete"
											label={$_('common.delete')}
											variant="danger"
											onclick={() => ondelete(tx)}
										/>
									{/if}
								</div>
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
							{formatAPIDateTimeForDisplay(tx.transaction_date, tz)}
							{#if tx.kind === 'future'}
								<span title={$_('transactions.planned')}> 📅</span>
							{/if}
						</p>
						<p class="mt-1 font-medium">
							<TransactionAccountCell {tx} {siblings} mode="prefix" />
						</p>
					</div>
					<p class="shrink-0 text-base font-semibold tabular-nums">
						{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}
						{formatMoneyDisplay(tx.amount_display)}
					</p>
				</div>
				<p class="mt-2 text-sm">
					{#if tx.category_icon}
						<span class="inline-flex items-center gap-1">
							<CategoryIcon icon={tx.category_icon} size={16} />
							{tx.category_name ?? tx.type}
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
				{#if showActions}
					<div class="mt-3 flex justify-end gap-0.5">
						{#if showEdit && onedit && tx.type !== 'transfer'}
							<IconButton icon="edit" label={$_('common.edit')} onclick={() => onedit(tx)} />
						{/if}
						{#if showDelete && ondelete}
							<IconButton
								icon="delete"
								label={$_('common.delete')}
								variant="danger"
								onclick={() => ondelete(tx)}
							/>
						{/if}
					</div>
				{/if}
			</article>
		{/each}
	</div>
{/if}
