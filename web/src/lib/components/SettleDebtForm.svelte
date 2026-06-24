<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { ApiError, listAccounts, settleDebt, type Account, type Debt } from '$lib/api/client';
	import { defaultAccountId } from '$lib/accounts';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import { toast } from '$lib/toast';
	import { fromDatetimeLocalValue, nowDatetimeLocal } from '$lib/dates';
	import { formatMoneyDisplay, toAPIAmount } from '$lib/money';
	import { user } from '$lib/stores/auth';

	type Props = {
		open: boolean;
		debt: Debt | null;
		onclose: () => void;
		onsaved: () => void;
	};

	let { open = $bindable(), debt = $bindable(), onclose, onsaved }: Props = $props();

	let amount = $state('');
	let settledAtLocal = $state('');
	let accountId = $state('');
	let accounts = $state<Account[]>([]);
	let saving = $state(false);
	let error = $state('');

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const affectsBalance = $derived(debt?.affects_balance ?? false);
	const accountOptions = $derived(accounts.map((acc) => ({ value: acc.id, label: acc.name })));

	$effect(() => {
		if (!open || !debt) return;
		void init();
	});

	async function init() {
		if (!debt) return;
		const currentDebt = debt;
		error = '';
		amount = currentDebt.amount_display;
		settledAtLocal = nowDatetimeLocal(tz);
		accounts = await listAccounts('active');
		accountId =
			currentDebt.account_id && accounts.some((a) => a.id === currentDebt.account_id)
				? currentDebt.account_id
				: defaultAccountId(accounts);
	}

	async function save() {
		if (!debt) return;
		saving = true;
		error = '';
		try {
			const settled_at = fromDatetimeLocalValue(settledAtLocal, tz);
			await settleDebt(debt.id, {
				amount: toAPIAmount(amount),
				settled_at,
				affects_balance: affectsBalance,
				account_id: affectsBalance ? accountId : undefined
			});
			open = false;
			debt = null;
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			error =
				err instanceof ApiError
					? err.message
					: err instanceof Error
						? err.message
						: $_('common.error');
		} finally {
			saving = false;
		}
	}

	function close() {
		open = false;
		debt = null;
		onclose();
	}
</script>

{#if debt}
	{@const settledDebt = debt}
	<ModalShell bind:open title={$_('debts.settle.title')} onclose={close}>
		<div class="space-y-4">
			<p class="text-sm" style:color="var(--text-muted)">
				{settledDebt.debtor_name} · {formatMoneyDisplay(settledDebt.amount_display)}
			</p>

			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('debts.settle.amount')}</span>
				<MoneyInput bind:value={amount} />
			</label>

			<DateTimePicker
				label={$_('debts.settle.date')}
				bind:value={settledAtLocal}
				usePortal
				required
			/>

			{#if affectsBalance}
				<Select
					label={$_('transactions.field.account')}
					bind:value={accountId}
					options={accountOptions}
					usePortal
				/>
			{/if}

			<FormFeedback {error} />
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={close}>{$_('common.cancel')}</button>
			<button type="button" class="btn-primary" disabled={saving} onclick={() => void save()}>
				{saving ? $_('common.loading') : $_('debts.action.settle')}
			</button>
		{/snippet}
	</ModalShell>
{/if}
