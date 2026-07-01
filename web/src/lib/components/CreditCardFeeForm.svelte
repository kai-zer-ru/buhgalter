<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { createTransaction, listCategories, type Account } from '$lib/api/client';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { operationDatetimePickerCreate } from '$lib/datetime-picker-standards';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import { COMMISSION_USAGE_COMMENT } from '$lib/credit-card';
	import { fromDatetimeLocalValue, nowDatetimeLocal } from '$lib/dates';
	import { toAPIAmount } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	type Props = {
		open: boolean;
		account: Account;
		onclose: () => void;
		onsaved: () => void;
	};

	let { open = $bindable(), account, onclose, onsaved }: Props = $props();

	let amount = $state('');
	let dateTimeValue = $state('');
	let commissionCategoryId = $state('');
	let saving = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');

	$effect(() => {
		if (!open) return;
		amount = '';
		dateTimeValue = nowDatetimeLocal(tz);
		void loadCommissionCategory();
	});

	async function loadCommissionCategory() {
		const cats = await listCategories('expense');
		const commission = cats.find((c) => c.is_system && c.name === 'Комиссия');
		commissionCategoryId = commission?.id ?? '';
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!commissionCategoryId) {
			toast($_('common.error'), 'error');
			return;
		}
		saving = true;
		try {
			await createTransaction({
				account_id: account.id,
				type: 'expense',
				amount: toAPIAmount(amount),
				category_id: commissionCategoryId,
				description: COMMISSION_USAGE_COMMENT,
				transaction_date: fromDatetimeLocalValue(dateTimeValue, tz)
			});
			open = false;
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			toast.fromError(err, 'common.error');
		} finally {
			saving = false;
		}
	}

	function close() {
		open = false;
		onclose();
	}
</script>

<ModalShell bind:open title={$_('accounts.creditCard.chargeFee')} onclose={close}>
	<form id="cc-fee-form" class="space-y-4" onsubmit={save}>
		<div>
			<label class="mb-1 block text-sm font-medium" for="cc-fee-amount"
				>{$_('transactions.field.amount')}</label
			>
			<MoneyInput id="cc-fee-amount" bind:value={amount} required />
		</div>
		<DateTimePicker
			id="cc-fee-date"
			label={$_('transactions.field.dateOnly')}
			bind:value={dateTimeValue}
			{...operationDatetimePickerCreate}
			usePortal
			required
		/>
	</form>

	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={close}>{$_('common.cancel')}</button>
		<button type="submit" form="cc-fee-form" class="btn-primary" disabled={saving}>
			{saving ? $_('common.loading') : $_('common.save')}
		</button>
	{/snippet}
</ModalShell>
