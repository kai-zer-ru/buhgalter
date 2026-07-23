<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { listAccounts, settleDebt, type Account, type Debt } from '$lib/api/client';
	import { defaultAccountId } from '$lib/accounts';
	import { accountSelectOptions } from '$lib/select-options';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { operationDatetimePickerCreate } from '$lib/datetime-picker-standards';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { toast } from '$lib/toast';
	import { fromDatetimeLocalValue, nowDatetimeLocal } from '$lib/dates';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { formatMoneyForInput, toAPIAmount } from '$lib/money';
	import { user } from '$lib/stores/auth';

	type Props = {
		variant?: 'modal' | 'page';
		open?: boolean;
		backHref?: string;
		debt: Debt | null;
		onclose: () => void;
		onsaved: () => void;
	};

	let {
		variant = 'modal',
		open = $bindable(false),
		backHref = '/debts',
		debt = $bindable(),
		onclose,
		onsaved
	}: Props = $props();

	let amount = $state('');
	let settledAtLocal = $state('');
	let accountId = $state('');
	let skipBalance = $state(false);
	let accounts = $state<Account[]>([]);
	let saving = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const accountOptions = $derived(accountSelectOptions(accounts));
	const active = $derived(variant === 'page' || open);

	$effect(() => {
		if (!active || !debt) return;
		void init();
	});

	async function init() {
		if (!debt) return;
		const currentDebt = debt;
		amount = formatMoneyForInput(currentDebt.amount_display);
		settledAtLocal = nowDatetimeLocal(tz);
		skipBalance = false;
		accounts = await listAccounts('active');
		accountId =
			currentDebt.account_id && accounts.some((a) => a.id === currentDebt.account_id)
				? currentDebt.account_id
				: defaultAccountId(accounts);
	}

	async function save() {
		if (!debt) return;
		saving = true;
		try {
			const settled_at = fromDatetimeLocalValue(settledAtLocal, tz);
			const affectsBalance = !skipBalance;
			if (affectsBalance && !accountId) {
				throw new Error($_('transactions.field.account'));
			}
			await settleDebt(debt.id, {
				amount: toAPIAmount(amount),
				settled_at,
				affects_balance: affectsBalance,
				account_id: affectsBalance ? accountId : undefined
			});
			if (variant === 'modal') {
				open = false;
				debt = null;
			}
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}

	function close() {
		if (variant === 'modal') {
			open = false;
			debt = null;
		}
		onclose();
	}
</script>

{#if debt}
	{@const settledDebt = debt}
	{#snippet formBody()}
		<div class="space-y-4">
			<p class="text-sm" style:color="var(--text-muted)">
				{settledDebt.debtor_name} · <MoneyDisplay value={settledDebt.amount_display} class="" />
			</p>

			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('debts.settle.amount')}</span>
				<MoneyInput bind:value={amount} />
			</label>

			<DateTimePicker
				label={$_('debts.settle.date')}
				bind:value={settledAtLocal}
				{...operationDatetimePickerCreate}
				usePortal={variant === 'modal'}
				required
			/>

			<div class="space-y-1">
				<div class="flex items-center justify-between gap-4">
					<div>
						<p class="text-sm">{$_('debts.settle.skipBalance')}</p>
						<FieldHint text={$_('debts.settle.skipBalanceHint')} />
					</div>
					<ToggleSwitch
						checked={skipBalance}
						label={$_('debts.settle.skipBalance')}
						onchange={() => (skipBalance = !skipBalance)}
					/>
				</div>
			</div>

			{#if !skipBalance}
				<Select
					label={$_('transactions.field.account')}
					bind:value={accountId}
					options={accountOptions}
					usePortal={variant === 'modal'}
				/>
			{/if}
		</div>
	{/snippet}

	{#snippet formFooter()}
		<button type="button" class="btn-ghost" onclick={close}>{$_('common.cancel')}</button>
		<button type="button" class="btn-primary" disabled={saving} onclick={() => void save()}>
			{saving ? $_('common.loading') : $_('debts.action.settle')}
		</button>
	{/snippet}

	{#if variant === 'page'}
		<FormPageShell title={$_('debts.settle.title')} {backHref} onback={close}>
			{@render formBody()}
			{#snippet footer()}
				{@render formFooter()}
			{/snippet}
		</FormPageShell>
	{:else}
		<ModalShell bind:open title={$_('debts.settle.title')} onclose={close}>
			{@render formBody()}
			{#snippet footer()}
				{@render formFooter()}
			{/snippet}
		</ModalShell>
	{/if}
{/if}
