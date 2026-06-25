<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { createTransfer, listAccounts, type Account } from '$lib/api/client';
	import { ApiError } from '$lib/api/client';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { fromDatetimeLocalValue, nowDatetimeLocal } from '$lib/dates';
	import { toAPIAmount } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	type Props = {
		open: boolean;
		onclose: () => void;
		onsaved: () => void;
	};

	let { open = $bindable(), onclose, onsaved }: Props = $props();

	let fromAccount = $state('');
	let toAccount = $state('');
	let amount = $state('');
	let commission = $state('');
	let description = $state('');
	let dateTimeValue = $state('');
	let accounts = $state<Account[]>([]);
	let saving = $state(false);
	let error = $state('');

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const fromAcc = $derived(accounts.find((a) => a.id === fromAccount));
	const canFullBalance = $derived(Boolean(fromAcc && fromAcc.balance > 0));
	const accountOptions = $derived(accounts.map((acc) => ({ value: acc.id, label: acc.name })));

	$effect(() => {
		if (!open) return;
		void init();
	});

	async function init() {
		error = '';
		accounts = await listAccounts('active');
		const primary = defaultAccountId(accounts);
		fromAccount = primary;
		toAccount = accounts.find((a) => a.id !== primary)?.id ?? primary;
		amount = '';
		commission = '';
		description = '';
		dateTimeValue = nowDatetimeLocal(tz);
	}

	async function save(e: Event) {
		e.preventDefault();
		saving = true;
		error = '';
		try {
			await createTransfer({
				from_account_id: fromAccount,
				to_account_id: toAccount,
				amount: toAPIAmount(amount),
				commission: commission.trim() ? toAPIAmount(commission) : undefined,
				description: description || undefined,
				transaction_date: fromDatetimeLocalValue(dateTimeValue, tz)
			});
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

<ModalShell bind:open title={$_('transactions.transfer')} onclose={close}>
	<form id="transfer-form" class="space-y-4" onsubmit={save}>
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
					options={accountOptions}
					usePortal
				/>
			</div>
			<div class="col-start-2 row-start-2">
				<Select
					id="to-acc"
					label={$_('transactions.field.to')}
					bind:value={toAccount}
					options={accountOptions}
					usePortal
				/>
			</div>
		</div>
		<div>
			<label class="mb-1 block text-sm font-medium" for="tr-amount"
				>{$_('transactions.field.amount')}</label
			>
			<MoneyInput id="tr-amount" bind:value={amount} required />
			<button
				type="button"
				class="btn-ghost mt-1 text-sm"
				disabled={!canFullBalance}
				onclick={fillFullBalance}
			>
				{$_('transfers.full_balance')}
			</button>
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
			timeMode="optional"
			defaultTime="now"
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
