<script lang="ts">
	import type { Snippet } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { detachReceipt, updateCreditPaymentTime, type Transaction } from '$lib/api/client';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { receiptViewPath } from '$lib/android/form-routes';
	import { formatApiError } from '$lib/api/errors';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import IconButton from '$lib/components/IconButton.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import TransactionAccountCell from '$lib/components/TransactionAccountCell.svelte';
	import { formatAPIOperationDateTimeForDisplay, fromAPIDateTime } from '$lib/dates';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import {
		transactionAmountSign,
		transactionCategoryLabel,
		resolveTransactionSubcategoryName,
		transferCommissionDisplay,
		canEditTransaction,
		canRepeatTransaction,
		canDeleteTransaction,
		canLinkToSubscription,
		canAttachReceipt,
		transactionContributor
	} from '$lib/transaction-display';
	import { user } from '$lib/stores/auth';
	import { isReceiptScannerEnabled } from '$lib/receipt/receipt-module';
	import { afterOnlineWrite } from '$lib/offline/after-online-write';
	import { isPendingTransaction, pendingSyncFailed } from '$lib/offline/pending-display';
	import { touchLedgerAfterTransactionMutation } from '$lib/offline/ref-cache-mutations';
	import TransactionTimeDialog from '$lib/components/TransactionTimeDialog.svelte';
	import TransactionCategoryCell from '$lib/components/TransactionCategoryCell.svelte';
	import TransactionMerchantTags from '$lib/components/TransactionMerchantTags.svelte';
	import { featureFlags, isFeatureEnabled } from '$lib/features';
	import { isNativeApp } from '$lib/platform/native';

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
		onsaveAsTemplate,
		onmakeRecurring,
		onmakeSubscription,
		onattachSubscription,
		onattachCredit,
		onattachReceipt,
		onreceiptChanged,
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
		onsaveAsTemplate?: (tx: Transaction) => void;
		onmakeRecurring?: (tx: Transaction) => void;
		onmakeSubscription?: (tx: Transaction) => void;
		onattachSubscription?: (tx: Transaction) => void;
		onattachCredit?: (tx: Transaction) => void;
		onattachReceipt?: (tx: Transaction) => void;
		onreceiptChanged?: () => void;
		descriptionExtra?: Snippet<[Transaction]>;
	} = $props();

	let clockTx = $state<Transaction | null>(null);
	let clockValue = $state('');
	let clockSaving = $state(false);
	let clockSaveGen = 0;
	let clockOverrides = $state<Record<string, string>>({});

	const templatesEnabled = $derived(isFeatureEnabled('transaction_templates', $featureFlags));
	const recurringEnabled = $derived(isFeatureEnabled('recurring', $featureFlags));
	const subscriptionsEnabled = $derived(isFeatureEnabled('subscriptions', $featureFlags));
	const receiptScannerActive = $derived(isReceiptScannerEnabled($featureFlags));
	const compactLayout = isNativeApp();

	const showActions = $derived(
		Boolean(
			(showDelete && ondelete) ||
			(showEdit && onedit) ||
			onrepeat ||
			(onsaveAsTemplate && templatesEnabled) ||
			(onmakeRecurring && recurringEnabled) ||
			(onmakeSubscription && subscriptionsEnabled) ||
			(onattachSubscription && subscriptionsEnabled) ||
			onattachCredit ||
			(onattachReceipt && receiptScannerActive) ||
			(onreceiptChanged && receiptScannerActive) ||
			transactions.some((tx) => tx.credit_payment_linked && !isPendingTransaction(tx))
		)
	);

	function canDetachReceipt(tx: Transaction): boolean {
		return Boolean(
			receiptScannerActive &&
			onreceiptChanged &&
			tx.type === 'expense' &&
			tx.receipt_import_id &&
			!tx.transfer_group_id
		);
	}

	function openReceiptView(tx: Transaction) {
		if (!tx.receipt_import_id) return;
		void goto(
			resolve(
				receiptViewPath(tx.receipt_import_id, {
					transactionId: tx.id
				})
			)
		);
	}

	async function detachReceiptFromRow(tx: Transaction) {
		const ok = await confirm({
			title: $_('receipt.detach.title'),
			message: $_('receipt.detach.message'),
			confirmLabel: $_('receipt.detach.confirm'),
			cancelLabel: $_('common.cancel'),
			danger: true
		});
		if (!ok) return;
		try {
			await detachReceipt(tx.id);
			toast.success($_('receipt.detach.success'));
			onreceiptChanged?.();
		} catch (e) {
			toast.error(formatApiError(e, 'receipt.detach.error'));
		}
	}

	function contributorLine(tx: Transaction): string | null {
		const info = transactionContributor(tx, $user?.id);
		if (!info) return null;
		if (info.kind === 'deleted') return $_('transactions.contributedDeleted');
		return $_('transactions.contributed', { values: { name: info.name } });
	}

	function rowDateLabel(tx: Transaction): string {
		return formatAPIOperationDateTimeForDisplay(clockOverrides[tx.id] ?? tx.transaction_date, tz);
	}

	function clockInputValue(apiDate: string): string {
		try {
			const local = fromAPIDateTime(apiDate, tz);
			const hour = String(local.getHours()).padStart(2, '0');
			const minute = String(local.getMinutes()).padStart(2, '0');
			return `${hour}:${minute}`;
		} catch {
			return '00:00';
		}
	}

	function openClock(tx: Transaction) {
		clockSaveGen += 1;
		clockTx = tx;
		clockValue = clockInputValue(clockOverrides[tx.id] ?? tx.transaction_date);
		clockSaving = false;
	}

	async function saveClock() {
		if (!clockTx || clockSaving || !clockValue) return;
		const id = clockTx.id;
		const time = clockValue;
		const gen = clockSaveGen;
		clockSaving = true;
		try {
			const updated = await updateCreditPaymentTime(id, time);
			clockOverrides = { ...clockOverrides, [id]: updated.transaction_date };
			touchLedgerAfterTransactionMutation();
			afterOnlineWrite();
			if (clockSaveGen === gen && clockTx?.id === id) clockTx = null;
		} catch (e) {
			if (clockSaveGen === gen && clockTx?.id === id)
				toast.error(formatApiError(e, 'common.error'));
		} finally {
			if (clockSaveGen === gen) clockSaving = false;
		}
	}

	function canChangeClock(tx: Transaction): boolean {
		return Boolean(tx.credit_payment_linked) && !isPendingTransaction(tx);
	}

	function canMakeRecurring(tx: Transaction): boolean {
		return Boolean(onmakeRecurring && tx.type !== 'transfer' && !tx.category_is_system);
	}

	function canMakeSubscription(tx: Transaction): boolean {
		return Boolean(onmakeSubscription && canLinkToSubscription(tx));
	}

	function canAttachSubscription(tx: Transaction): boolean {
		return Boolean(onattachSubscription && canLinkToSubscription(tx));
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
		if (templatesEnabled && onsaveAsTemplate && canRepeatTransaction(tx)) {
			actions.push({
				icon: 'save',
				label: $_('templates.saveAs'),
				onclick: () => onsaveAsTemplate(tx)
			});
		}
		if (recurringEnabled && canMakeRecurring(tx)) {
			actions.push({
				icon: 'repeat',
				label: $_('recurring.fromTransaction'),
				onclick: () => onmakeRecurring?.(tx)
			});
		}
		if (subscriptionsEnabled && canMakeSubscription(tx)) {
			actions.push({
				icon: 'pay',
				label: $_('subscriptions.fromTransaction'),
				onclick: () => onmakeSubscription?.(tx)
			});
		}
		if (subscriptionsEnabled && canAttachSubscription(tx)) {
			actions.push({
				icon: 'add',
				label: $_('subscriptions.attachToSubscription'),
				onclick: () => onattachSubscription?.(tx)
			});
		}
		if (
			onattachCredit &&
			tx.type === 'expense' &&
			!tx.credit_payment_linked &&
			!tx.subscription_id &&
			!tx.transfer_group_id
		) {
			actions.push({
				icon: 'pay',
				label: $_('credits.attachFromTransaction'),
				onclick: () => onattachCredit?.(tx)
			});
		}
		if (receiptScannerActive && onattachReceipt && canAttachReceipt(tx)) {
			actions.push({
				icon: 'receipt',
				label: $_('receipt.attach.menu'),
				onclick: () => onattachReceipt?.(tx)
			});
		}
		if (canDetachReceipt(tx)) {
			actions.push({
				icon: 'delete',
				label: $_('receipt.detach.menu'),
				variant: 'danger',
				onclick: () => void detachReceiptFromRow(tx)
			});
		}
		if (showEdit && onedit && canEditTransaction(tx)) {
			actions.push({
				icon: 'edit',
				label: $_('common.edit'),
				onclick: () => onedit(tx)
			});
		}
		if (canChangeClock(tx)) {
			actions.push({
				icon: 'edit',
				label: $_('transactions.changeTime'),
				onclick: () => openClock(tx)
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
{:else if compactLayout}
	<div class="space-y-3">
		{#each transactions as tx (tx.id)}
			{@const commissionDisplay = transferCommissionDisplay(tx, siblings)}
			<article class="rounded-xl border p-4" style:border-color="var(--border)">
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0">
						<p class="text-sm" style:color="var(--text-muted)">
							{rowDateLabel(tx)}
							{#if tx.kind === 'future'}
								<span title={$_('transactions.planned')}> 📅</span>
							{/if}
							{#if pendingSyncFailed(tx)}
								<span title={pendingSyncFailed(tx)}> ⚠️</span>
							{:else if isPendingTransaction(tx)}
								<span title={$_('offline.pending')}> ⏳</span>
							{/if}
						</p>
						{#if contributorLine(tx)}
							<p class="mt-0.5 text-xs" style:color="var(--text-muted)">{contributorLine(tx)}</p>
						{/if}
						<p class="mt-1 text-sm font-medium">
							<TransactionAccountCell {tx} {siblings} mode="prefix" />
						</p>
					</div>
					<div class="flex shrink-0 items-start gap-2">
						<div class="text-right">
							<div class="inline-flex items-center justify-end gap-1">
								<p class="text-sm font-semibold tabular-nums">
									{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}<MoneyDisplay
										value={tx.amount_display}
										currency={tx.currency}
										class=""
									/>
								</p>
								{#if tx.receipt_import_id}
									<IconButton
										icon="receipt"
										label={$_('receipt.view')}
										class="!h-8 !w-8"
										onclick={() => openReceiptView(tx)}
									/>
								{/if}
							</div>
							{#if commissionDisplay}
								<p class="text-xs tabular-nums" style:color="var(--text-muted)">
									{$_('transactions.commission')}:
									<MoneyDisplay value={commissionDisplay} currency={tx.currency} class="" />
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
							categoryName={transactionCategoryLabel(tx, $_)}
							categoryIcon={tx.category_icon}
							subcategoryName={resolveTransactionSubcategoryName(tx)}
							subcategoryIcon={tx.subcategory_icon}
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
							{rowDateLabel(tx)}
							{#if tx.kind === 'future'}
								<span title={$_('transactions.planned')}> 📅</span>
							{/if}
							{#if pendingSyncFailed(tx)}
								<span title={pendingSyncFailed(tx)}> ⚠️</span>
							{:else if isPendingTransaction(tx)}
								<span title={$_('offline.pending')}> ⏳</span>
							{/if}
							{#if contributorLine(tx)}
								<div class="text-xs font-normal" style:color="var(--text-muted)">
									{contributorLine(tx)}
								</div>
							{/if}
						</td>
						<td class="p-3 align-middle whitespace-nowrap">
							<TransactionAccountCell {tx} {siblings} mode="prefix" />
						</td>
						{#if showCategory}
							<td class="p-3 align-middle whitespace-nowrap">
								<TransactionCategoryCell
									categoryName={transactionCategoryLabel(tx, $_)}
									categoryIcon={tx.category_icon}
									subcategoryName={resolveTransactionSubcategoryName(tx)}
									subcategoryIcon={tx.subcategory_icon}
								/>
							</td>
						{/if}
						<td class="p-3 align-middle whitespace-nowrap tabular-nums font-medium">
							<div class="inline-flex items-center gap-1">
								{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}<MoneyDisplay
									value={tx.amount_display}
									currency={tx.currency}
									class=""
								/>
								{#if tx.receipt_import_id}
									<IconButton
										icon="receipt"
										label={$_('receipt.view')}
										class="!h-8 !w-8"
										onclick={() => openReceiptView(tx)}
									/>
								{/if}
							</div>
							{#if commissionDisplay}
								<div class="text-xs font-normal" style:color="var(--text-muted)">
									{$_('transactions.commission')}:
									<MoneyDisplay value={commissionDisplay} currency={tx.currency} class="" />
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
							{rowDateLabel(tx)}
							{#if tx.kind === 'future'}
								<span title={$_('transactions.planned')}> 📅</span>
							{/if}
							{#if pendingSyncFailed(tx)}
								<span title={pendingSyncFailed(tx)}> ⚠️</span>
							{:else if isPendingTransaction(tx)}
								<span title={$_('offline.pending')}> ⏳</span>
							{/if}
						</p>
						{#if contributorLine(tx)}
							<p class="mt-0.5 text-xs" style:color="var(--text-muted)">{contributorLine(tx)}</p>
						{/if}
						<p class="mt-1 text-sm font-medium">
							<TransactionAccountCell {tx} {siblings} mode="prefix" />
						</p>
					</div>
					<div class="flex shrink-0 items-start gap-2">
						<div class="text-right">
							<div class="inline-flex items-center justify-end gap-1">
								<p class="text-sm font-semibold tabular-nums">
									{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}<MoneyDisplay
										value={tx.amount_display}
										currency={tx.currency}
										class=""
									/>
								</p>
								{#if tx.receipt_import_id}
									<IconButton
										icon="receipt"
										label={$_('receipt.view')}
										class="!h-8 !w-8"
										onclick={() => openReceiptView(tx)}
									/>
								{/if}
							</div>
							{#if commissionDisplay}
								<p class="text-xs tabular-nums" style:color="var(--text-muted)">
									{$_('transactions.commission')}:
									<MoneyDisplay value={commissionDisplay} currency={tx.currency} class="" />
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
							categoryName={transactionCategoryLabel(tx, $_)}
							categoryIcon={tx.category_icon}
							subcategoryName={resolveTransactionSubcategoryName(tx)}
							subcategoryIcon={tx.subcategory_icon}
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

<TransactionTimeDialog
	open={clockTx !== null}
	bind:time={clockValue}
	saving={clockSaving}
	onclose={() => (clockTx = null)}
	onsave={() => void saveClock()}
/>
