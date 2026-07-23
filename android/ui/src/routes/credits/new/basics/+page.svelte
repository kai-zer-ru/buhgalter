<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import { accountSelectOptions } from '$lib/select-options';
	import { user } from '$lib/stores/auth';
	import { toast } from '$lib/toast';
	import {
		creditCreateDraft,
		effectivePrincipal,
		emptyCreditCreateDraft,
		hasDownPayment,
		loadCreditCreateRefs,
		patchCreditCreate,
		validateBasics,
		type PaymentInterval,
		type ProductType
	} from '$lib/credits/create-draft';
	import {
		abandonCreditCreate,
		creditCreateReturnTo,
		ensureCreditCreateDraft,
		nextCreditCreateStep
	} from '$lib/credits/create-nav';
	import { get } from 'svelte/store';

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const fromRaw = $derived($page.url.searchParams.get('from'));
	const returnTo = $derived(creditCreateReturnTo(fromRaw));

	let ready = $state(false);
	let productType = $state<ProductType>('credit');
	let name = $state('');
	let principal = $state('');
	let propertyPrice = $state('');
	let downPayment = $state('');
	let downPaymentAffectsBalance = $state(false);
	let downPaymentAccountId = $state('');
	let issueDateLocal = $state('');
	let termMonths = $state('12');
	let interestRate = $state('12');
	let interval = $state<PaymentInterval>('month');
	let accounts = $state(get(creditCreateDraft)?.accounts ?? []);

	const accountOptions = $derived(accountSelectOptions(accounts));
	const intervalOptions = $derived([
		{ value: 'month', label: tr('credits.interval.month') },
		{ value: 'week', label: tr('credits.interval.week') },
		{ value: 'two_weeks', label: tr('credits.interval.two_weeks') },
		{ value: 'manual', label: tr('credits.interval.manual') }
	]);
	const principalComputed = $derived(
		effectivePrincipal({
			...emptyCreditCreateDraft(tz),
			productType,
			principal,
			propertyPrice,
			downPayment
		})
	);

	onMount(() => {
		ensureCreditCreateDraft(tz);
		const d = get(creditCreateDraft)!;
		productType = d.productType;
		name = d.name;
		principal = d.principal;
		propertyPrice = d.propertyPrice;
		downPayment = d.downPayment;
		downPaymentAffectsBalance = d.downPaymentAffectsBalance;
		downPaymentAccountId = d.downPaymentAccountId;
		issueDateLocal = d.issueDateLocal;
		termMonths = d.termMonths;
		interestRate = d.interestRate;
		interval = d.interval;
		accounts = d.accounts;
		ready = true;
		void loadCreditCreateRefs().then(() => {
			accounts = get(creditCreateDraft)?.accounts ?? accounts;
			if (!downPaymentAccountId) {
				downPaymentAccountId = get(creditCreateDraft)?.downPaymentAccountId ?? '';
			}
		});
	});

	function setProductType(next: ProductType) {
		productType = next;
		if (next === 'mortgage') {
			interval = 'month';
		}
	}

	function persist() {
		patchCreditCreate({
			productType,
			name,
			principal,
			propertyPrice,
			downPayment,
			downPaymentAffectsBalance,
			downPaymentAccountId,
			issueDateLocal,
			termMonths,
			interestRate,
			interval: productType === 'mortgage' ? 'month' : interval,
			firstPaymentToday:
				productType === 'mortgage' ? false : get(creditCreateDraft)?.firstPaymentToday,
			principalAffectsBalance:
				productType === 'credit'
					? (get(creditCreateDraft)?.principalAffectsBalance ?? false)
					: false,
			lastScheduleKey: ''
		});
	}

	function goNext() {
		persist();
		const d = get(creditCreateDraft);
		if (!d) return;
		const err = validateBasics(d);
		if (err) {
			toast.error($_(err));
			return;
		}
		nextCreditCreateStep('basics', fromRaw);
	}
</script>

{#if ready}
	<FormPageShell
		title={$_('credits.create.step.basics')}
		backHref={returnTo}
		onback={() => abandonCreditCreate(returnTo)}
	>
		<div class="space-y-4">
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.name')}</span>
				<input type="text" class="input w-full" bind:value={name} />
			</label>

			<fieldset class="space-y-2">
				<legend class="text-sm" style:color="var(--text-muted)"
					>{$_('credits.field.productType')}</legend
				>
				<label class="flex items-center gap-2">
					<input
						type="radio"
						name="productType"
						checked={productType === 'credit'}
						onchange={() => setProductType('credit')}
					/>
					{$_('credits.field.credit')}
				</label>
				<label class="flex items-center gap-2">
					<input
						type="radio"
						name="productType"
						checked={productType === 'installment'}
						onchange={() => setProductType('installment')}
					/>
					{$_('credits.field.installment')}
				</label>
				<label class="flex items-center gap-2">
					<input
						type="radio"
						name="productType"
						checked={productType === 'mortgage'}
						onchange={() => setProductType('mortgage')}
					/>
					{$_('credits.field.mortgage')}
				</label>
			</fieldset>

			{#if productType === 'mortgage'}
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('credits.field.propertyPrice')}</span
					>
					<MoneyInput bind:value={propertyPrice} />
				</label>
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('credits.field.downPayment')}</span
					>
					<MoneyInput bind:value={downPayment} />
				</label>
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('credits.field.principalComputed')}</span
					>
					<input class="input w-full" value={principalComputed} readonly />
				</label>
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.rate')}</span>
					<input type="number" step="0.1" min="0" class="input w-full" bind:value={interestRate} />
				</label>
				{#if hasDownPayment({ ...emptyCreditCreateDraft(tz), downPayment })}
					<div class="flex items-center justify-between gap-4">
						<div>
							<p class="text-sm">{$_('credits.field.downPaymentAffectsBalance')}</p>
							<FieldHint text={$_('credits.field.downPaymentAffectsBalanceHint')} />
						</div>
						<ToggleSwitch
							checked={downPaymentAffectsBalance}
							label={$_('credits.field.downPaymentAffectsBalance')}
							onchange={() => (downPaymentAffectsBalance = !downPaymentAffectsBalance)}
						/>
					</div>
					{#if downPaymentAffectsBalance}
						<Select
							label={$_('credits.field.downPaymentAccount')}
							bind:value={downPaymentAccountId}
							options={accountOptions}
						/>
					{/if}
				{/if}
			{:else}
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('credits.field.principal')}</span
					>
					<MoneyInput bind:value={principal} />
				</label>
			{/if}

			<DateTimePicker
				label={$_('credits.field.issueDate')}
				bind:value={issueDateLocal}
				{...dateOnlyPicker}
			/>

			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.term')}</span>
				<input type="number" min="1" class="input w-full" bind:value={termMonths} />
			</label>

			{#if productType === 'credit'}
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.rate')}</span>
					<input type="number" step="0.1" min="0" class="input w-full" bind:value={interestRate} />
				</label>
			{/if}

			{#if productType !== 'mortgage'}
				<Select
					label={$_('credits.field.interval')}
					bind:value={interval}
					options={intervalOptions}
				/>
			{/if}
		</div>

		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={() => abandonCreditCreate(returnTo)}>
				{$_('common.cancel')}
			</button>
			<button type="button" class="btn-primary" onclick={goNext}>{$_('common.next')}</button>
		{/snippet}
	</FormPageShell>
{/if}
