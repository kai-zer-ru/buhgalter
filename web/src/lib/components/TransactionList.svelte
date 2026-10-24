<script lang="ts">
	import type { Snippet } from 'svelte';
	import { _ } from 'svelte-i18n';
	import type { Transaction } from '$lib/api/client';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import TransactionAccountCell from '$lib/components/TransactionAccountCell.svelte';
	import { formatAPIOperationDateTimeForDisplay, fromAPIDateTime } from '$lib/dates';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import {
		transactionAmountSign,
		transferCommissionDisplay,
		canEditTransaction,
		canRepeatTransaction,
		canDeleteTransaction,
		canLinkToSubscription
	} from '$lib/transaction-display';
	import TransactionCategoryCell from '$lib/components/TransactionCategoryCell.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import ReceiptDetailDialog from '$lib/components/ReceiptDetailDialog.svelte';
	import ReceiptAttachForm from '$lib/components/ReceiptAttachForm.svelte';
	import { detachReceipt, updateCreditPaymentTime } from '$lib/api/client';
	import TransactionTimeDialog from '$lib/components/TransactionTimeDialog.svelte';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import { formatApiError } from '$lib/api/errors';

	let receiptOpen = $state(false);
	let attachOpen = $state(false);
	let attachTx = $state<Transaction | null>(null);
	let clockTx = $state<Transaction | null>(null);
	let clockValue = $state('');
	let clockSaving = $state(false);
	let clockSaveGen = 0;
	let clockOverrides = $state<Record<string, string>>({});
	let receiptImportId = $state('');
	let receiptTransactionId = $state('');

	function openReceipt(tx: Transaction) {
		if (!tx.receipt_import_id) return;
		receiptImportId = tx.receipt_import_id;
		receiptTransactionId = tx.id;
		receiptOpen = true;
	}
	import TransactionMerchantTags from '$lib/components/TransactionMerchantTags.svelte';
	import { featureFlags, isFeatureEnabled } from '$lib/features';

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
		onreceiptAttached,
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
		onreceiptAttached?: () => void;
		descriptionExtra?: Snippet<[Transaction]>;
	} = $props();

	/** Attach in row menu when module flag is on (token checked on API). Import «Чек» gated separately on pages. */
	const receiptAttachEnabled = $derived(isFeatureEnabled('receipt_scanner', $featureFlags));
	const templatesEnabled = $derived(isFeatureEnabled('transaction_templates', $featureFlags));
	const recurringEnabled = $derived(isFeatureEnabled('recurring', $featureFlags));
	const subscriptionsEnabled = $derived(isFeatureEnabled('subscriptions', $featureFlags));

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
			(receiptAttachEnabled && onreceiptAttached) ||
			transactions.some((tx) => tx.credit_payment_linked)
		)
	);

	function canMakeRecurring(tx: Transaction): boolean {
		return Boolean(onmakeRecurring && tx.type !== 'transfer' && !tx.category_is_system);
	}

	function canMakeSubscription(tx: Transaction): boolean {
		return Boolean(onmakeSubscription && canLinkToSubscription(tx));
	}

	function canAttachSubscription(tx: Transaction): boolean {
		return Boolean(onattachSubscription && canLinkToSubscription(tx));
	}

	function canAttachCredit(tx: Transaction): boolean {
		return Boolean(
			onattachCredit &&
			tx.type === 'expense' &&
			!tx.credit_payment_linked &&
			!tx.subscription_id &&
			!tx.transfer_group_id
		);
	}

	function canAttachReceipt(tx: Transaction): boolean {
		return Boolean(
			receiptAttachEnabled &&
			onreceiptAttached &&
			tx.type === 'expense' &&
			!tx.receipt_import_id &&
			!tx.transfer_group_id &&
			!tx.category_is_system
		);
	}

	function canDetachReceipt(tx: Transaction): boolean {
		return Boolean(
			receiptAttachEnabled &&
			onreceiptAttached &&
			tx.type === 'expense' &&
			tx.receipt_import_id &&
			!tx.transfer_group_id
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
			onreceiptAttached?.();
		} catch (e) {
			toast.error(formatApiError(e, 'receipt.detach.error'));
		}
	}

	function openAttachReceipt(tx: Transaction) {
		attachTx = tx;
		attachOpen = true;
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
			if (clockSaveGen === gen && clockTx?.id === id) clockTx = null;
		} catch (e) {
			if (clockSaveGen === gen && clockTx?.id === id)
				toast.error(formatApiError(e, 'common.error'));
		} finally {
			if (clockSaveGen === gen) clockSaving = false;
		}
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
		if (canAttachCredit(tx)) {
			actions.push({
				icon: 'pay',
				label: $_('credits.attachFromTransaction'),
				onclick: () => onattachCredit?.(tx)
			});
		}
		if (canAttachReceipt(tx)) {
			actions.push({
				icon: 'receipt',
				label: $_('receipt.attach.menu'),
				onclick: () => openAttachReceipt(tx)
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
		if (tx.credit_payment_linked) {
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
							<div class="inline-flex items-center gap-1">
								{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}<MoneyDisplay
									value={tx.amount_display}
									class=""
								/>
								{#if tx.receipt_import_id}
									<IconButton
										icon="receipt"
										label={$_('receipt.view')}
										class="!h-8 !w-8"
										onclick={() => openReceipt(tx)}
									/>
								{/if}
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
							{rowDateLabel(tx)}
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
							<div class="inline-flex items-center justify-end gap-1">
								<p class="text-sm font-semibold tabular-nums">
									{showAmountSign ? transactionAmountSign(tx, { singleAccount }) : ''}<MoneyDisplay
										value={tx.amount_display}
										class=""
									/>
								</p>
								{#if tx.receipt_import_id}
									<IconButton
										icon="receipt"
										label={$_('receipt.view')}
										class="!h-8 !w-8"
										onclick={() => openReceipt(tx)}
									/>
								{/if}
							</div>
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

<ReceiptDetailDialog
	bind:open={receiptOpen}
	{receiptImportId}
	transactionId={receiptTransactionId}
	canDetach={receiptAttachEnabled && Boolean(onreceiptAttached)}
	onclose={() => (receiptOpen = false)}
	ondetached={() => onreceiptAttached?.()}
/>
<ReceiptAttachForm
	bind:open={attachOpen}
	transaction={attachTx}
	onclose={() => {
		attachOpen = false;
		attachTx = null;
	}}
	onsaved={() => onreceiptAttached?.()}
/>
<TransactionTimeDialog
	open={clockTx !== null}
	bind:time={clockValue}
	saving={clockSaving}
	onclose={() => (clockTx = null)}
	onsave={() => void saveClock()}
/>
