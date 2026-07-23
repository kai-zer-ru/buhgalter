<script lang="ts">
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		addCreditPayment,
		getCredit,
		listAccounts,
		type Account,
		type Credit
	} from '$lib/api/client';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { leaveForm } from '$lib/android/form-nav';
	import { defaultPayAmount, defaultPayDate } from '$lib/credits/pay-helpers';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { accountSelectOptions } from '$lib/select-options';
	import { fromDatetimeLocalValue } from '$lib/dates';
	import { toAPIAmount } from '$lib/money';
	import { user } from '$lib/stores/auth';
	import { toast } from '$lib/toast';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { reportPageLoadFailure } from '$lib/page-load';

	const creditId = $derived($page.params.id ?? '');
	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), `/credits/${creditId}`)
	);
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	let credit = $state<Credit | null>(null);
	let accounts = $state<Account[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let saving = $state(false);
	let payAmount = $state('');
	let payAccountId = $state('');
	let payDateLocal = $state('');

	const accountOptions = $derived(accountSelectOptions(accounts));
	const payRemaining = $derived(() => {
		if (!credit || !payAmount) return null;
		const amt = Math.round(parseFloat(payAmount.replace(',', '.')) * 100) || 0;
		return credit.remaining_amount - amt;
	});

	$effect(() => {
		if (!creditId) return;
		void load();
	});

	async function load() {
		loading = true;
		try {
			const [c, accs] = await Promise.all([getCredit(creditId), listAccounts()]);
			credit = c;
			accounts = accs.filter((a) => a.status === 'active');
			const amountQ = $page.url.searchParams.get('amount');
			const dateQ = $page.url.searchParams.get('date');
			payAmount = amountQ ?? defaultPayAmount(c);
			payAccountId = c.debit_account_id;
			payDateLocal = dateQ ?? defaultPayDate(c, tz);
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: !!credit });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}

	async function save() {
		if (!credit) return;
		saving = true;
		try {
			await addCreditPayment(credit.id, {
				amount: toAPIAmount(payAmount),
				payment_date: fromDatetimeLocalValue(payDateLocal, tz),
				account_id: payAccountId || undefined
			});
			toast($_('common.saved'));
			finish();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
	{#if credit}
		<FormPageShell title={$_('credits.pay.title')} backHref={returnTo} onback={finish}>
			<div class="space-y-4">
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)">{$_('credits.pay.amount')}</span>
					<MoneyInput bind:value={payAmount} />
				</label>
				<Select
					id="pay-account"
					label={$_('credits.pay.account')}
					bind:value={payAccountId}
					options={accountOptions}
				/>
				<DateTimePicker
					label={$_('credits.pay.date')}
					bind:value={payDateLocal}
					{...dateOnlyPicker}
				/>
				{#if payRemaining() !== null}
					<p class="text-sm" style:color="var(--text-muted)">
						{$_('credits.pay.preview')}:
						<MoneyDisplay cents={payRemaining()!} {currency} class="" />
					</p>
				{/if}
			</div>
			{#snippet footer()}
				<button type="button" class="btn-ghost" onclick={finish}>{$_('common.cancel')}</button>
				<button type="button" class="btn-primary" disabled={saving} onclick={() => void save()}>
					{saving ? $_('common.loading') : $_('credits.action.pay')}
				</button>
			{/snippet}
		</FormPageShell>
	{/if}
</PageLoadGate>
