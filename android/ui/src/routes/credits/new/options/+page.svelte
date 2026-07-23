<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import { defaultAutoDebitTimeLocal } from '$lib/datetime-picker-standards';
	import { formatMoneyForInput } from '$lib/money';
	import { accountSelectOptions } from '$lib/select-options';
	import { user } from '$lib/stores/auth';
	import { toast } from '$lib/toast';
	import {
		applyPaymentOverride,
		creditCreateDraft,
		displayedPayment,
		hasPastSchedulePayments,
		isManualInterval,
		loadCreditCreateRefs,
		patchCreditCreate,
		principalIncomeBlocked,
		refreshCreditCreateSchedule
	} from '$lib/credits/create-draft';
	import {
		creditCreateReturnTo,
		ensureCreditCreateDraft,
		goCreditCreateStep,
		nextCreditCreateStep,
		prevCreditCreateStep
	} from '$lib/credits/create-nav';

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const fromRaw = $derived($page.url.searchParams.get('from'));
	const returnTo = $derived(creditCreateReturnTo(fromRaw));

	let ready = $state(false);
	let debitAccountId = $state('');
	let bankId = $state('');
	let firstPaymentToday = $state(false);
	let principalAffectsBalance = $state(false);
	let retroactive = $state(false);
	let createTransactions = $state(true);
	let debitTimeLocal = $state(defaultAutoDebitTimeLocal);
	let editingPayment = $state(false);
	let paymentDraft = $state('');
	let paymentLabel = $state('—');
	let productType = $state<'credit' | 'installment' | 'mortgage'>('credit');
	let manual = $state(false);
	let incomeBlocked = $state(false);
	let showRetroactive = $state(false);
	let accountOptions = $state<{ value: string; label: string }[]>([]);
	let bankOptions = $state<{ value: string; label: string }[]>([]);
	const showFirstPayment = $derived(productType !== 'mortgage');

	onMount(() => {
		ensureCreditCreateDraft(tz);
		const d = get(creditCreateDraft);
		if (!d) {
			goCreditCreateStep('basics', fromRaw);
			return;
		}
		void loadCreditCreateRefs().then(() => {
			const cur = get(creditCreateDraft);
			if (!cur) return;
			debitAccountId = cur.debitAccountId;
			bankId = cur.bankId;
			firstPaymentToday = cur.firstPaymentToday;
			principalAffectsBalance = cur.principalAffectsBalance;
			retroactive = cur.retroactive;
			createTransactions = cur.createTransactions;
			debitTimeLocal = cur.debitTimeLocal || defaultAutoDebitTimeLocal;
			productType = cur.productType;
			manual = isManualInterval(cur);
			paymentLabel = displayedPayment(cur);
			incomeBlocked = principalIncomeBlocked(cur, tz);
			showRetroactive = hasPastSchedulePayments(cur, tz);
			if (!showRetroactive) retroactive = false;
			accountOptions = accountSelectOptions(cur.accounts);
			bankOptions = [
				{ value: '', label: $_('credits.field.bankNotSelected') },
				...cur.banks.map((b) => ({ value: b.id, label: b.name }))
			];
			ready = true;
			if (!manual) void refreshCreditCreateSchedule(tz).then(syncDerived);
			else syncDerived();
		});
	});

	function syncDerived() {
		const d = get(creditCreateDraft);
		if (!d) return;
		paymentLabel = displayedPayment(d);
		incomeBlocked = principalIncomeBlocked(d, tz);
		showRetroactive = hasPastSchedulePayments(d, tz);
		if (incomeBlocked) principalAffectsBalance = false;
		if (!showRetroactive) retroactive = false;
	}

	function persist() {
		patchCreditCreate({
			debitAccountId,
			bankId,
			firstPaymentToday: showFirstPayment ? firstPaymentToday : false,
			principalAffectsBalance: productType === 'credit' ? principalAffectsBalance : false,
			retroactive: showRetroactive ? retroactive : false,
			retroactiveDebitCount:
				showRetroactive && retroactive ? (get(creditCreateDraft)?.retroactiveDebitCount ?? 0) : 0,
			createTransactions,
			debitTimeLocal: createTransactions ? debitTimeLocal : ''
		});
	}

	async function applyPayment() {
		const ok = await applyPaymentOverride(paymentDraft, tz);
		if (!ok) {
			toast.error($_('credits.error.invalidPayment'));
			return;
		}
		editingPayment = false;
		syncDerived();
	}

	function startPaymentEdit() {
		const d = get(creditCreateDraft);
		paymentDraft = formatMoneyForInput(d?.paymentOverride ?? d?.calculatedPayment ?? '');
		editingPayment = true;
	}

	function goNext() {
		persist();
		if (!debitAccountId) {
			toast.error($_('credits.error.noAccount'));
			return;
		}
		nextCreditCreateStep('options', fromRaw);
	}
</script>

{#if ready}
	<FormPageShell
		title={$_('credits.create.step.options')}
		onback={() => {
			persist();
			prevCreditCreateStep('options', fromRaw, returnTo);
		}}
	>
		<div class="space-y-4">
			<Select
				label={$_('credits.field.debitAccount')}
				bind:value={debitAccountId}
				options={accountOptions}
			/>

			<div class="flex flex-wrap items-center gap-2 text-sm">
				{#if !manual && editingPayment}
					<MoneyInput bind:value={paymentDraft} class="input w-40 tabular-nums" autoFocus />
					<button type="button" class="btn-ghost text-sm" onclick={() => void applyPayment()}>
						{$_('common.save')}
					</button>
					<button type="button" class="btn-ghost text-sm" onclick={() => (editingPayment = false)}>
						{$_('common.cancel')}
					</button>
				{:else}
					<span class="font-medium">
						{$_('credits.field.paymentSum', { values: { amount: paymentLabel } })}
					</span>
					{#if !manual}
						<button type="button" class="btn-ghost text-sm" onclick={startPaymentEdit}>
							{$_('common.edit')}
						</button>
					{/if}
				{/if}
			</div>

			{#if productType === 'credit'}
				<div class="space-y-1">
					<div class="flex items-center justify-between gap-4">
						<div>
							<p class="text-sm">{$_('credits.field.principalAffectsBalance')}</p>
							<FieldHint text={$_('credits.field.principalAffectsBalanceHint')} />
						</div>
						<ToggleSwitch
							checked={principalAffectsBalance}
							disabled={incomeBlocked}
							label={$_('credits.field.principalAffectsBalance')}
							onchange={() => (principalAffectsBalance = !principalAffectsBalance)}
						/>
					</div>
					{#if incomeBlocked}
						<FieldHint text={$_('credits.field.principalAffectsBalancePastPaymentBlocked')} />
					{/if}
				</div>
			{/if}

			{#if showFirstPayment}
				<div class="flex items-center justify-between gap-4">
					<div>
						<p class="text-sm">{$_('credits.field.firstPaymentToday')}</p>
						<FieldHint text={$_('credits.field.firstPaymentTodayHint')} />
					</div>
					<ToggleSwitch
						checked={firstPaymentToday}
						label={$_('credits.field.firstPaymentToday')}
						onchange={() => {
							firstPaymentToday = !firstPaymentToday;
							patchCreditCreate({ firstPaymentToday, lastScheduleKey: '' });
							void refreshCreditCreateSchedule(tz).then(syncDerived);
						}}
					/>
				</div>
			{/if}

			{#if showRetroactive}
				<div class="space-y-1">
					<div class="flex items-center justify-between gap-4">
						<div>
							<p class="text-sm">{$_('credits.field.retroactive')}</p>
							<FieldHint text={$_('credits.field.retroactiveHint')} />
						</div>
						<ToggleSwitch
							checked={retroactive}
							label={$_('credits.field.retroactive')}
							onchange={() => (retroactive = !retroactive)}
						/>
					</div>
				</div>
			{/if}

			<Select label={$_('credits.field.bank')} bind:value={bankId} options={bankOptions} />

			<div class="flex items-center justify-between gap-4">
				<div>
					<p class="text-sm">{$_('credits.field.createTransactions')}</p>
					<FieldHint text={$_('credits.field.createTransactionsHint')} />
				</div>
				<ToggleSwitch
					checked={createTransactions}
					label={$_('credits.field.createTransactions')}
					onchange={() => (createTransactions = !createTransactions)}
				/>
			</div>

			{#if createTransactions}
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('credits.field.debitTime')}</span
					>
					<input type="time" class="input w-full" bind:value={debitTimeLocal} />
					<FieldHint text={$_('credits.field.debitTimeHint')} />
				</label>
			{/if}
		</div>

		{#snippet footer()}
			<button
				type="button"
				class="btn-ghost"
				onclick={() => {
					persist();
					prevCreditCreateStep('options', fromRaw, returnTo);
				}}
			>
				{$_('common.back')}
			</button>
			<button type="button" class="btn-primary" onclick={goNext}>{$_('common.next')}</button>
		{/snippet}
	</FormPageShell>
{/if}
