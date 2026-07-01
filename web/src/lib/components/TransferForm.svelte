<script lang="ts">
	import { _ } from 'svelte-i18n';
	import {
		createTransfer,
		listAccounts,
		updateTransfer,
		type Account,
		type Transaction
	} from '$lib/api/client';
	import { ApiError } from '$lib/api/client';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import {
		operationDatetimePickerCreate,
		operationDatetimePickerEdit
	} from '$lib/datetime-picker-standards';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { maxCreditCardPaymentKopecks, resolvePaymentAccountId } from '$lib/credit-card';
	import { fromDatetimeLocalValue, nowDatetimeLocal, toDatetimeLocalValue } from '$lib/dates';
	import { formatMoneyForInput, fromCents, toAPIAmount, toCents } from '$lib/money';
	import { transferAccountIds, transferGroupLegs, transferOutLeg } from '$lib/transaction-display';
	import { pickOtherAccountId, transferAccountOptions } from '$lib/transfer-accounts';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	type Props = {
		open: boolean;
		accountId?: string;
		creditCardPay?: Account | null;
		editTx?: Transaction | null;
		repeatFrom?: Transaction | null;
		siblings?: Transaction[];
		onclose: () => void;
		onsaved: () => void;
	};

	let {
		open = $bindable(),
		accountId = '',
		creditCardPay = null,
		editTx = null,
		repeatFrom = null,
		siblings = [],
		onclose,
		onsaved
	}: Props = $props();

	let fromAccount = $state('');
	let toAccount = $state('');
	let amount = $state('');
	let commission = $state('');
	let description = $state('');
	let dateTimeValue = $state('');
	let accounts = $state<Account[]>([]);
	let saving = $state(false);
	let error = $state('');
	let groupId = $state('');

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const editing = $derived(Boolean(groupId));
	const fromAcc = $derived(accounts.find((a) => a.id === fromAccount));
	const canFullBalance = $derived(Boolean(fromAcc && fromAcc.balance > 0 && !editing));
	const fromAccountOptions = $derived(
		creditCardPay
			? transferAccountOptions(
					accounts.filter((a) => a.type !== 'credit_card'),
					toAccount
				)
			: transferAccountOptions(accounts, toAccount)
	);
	const toAccountOptions = $derived(transferAccountOptions(accounts, fromAccount));
	const payMode = $derived(Boolean(creditCardPay && !editing));
	const maxPayKopecks = $derived(creditCardPay ? maxCreditCardPaymentKopecks(creditCardPay) : null);
	const payExceedsMax = $derived(
		maxPayKopecks != null && amount.trim() !== '' && toCents(amount) > maxPayKopecks
	);
	const canFullLimit = $derived(payMode && maxPayKopecks != null && maxPayKopecks > 0);

	$effect(() => {
		if (!open) return;
		void init(editTx, repeatFrom, siblings, accountId, creditCardPay);
	});

	$effect(() => {
		if (!open || !fromAccount || !toAccount || fromAccount !== toAccount) return;
		toAccount = pickOtherAccountId(accounts, fromAccount);
	});

	async function init(
		editSource: Transaction | null | undefined,
		repeatSource: Transaction | null | undefined,
		related: Transaction[],
		contextAccountId: string,
		payCard: Account | null | undefined
	) {
		error = '';
		if (editSource?.transfer_group_id) {
			const legs = transferGroupLegs(editSource, related);
			const metaLeg = legs.length >= 2 ? transferOutLeg(editSource, legs) : editSource;
			const commissionLeg = legs.find((leg) => leg.type === 'expense');
			const { fromAccountId, toAccountId } = transferAccountIds(editSource, related);
			groupId = editSource.transfer_group_id;
			fromAccount = fromAccountId;
			toAccount = toAccountId;
			amount = formatMoneyForInput(metaLeg.amount_display);
			commission = formatMoneyForInput(commissionLeg?.amount_display ?? '');
			description = metaLeg.description ?? '';
			dateTimeValue = toDatetimeLocalValue(metaLeg.transaction_date, tz);
		} else if (repeatSource?.transfer_group_id) {
			const legs = transferGroupLegs(repeatSource, related);
			const metaLeg = legs.length >= 2 ? transferOutLeg(repeatSource, legs) : repeatSource;
			const commissionLeg = legs.find((leg) => leg.type === 'expense');
			const { fromAccountId, toAccountId } = transferAccountIds(repeatSource, related);
			groupId = '';
			fromAccount = fromAccountId;
			toAccount = toAccountId;
			amount = formatMoneyForInput(metaLeg.amount_display);
			commission = formatMoneyForInput(commissionLeg?.amount_display ?? '');
			description = metaLeg.description ?? '';
			dateTimeValue = nowDatetimeLocal(tz);
		} else {
			groupId = '';
			fromAccount = '';
			toAccount = '';
			amount = '';
			commission = '';
			description = '';
			dateTimeValue = nowDatetimeLocal(tz);
		}
		accounts = await listAccounts('active');
		if (!editSource?.transfer_group_id && !repeatSource?.transfer_group_id) {
			if (payCard) {
				fromAccount = resolvePaymentAccountId(payCard, accounts) ?? '';
				toAccount = payCard.id;
			} else {
				const from = defaultAccountId(accounts, contextAccountId);
				fromAccount = from;
				toAccount = pickOtherAccountId(accounts, from);
			}
		}
	}

	async function save(e: Event) {
		e.preventDefault();
		saving = true;
		error = '';
		const payload = {
			from_account_id: fromAccount,
			to_account_id: toAccount,
			amount: toAPIAmount(amount),
			commission: commission.trim() ? toAPIAmount(commission) : undefined,
			description: description || undefined,
			transaction_date: fromDatetimeLocalValue(dateTimeValue, tz)
		};
		try {
			if (groupId) {
				await updateTransfer(groupId, payload);
			} else {
				await createTransfer(payload);
			}
			open = false;
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			saving = false;
		}
	}

	function fillFullBalance() {
		if (!fromAcc || fromAcc.balance <= 0) return;
		amount = fromAcc.balance_display;
	}

	function fillFullLimit() {
		if (maxPayKopecks == null || maxPayKopecks <= 0) return;
		amount = formatMoneyForInput(fromCents(maxPayKopecks));
	}

	function swapAccounts() {
		const from = fromAccount;
		fromAccount = toAccount;
		toAccount = from;
	}

	function close() {
		open = false;
		onclose();
	}
</script>

<ModalShell
	bind:open
	title={payMode ? $_('accounts.creditCard.pay') : $_('transactions.transfer')}
	onclose={close}
>
	<form id="transfer-form" class="space-y-4" onsubmit={save}>
		{#if payMode}
			<Select
				id="from-acc"
				label={$_('transactions.field.from')}
				bind:value={fromAccount}
				options={fromAccountOptions}
				usePortal
			/>
		{:else}
			<div class="grid grid-cols-[auto_1fr] gap-x-2 gap-y-4">
				<button
					type="button"
					class="btn-ghost col-start-1 row-start-1 row-span-2 flex h-11 w-10 shrink-0 items-center justify-center self-center"
					onclick={swapAccounts}
					aria-label={$_('transfers.swap_accounts')}
					title={$_('transfers.swap_accounts')}
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<path d="M7 16V4M7 4L3 8M7 4l4 4" />
						<path d="M17 8v12M17 20l4-4M17 20l-4-4" />
					</svg>
				</button>
				<div class="col-start-2 row-start-1">
					<Select
						id="from-acc"
						label={$_('transactions.field.from')}
						bind:value={fromAccount}
						options={fromAccountOptions}
						usePortal
					/>
				</div>
				<div class="col-start-2 row-start-2">
					<Select
						id="to-acc"
						label={$_('transactions.field.to')}
						bind:value={toAccount}
						options={toAccountOptions}
						usePortal
					/>
				</div>
			</div>
		{/if}
		<div>
			<label class="mb-1 block text-sm font-medium" for="tr-amount"
				>{$_('transactions.field.amount')}</label
			>
			<MoneyInput id="tr-amount" bind:value={amount} required />
			{#if payExceedsMax && maxPayKopecks != null}
				<p class="mt-1 text-sm" style:color="var(--danger)">
					{$_('accounts.creditCard.payExceedsMax', {
						values: { max: fromCents(maxPayKopecks) }
					})}
				</p>
			{/if}
			<div class="mt-1 flex flex-wrap gap-2">
				<button
					type="button"
					class="btn-ghost text-sm"
					disabled={!canFullBalance}
					onclick={fillFullBalance}
				>
					{$_('transfers.full_balance')}
				</button>
				{#if payMode}
					<button
						type="button"
						class="btn-ghost text-sm"
						disabled={!canFullLimit}
						onclick={fillFullLimit}
					>
						{$_('accounts.creditCard.payFullLimit')}
					</button>
				{/if}
			</div>
		</div>
		<div>
			<label class="mb-1 block text-sm font-medium" for="tr-commission"
				>{$_('transfers.field.commission')}</label
			>
			<MoneyInput id="tr-commission" bind:value={commission} />
		</div>
		<div>
			<label class="mb-1 block text-sm font-medium" for="tr-desc"
				>{$_('transactions.field.description')}</label
			>
			<input id="tr-desc" class="input w-full" bind:value={description} />
		</div>
		<DateTimePicker
			id="tr-date"
			label={$_('transactions.field.dateOnly')}
			bind:value={dateTimeValue}
			{...editing ? operationDatetimePickerEdit : operationDatetimePickerCreate}
			usePortal
			required
		/>
		<FormFeedback {error} />
	</form>

	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={close}>{$_('common.cancel')}</button>
		<button type="submit" form="transfer-form" class="btn-primary" disabled={saving}>
			{saving ? $_('common.loading') : $_('common.save')}
		</button>
	{/snippet}
</ModalShell>
