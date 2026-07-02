<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		addCreditPayment,
		completeCredit,
		deleteCredit,
		deleteCreditPayment,
		getCredit,
		listAccounts,
		listBanks,
		updateCredit,
		updateCreditSchedule,
		type Account,
		type Bank,
		type Credit,
		type CreditPayment
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker, defaultAutoDebitTimeLocal } from '$lib/datetime-picker-standards';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import Select from '$lib/components/Select.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import TransactionPagination from '$lib/components/TransactionPagination.svelte';
	import { toast } from '$lib/toast';
	import { confirm } from '$lib/confirm';
	import {
		formatAPIDateForDisplay,
		formatAPIOperationDateTimeForDisplay,
		formatCreditPaymentDateForDisplay,
		dateOnlyLocalValue,
		fromDatetimeLocalValue,
		todayDateLocal,
		toDatetimeLocalValue
	} from '$lib/dates';
	import { bankIconUrl } from '$lib/finance';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { formatMoneyForDisplay } from '$lib/money-display';
	import { toAPIAmount, fromCents, formatMoneyForInput } from '$lib/money';
	import { user } from '$lib/stores/auth';

	const id = $derived($page.params.id ?? '');
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	let credit = $state<Credit | null>(null);
	let accounts = $state<Account[]>([]);
	let banks = $state<Bank[]>([]);
	let loading = $state(true);
	let payOpen = $state(false);
	let payAmount = $state('');
	let payAccountId = $state('');
	let payDateLocal = $state('');
	let changeAccountOpen = $state(false);
	let newAccountId = $state('');
	let setDebitTimeOpen = $state(false);
	let debitTimeLocal = $state('');
	let autoDebitEnabled = $state(false);
	let changeNameOpen = $state(false);
	let newCreditName = $state('');
	let changeBankOpen = $state(false);
	let newBankId = $state('');
	let completeOpen = $state(false);
	let completeDateLocal = $state('');
	let completeMode = $state<'account' | 'skip'>('account');
	let paySubmitting = $state(false);
	let scheduleEditing = $state(false);
	let scheduleEditRows = $state<{ id: string; amount: string }[]>([]);
	let scheduleSaving = $state(false);
	let refreshing = $state(false);
	let pendingPage = $state(1);
	let appliedPage = $state(1);
	let retroactivePage = $state(1);
	let loadedForID = $state('');

	const schedulePageSize = 10;

	const accountOptions = $derived(accounts.map((acc) => ({ value: acc.id, label: acc.name })));
	const bankOptions = $derived([
		{ value: '', label: $_('credits.field.bankNotSelected') },
		...banks.map((bank) => ({ value: bank.id, label: bank.name }))
	]);

	$effect(() => {
		if (!id) return;
		if (loadedForID === id) return;
		loadedForID = id;
		void load();
	});

	async function load(options: { silent?: boolean } = {}) {
		const silent = options.silent ?? false;
		if (!silent || !credit) {
			loading = true;
		} else {
			refreshing = true;
		}
		try {
			const [c, accs, bankList] = await Promise.all([getCredit(id), listAccounts(), listBanks()]);
			credit = c;
			accounts = accs.filter((a) => a.status === 'active');
			banks = bankList;
			newAccountId = c.debit_account_id;
			newBankId = c.bank_id ?? '';
			debitTimeLocal = c.debit_time_local ?? '';
			pendingPage = 1;
			appliedPage = 1;
			retroactivePage = 1;
		} catch (err) {
			toast.fromError(err);
		} finally {
			if (!silent || !credit) {
				loading = false;
			}
			refreshing = false;
		}
	}

	function creditName(c: Credit): string {
		return c.name?.trim() || $_('credits.title');
	}

	function creditBankIcon(c: Credit | null): string | null {
		if (!c?.bank_id) return null;
		return banks.find((item) => item.id === c.bank_id)?.icon_path ?? null;
	}

	function totalInterestCents(c: Credit): number {
		if (c.is_installment) return 0;
		if (c.schedule?.length) {
			const sum = c.schedule.reduce((acc, p) => acc + p.amount, 0);
			return Math.max(0, sum - c.principal_amount);
		}
		return Math.max(0, c.monthly_payment * c.term_months - c.principal_amount);
	}

	function formatInterestRate(rate: number): string {
		if (!Number.isFinite(rate)) return '0';
		return String(rate);
	}

	function nextPendingPayment(c: Credit): CreditPayment | undefined {
		return c.schedule?.find((p) => !p.is_applied && p.kind === 'scheduled');
	}

	function paymentStatus(p: NonNullable<Credit['schedule']>[number]): string {
		if (p.kind === 'retroactive') return $_('credits.payment.status.retroactive');
		if (!p.is_applied) return $_('credits.payment.status.pending');
		if (p.transaction_kind === 'future') {
			return $_('credits.payment.status.future');
		}
		return $_('credits.payment.status.applied');
	}

	function paymentStatusExtra(p: NonNullable<Credit['schedule']>[number]): string {
		if (p.kind === 'retroactive' && p.transaction_id) {
			return $_('credits.payment.status.debitedFromAccount');
		}
		if (p.exclude_from_stats && p.kind !== 'retroactive') {
			return $_('credits.payment.excludeFromStats');
		}
		return '';
	}

	const payRemaining = $derived(() => {
		if (!credit || !payAmount) return null;
		const amt = Math.round(parseFloat(payAmount.replace(',', '.')) * 100) || 0;
		return credit.remaining_amount - amt;
	});

	async function submitPay() {
		if (!credit) return;
		paySubmitting = true;
		const previousCredit = credit;
		const schedule = credit.schedule ?? [];
		const optimisticIndex = schedule.findIndex((p) => !p.is_applied && p.kind === 'scheduled');
		if (optimisticIndex >= 0) {
			credit = {
				...credit,
				schedule: schedule.map((p, i) =>
					i === optimisticIndex ? { ...p, is_applied: true, transaction_kind: 'future' } : p
				)
			};
		}
		try {
			await addCreditPayment(credit.id, {
				amount: toAPIAmount(payAmount),
				payment_date: fromDatetimeLocalValue(payDateLocal, tz),
				account_id: payAccountId || undefined
			});
			payOpen = false;
			toast($_('common.saved'));
			await load({ silent: true });
		} catch (err) {
			credit = previousCredit;
			toast.fromError(err);
			await load({ silent: true });
		} finally {
			paySubmitting = false;
		}
	}

	async function submitChangeAccount() {
		if (!credit) return;
		try {
			await updateCredit(credit.id, { debit_account_id: newAccountId });
			changeAccountOpen = false;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function submitComplete() {
		if (!credit) return;
		try {
			await completeCredit(credit.id, {
				affects_balance: completeMode === 'account',
				payment_date: fromDatetimeLocalValue(completeDateLocal, tz)
			});
			completeOpen = false;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function doDelete() {
		if (!credit) return;
		const cascade = await confirm({
			title: $_('credits.delete.title'),
			message: $_('credits.confirm.delete'),
			confirmLabel: $_('credits.delete.cascade'),
			cancelLabel: $_('credits.delete.keep'),
			danger: true
		});
		const mode = cascade ? 'cascade' : 'keep_transactions';
		try {
			await deleteCredit(credit.id, mode);
			window.location.href = resolve('/credits');
		} catch (err) {
			toast.fromError(err);
		}
	}

	function defaultPayAmount(c: Credit): string {
		const next = nextPendingPayment(c);
		let cents = next?.amount ?? c.next_payment_amount ?? c.monthly_payment;
		if (cents > c.remaining_amount) {
			cents = c.remaining_amount;
		}
		return fromCents(cents);
	}

	function defaultPayDate(c: Credit): string {
		const next = nextPendingPayment(c);
		if (next) {
			return dateOnlyLocalValue(toDatetimeLocalValue(next.payment_date, tz));
		}
		return todayDateLocal(tz);
	}

	function openPay() {
		if (!credit) return;
		payAmount = defaultPayAmount(credit);
		payAccountId = credit.debit_account_id;
		payDateLocal = defaultPayDate(credit);
		payOpen = true;
	}

	function openComplete() {
		if (!credit) return;
		completeDateLocal = todayDateLocal(tz);
		completeMode = credit.remaining_amount > 0 ? 'account' : 'skip';
		completeOpen = true;
	}

	function openChangeAccount() {
		if (!credit) return;
		newAccountId = credit.debit_account_id;
		changeAccountOpen = true;
	}

	function openSetDebitTime() {
		if (!credit) return;
		debitTimeLocal = credit.debit_time_local ?? '';
		autoDebitEnabled = Boolean((credit.debit_time_local ?? '').trim());
		setDebitTimeOpen = true;
	}

	function openChangeName() {
		if (!credit) return;
		newCreditName = credit.name?.trim() || '';
		changeNameOpen = true;
	}

	async function submitChangeName() {
		if (!credit) return;
		const trimmedName = newCreditName.trim();
		if (!trimmedName) {
			toast.error($_('credits.error.nameRequired'));
			return;
		}
		try {
			await updateCredit(credit.id, { name: trimmedName });
			changeNameOpen = false;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function submitDebitTime() {
		if (!credit) return;
		try {
			const nextDebitTime = autoDebitEnabled
				? debitTimeLocal.trim() || defaultAutoDebitTimeLocal
				: null;
			await updateCredit(credit.id, { debit_time_local: nextDebitTime });
			setDebitTimeOpen = false;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	function openChangeBank() {
		if (!credit) return;
		newBankId = credit.bank_id ?? '';
		changeBankOpen = true;
	}

	async function submitChangeBank() {
		if (!credit) return;
		try {
			await updateCredit(credit.id, { bank_id: newBankId || null });
			changeBankOpen = false;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}

	function paymentDateDisplay(p: CreditPayment): string {
		return formatCreditPaymentDateForDisplay(p.payment_date, tz, credit?.debit_time_local);
	}

	function canEditPayment(p: CreditPayment): boolean {
		return (
			credit?.status === 'active' &&
			!paySubmitting &&
			!refreshing &&
			!p.is_applied &&
			p.kind === 'scheduled'
		);
	}

	function openScheduleEdit() {
		if (!credit) return;
		scheduleEditRows = (credit.schedule ?? [])
			.filter(canEditPayment)
			.map((p) => ({ id: p.id, amount: formatMoneyForInput(fromCents(p.amount)) }));
		scheduleEditing = true;
	}

	function cancelScheduleEdit() {
		scheduleEditing = false;
		scheduleEditRows = [];
	}

	async function submitScheduleEdit() {
		if (!credit) return;
		scheduleSaving = true;
		try {
			const payments = scheduleEditRows.map((row) => ({
				id: row.id,
				amount: toAPIAmount(row.amount)
			}));
			await updateCreditSchedule(credit.id, { payments });
			scheduleEditing = false;
			scheduleEditRows = [];
			toast($_('common.saved'));
			await load();
		} catch (err) {
			toast.fromError(err);
		} finally {
			scheduleSaving = false;
		}
	}

	function canDeletePayment(p: CreditPayment): boolean {
		return (
			credit?.status === 'active' &&
			!paySubmitting &&
			!refreshing &&
			p.is_applied &&
			p.kind !== 'retroactive' &&
			p.id === latestDeletableAppliedPaymentId
		);
	}

	function canPayPayment(p: CreditPayment): boolean {
		return (
			credit?.status === 'active' &&
			!paySubmitting &&
			!refreshing &&
			!p.is_applied &&
			p.kind === 'scheduled' &&
			p.id === firstPendingScheduledPaymentId
		);
	}

	function paymentRowActions(p: CreditPayment): RowAction[] {
		const actions: RowAction[] = [];
		if (canPayPayment(p)) {
			actions.push({
				icon: 'pay',
				label: $_('credits.action.pay'),
				onclick: () => openPayForPayment(p)
			});
		} else if (canDeletePayment(p)) {
			actions.push({
				icon: 'delete',
				label: $_('credits.payment.delete'),
				variant: 'danger',
				onclick: () => void doDeletePayment(p)
			});
		}
		return actions;
	}

	function creditPageActions(): RowAction[] {
		if (!credit) return [];
		if (credit.status === 'active') {
			return [
				{
					icon: 'edit',
					label: $_('credits.action.changeName'),
					onclick: () => openChangeName()
				},
				{
					icon: 'transfer',
					label: $_('credits.action.changeAccount'),
					onclick: () => openChangeAccount()
				},
				{
					icon: 'repeat',
					label: credit.debit_time_local
						? $_('credits.action.changeDebitTime')
						: $_('credits.action.setDebitTime'),
					onclick: () => openSetDebitTime()
				},
				{
					icon: 'bank',
					label: $_('credits.action.changeBank'),
					onclick: () => openChangeBank()
				},
				{
					icon: 'archive',
					label: $_('credits.action.complete'),
					onclick: () => openComplete()
				},
				{
					icon: 'delete',
					label: $_('common.delete'),
					variant: 'danger',
					onclick: () => void doDelete()
				}
			];
		}
		return [
			{
				icon: 'edit',
				label: $_('credits.action.changeName'),
				onclick: () => openChangeName()
			}
		];
	}

	function openPayForPayment(p: CreditPayment) {
		if (!credit) return;
		const amountCents = Math.min(p.amount, credit.remaining_amount);
		payAmount = fromCents(amountCents);
		payAccountId = credit.debit_account_id;
		payDateLocal = dateOnlyLocalValue(toDatetimeLocalValue(p.payment_date, tz));
		payOpen = true;
	}

	type ScheduleGroup = 'pending' | 'applied' | 'retroactive';

	function scheduleGroup(p: CreditPayment): ScheduleGroup {
		if (p.kind === 'retroactive') return 'retroactive';
		if (!p.is_applied) return 'pending';
		return 'applied';
	}

	function comparePaymentsOldestFirst(a: CreditPayment, b: CreditPayment): number {
		const byDate = a.payment_date.localeCompare(b.payment_date);
		if (byDate !== 0) return byDate;
		return a.created_at.localeCompare(b.created_at);
	}

	const scheduleGroups = $derived.by(() => {
		const empty = {
			pending: [] as CreditPayment[],
			applied: [] as CreditPayment[],
			retroactive: [] as CreditPayment[]
		};
		if (!credit?.schedule?.length) return empty;
		for (const p of credit.schedule) {
			empty[scheduleGroup(p)].push(p);
		}
		empty.pending.sort(comparePaymentsOldestFirst);
		empty.applied.sort(comparePaymentsOldestFirst);
		empty.retroactive.sort(comparePaymentsOldestFirst);
		return empty;
	});
	const firstPendingScheduledPaymentId = $derived.by(() => {
		for (const p of scheduleGroups.pending) {
			if (p.kind === 'scheduled' && !p.is_applied) return p.id;
		}
		return '';
	});
	const latestDeletableAppliedPaymentId = $derived.by(() => {
		for (let i = scheduleGroups.applied.length - 1; i >= 0; i--) {
			const p = scheduleGroups.applied[i];
			if (p.transaction_id) return p.id;
		}
		return '';
	});

	const creditIsActive = $derived(credit?.status === 'active');
	const pendingPages = $derived(
		Math.max(1, Math.ceil(scheduleGroups.pending.length / schedulePageSize))
	);
	const appliedPages = $derived(
		Math.max(1, Math.ceil(scheduleGroups.applied.length / schedulePageSize))
	);
	const retroactivePages = $derived(
		Math.max(1, Math.ceil(scheduleGroups.retroactive.length / schedulePageSize))
	);
	const pendingPageSafe = $derived(Math.min(Math.max(1, pendingPage), pendingPages));
	const appliedPageSafe = $derived(Math.min(Math.max(1, appliedPage), appliedPages));
	const retroactivePageSafe = $derived(Math.min(Math.max(1, retroactivePage), retroactivePages));

	const visiblePendingPayments = $derived(
		scheduleGroups.pending.slice(
			(pendingPageSafe - 1) * schedulePageSize,
			pendingPageSafe * schedulePageSize
		)
	);
	const visibleAppliedPayments = $derived(
		scheduleGroups.applied.slice(
			(appliedPageSafe - 1) * schedulePageSize,
			appliedPageSafe * schedulePageSize
		)
	);
	const visibleRetroactivePayments = $derived(
		scheduleGroups.retroactive.slice(
			(retroactivePageSafe - 1) * schedulePageSize,
			retroactivePageSafe * schedulePageSize
		)
	);

	async function doDeletePayment(p: CreditPayment) {
		if (!credit) return;
		const ok = await confirm({
			message: $_(
				p.is_applied ? 'credits.confirm.deleteAppliedPayment' : 'credits.confirm.deletePayment'
			),
			danger: true
		});
		if (!ok) return;
		await deleteCreditPayment(credit.id, p.id);
		toast($_('common.deleted'));
		await load();
	}
</script>

<svelte:head>
	<title>{credit ? creditName(credit) : $_('credits.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-4">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/credits', label: $_('credits.title') },
			{ href: '/credits', label: credit ? creditName(credit) : $_('credits.title') }
		]}
	/>

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if credit}
		<div class="flex flex-col gap-3 md:flex-row md:flex-wrap md:items-start md:justify-between">
			<div>
				<div class="flex items-center gap-2">
					{#if creditBankIcon(credit)}
						<img
							src={bankIconUrl(creditBankIcon(credit)!)}
							alt=""
							class="h-7 w-7 rounded-md"
							width="28"
							height="28"
						/>
					{/if}
					<h1 class="text-2xl font-semibold">{creditName(credit)}</h1>
				</div>
				<div class="mt-2 flex flex-wrap items-center gap-2">
					{#if credit.credit_kind === 'mortgage'}
						<span class="badge">{$_('credits.badge.mortgage')}</span>
					{:else if credit.is_installment}
						<span class="badge">{$_('credits.badge.installment')}</span>
					{:else}
						<span class="badge">{$_('credits.badge.credit')}</span>
					{/if}
					{#if credit.added_retroactively}
						<span class="badge">{$_('credits.badge.retroactive')}</span>
					{/if}
					{#if credit.status === 'closed'}
						<span class="badge badge-success">{$_('credits.badge.closed')}</span>
					{/if}
				</div>
			</div>
			<div class="flex w-full shrink-0 items-center justify-end gap-2 md:w-auto">
				{#if credit.status === 'active' && nextPendingPayment(credit)}
					<button type="button" class="btn-primary" onclick={openPay} disabled={paySubmitting}>
						{paySubmitting ? $_('credits.action.paying') : $_('credits.action.pay')}
					</button>
				{/if}
				<RowActionsMenu actions={creditPageActions()} />
			</div>
		</div>

		<div class="card grid gap-3 p-4 text-sm md:grid-cols-3">
			<div>
				<span style:color="var(--text-muted)">{$_('credits.field.principal')}</span>
				<p class="font-medium">
					<MoneyDisplay value={credit.principal_amount_display} {currency} class="" />
				</p>
			</div>
			{#if credit.credit_kind === 'mortgage'}
				<div>
					<span style:color="var(--text-muted)">{$_('credits.field.propertyPrice')}</span>
					<p class="font-medium">
						{#if credit.property_price_display}
							<MoneyDisplay value={credit.property_price_display} {currency} class="" />
						{:else}
							—
						{/if}
					</p>
				</div>
				<div>
					<span style:color="var(--text-muted)">{$_('credits.field.downPayment')}</span>
					<p class="font-medium">
						{#if credit.down_payment_display}
							<MoneyDisplay value={credit.down_payment_display} {currency} class="" />
						{:else}
							—
						{/if}
					</p>
				</div>
			{/if}
			{#if !credit.is_installment}
				<div>
					<span style:color="var(--text-muted)">{$_('credits.field.totalInterest')}</span>
					<p class="font-medium">
						<MoneyDisplay cents={totalInterestCents(credit)} {currency} class="" />
					</p>
				</div>
			{/if}
			<div>
				<span style:color="var(--text-muted)">{$_('credits.field.payment')}</span>
				<p class="font-medium">
					<MoneyDisplay value={credit.monthly_payment_display} {currency} class="" />
				</p>
			</div>
			{#if !credit.is_installment}
				<div>
					<span style:color="var(--text-muted)">{$_('credits.field.rate')}</span>
					<p class="font-medium">
						{$_('credits.header.rate', {
							values: { rate: formatInterestRate(credit.interest_rate) }
						})}
					</p>
				</div>
			{/if}
			<div>
				<span style:color="var(--text-muted)">{$_('credits.field.issueDate')}</span>
				<p class="font-medium">{formatAPIDateForDisplay(credit.issue_date, tz)}</p>
			</div>
			{#if credit.added_retroactively}
				<div>
					<span style:color="var(--text-muted)">{$_('credits.field.recordedAt')}</span>
					<p class="font-medium">{formatAPIOperationDateTimeForDisplay(credit.recorded_at, tz)}</p>
				</div>
			{/if}
			<div>
				<span style:color="var(--text-muted)">{$_('credits.field.debitAccount')}</span>
				<p>
					<a
						href={resolve(`/accounts/${credit.debit_account_id}`)}
						class="font-medium hover:underline"
					>
						{credit.debit_account_name}
					</a>
				</p>
			</div>
			{#if credit.debit_time_local}
				<div>
					<span style:color="var(--text-muted)">{$_('credits.field.debitTime')}</span>
					<p class="font-medium">{credit.debit_time_local}</p>
				</div>
			{/if}
			<div>
				<span style:color="var(--text-muted)">{$_('credits.field.bank')}</span>
				<p class="font-medium">{credit.bank_name || $_('credits.field.bankNotSelected')}</p>
			</div>
		</div>
		{#if refreshing}
			<p class="text-sm" style:color="var(--text-muted)">
				{$_('credits.loading.updating')}
			</p>
		{/if}

		{#if credit.schedule?.length}
			<div class="card relative overflow-hidden">
				{#if paySubmitting || refreshing}
					<div
						class="absolute inset-0 z-20 flex items-center justify-center bg-[color-mix(in_srgb,var(--bg)_72%,transparent)]"
					>
						<span class="badge">{$_('credits.loading.updating')}</span>
					</div>
				{/if}
				<h2 class="border-b px-4 py-3 text-sm font-semibold" style:border-color="var(--border)">
					{$_('credits.schedule.title')}
				</h2>

				{#snippet paymentTable(payments: CreditPayment[], editable = false)}
					<div class="hidden overflow-x-auto md:block">
						<table class="w-full text-left text-sm">
							<thead>
								<tr style:color="var(--text-muted)">
									<th class="p-3">{$_('credits.pay.date')}</th>
									<th class="p-3">{$_('transactions.col.amount')}</th>
									<th class="p-3">{$_('transactions.col.status')}</th>
									{#if creditIsActive && !editable}
										<th class="p-3"></th>
									{/if}
								</tr>
							</thead>
							<tbody>
								{#each payments as p (p.id)}
									<tr class="border-t" style:border-color="var(--border)">
										<td class="p-3">{paymentDateDisplay(p)}</td>
										<td class="p-3">
											{#if editable && canEditPayment(p)}
												{@const editIdx = scheduleEditRows.findIndex((row) => row.id === p.id)}
												{#if editIdx >= 0}
													<MoneyInput bind:value={scheduleEditRows[editIdx].amount} />
												{/if}
											{:else}
												<MoneyDisplay value={p.amount_display} {currency} class="" />
											{/if}
										</td>
										<td class="p-3">
											{paymentStatus(p)}
											{#if paymentStatusExtra(p)}
												<span class="badge ml-2">{paymentStatusExtra(p)}</span>
											{/if}
										</td>
										{#if creditIsActive && !editable}
											<td class="p-3 text-right">
												<RowActionsMenu actions={paymentRowActions(p)} />
											</td>
										{/if}
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					<div class="space-y-3 p-3 md:hidden">
						{#each payments as p (p.id)}
							{@const paymentActions = paymentRowActions(p)}
							<article class="rounded-xl border p-3" style:border-color="var(--border)">
								<dl class="grid gap-2 text-sm">
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">{$_('credits.pay.date')}</dt>
										<dd>{paymentDateDisplay(p)}</dd>
									</div>
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">{$_('transactions.col.amount')}</dt>
										<dd class="font-medium tabular-nums">
											{#if editable && canEditPayment(p)}
												{@const editIdx = scheduleEditRows.findIndex((row) => row.id === p.id)}
												{#if editIdx >= 0}
													<MoneyInput bind:value={scheduleEditRows[editIdx].amount} />
												{/if}
											{:else}
												<MoneyDisplay value={p.amount_display} {currency} class="" />
											{/if}
										</dd>
									</div>
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">{$_('transactions.col.status')}</dt>
										<dd>
											{paymentStatus(p)}
											{#if paymentStatusExtra(p)}
												<span class="badge ml-2">{paymentStatusExtra(p)}</span>
											{/if}
										</dd>
									</div>
								</dl>
								{#if creditIsActive && !editable && paymentActions.length > 0}
									<div class="mt-3 flex justify-end">
										<RowActionsMenu actions={paymentActions} />
									</div>
								{/if}
							</article>
						{/each}
					</div>
				{/snippet}

				{#if scheduleGroups.pending.length > 0}
					<details open class="border-b" style:border-color="var(--border)">
						<summary
							class="cursor-pointer list-none px-4 py-3 text-sm font-medium select-none [&::-webkit-details-marker]:hidden"
						>
							<div class="flex flex-wrap items-center justify-between gap-2">
								<span>
									{$_('credits.schedule.group.pending')}
									<span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)">
										({scheduleGroups.pending.length})
									</span>
								</span>
								{#if creditIsActive && !scheduleEditing && scheduleGroups.pending.some(canEditPayment)}
									<IconButton
										icon="edit"
										label={$_('credits.schedule.edit')}
										onclick={(e) => {
											e.preventDefault();
											openScheduleEdit();
										}}
									/>
								{/if}
							</div>
						</summary>
						{#if scheduleEditing}
							<div class="px-4 pb-2">
								<FieldHint text={$_('credits.schedule.editHint')} />
								<div class="mt-2 flex flex-wrap gap-2">
									<button
										type="button"
										class="btn-primary"
										disabled={scheduleSaving}
										onclick={() => void submitScheduleEdit()}
									>
										{$_('common.save')}
									</button>
									<button
										type="button"
										class="btn-ghost"
										disabled={scheduleSaving}
										onclick={cancelScheduleEdit}
									>
										{$_('common.cancel')}
									</button>
								</div>
							</div>
						{/if}
						{@render paymentTable(visiblePendingPayments, scheduleEditing)}
						<div class="px-4 pb-4">
							<TransactionPagination
								page={pendingPageSafe}
								limit={schedulePageSize}
								total={scheduleGroups.pending.length}
								onchange={(p) => (pendingPage = p)}
							/>
						</div>
					</details>
				{/if}

				{#if scheduleGroups.applied.length > 0}
					<details class="border-b" style:border-color="var(--border)">
						<summary
							class="cursor-pointer list-none px-4 py-3 text-sm font-medium select-none [&::-webkit-details-marker]:hidden"
						>
							{$_('credits.schedule.group.applied')}
							<span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)">
								({scheduleGroups.applied.length})
							</span>
						</summary>
						<div class="px-4 pb-3">
							<FieldHint text={$_('credits.schedule.appliedHint')} />
						</div>
						{@render paymentTable(visibleAppliedPayments)}
						<div class="px-4 pb-4">
							<TransactionPagination
								page={appliedPageSafe}
								limit={schedulePageSize}
								total={scheduleGroups.applied.length}
								onchange={(p) => (appliedPage = p)}
							/>
						</div>
					</details>
				{/if}

				{#if scheduleGroups.retroactive.length > 0}
					<details class="border-b last:border-b-0" style:border-color="var(--border)">
						<summary
							class="cursor-pointer list-none px-4 py-3 text-sm font-medium select-none [&::-webkit-details-marker]:hidden"
						>
							{$_('credits.schedule.group.retroactive')}
							<span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)">
								({scheduleGroups.retroactive.length})
							</span>
						</summary>
						<div class="px-4 pb-3">
							<FieldHint text={$_('credits.field.retroactiveHint')} />
						</div>
						{@render paymentTable(visibleRetroactivePayments)}
						<div class="px-4 pb-4">
							<TransactionPagination
								page={retroactivePageSafe}
								limit={schedulePageSize}
								total={scheduleGroups.retroactive.length}
								onchange={(p) => (retroactivePage = p)}
							/>
						</div>
					</details>
				{/if}
			</div>
		{/if}
	{/if}
</div>

{#if payOpen && credit}
	<ModalShell bind:open={payOpen} title={$_('credits.pay.title')} onclose={() => (payOpen = false)}>
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
				usePortal
			/>
			<DateTimePicker
				label={$_('credits.pay.date')}
				bind:value={payDateLocal}
				{...dateOnlyPicker}
				usePortal
			/>
			{#if payRemaining() !== null}
				<p class="text-sm" style:color="var(--text-muted)">
					{$_('credits.pay.preview')}:
					<MoneyDisplay cents={payRemaining()!} {currency} class="" />
				</p>
			{/if}
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={() => (payOpen = false)}>
				{$_('common.cancel')}
			</button>
			<button type="button" class="btn-primary" onclick={() => void submitPay()}>
				{$_('credits.action.pay')}
			</button>
		{/snippet}
	</ModalShell>
{/if}

{#if completeOpen && credit}
	{@const activeCredit = credit}
	<ModalShell
		bind:open={completeOpen}
		title={$_('credits.complete.title')}
		onclose={() => (completeOpen = false)}
	>
		<div class="space-y-4">
			{#if activeCredit.remaining_amount > 0}
				<div class="space-y-2">
					<label class="flex cursor-pointer items-start gap-2">
						<input
							type="radio"
							class="mt-1"
							name="complete-mode"
							value="account"
							bind:group={completeMode}
						/>
						<span class="text-sm">
							{$_('credits.complete.payFromAccount', {
								values: {
									amount: formatMoneyForDisplay({
										value: activeCredit.remaining_amount_display,
										currency
									}),
									account: activeCredit.debit_account_name
								}
							})}
						</span>
					</label>
					<label class="flex cursor-pointer items-start gap-2">
						<input
							type="radio"
							class="mt-1"
							name="complete-mode"
							value="skip"
							bind:group={completeMode}
						/>
						<span class="text-sm">{$_('credits.complete.skipBalance')}</span>
					</label>
					<FieldHint text={$_('credits.complete.skipBalanceHint')} />
				</div>
			{/if}
			<DateTimePicker
				label={$_('credits.complete.date')}
				bind:value={completeDateLocal}
				{...dateOnlyPicker}
				usePortal
			/>
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={() => (completeOpen = false)}>
				{$_('common.cancel')}
			</button>
			<button type="button" class="btn-primary" onclick={() => void submitComplete()}>
				{$_('credits.action.complete')}
			</button>
		{/snippet}
	</ModalShell>
{/if}

{#if changeAccountOpen && credit}
	<ModalShell
		bind:open={changeAccountOpen}
		title={$_('credits.action.changeAccount')}
		onclose={() => (changeAccountOpen = false)}
	>
		<div class="space-y-4">
			<Select
				label={$_('transactions.field.account')}
				bind:value={newAccountId}
				options={accountOptions}
				usePortal
			/>
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={() => (changeAccountOpen = false)}>
				{$_('common.cancel')}
			</button>
			<button type="button" class="btn-primary" onclick={() => void submitChangeAccount()}>
				{$_('common.save')}
			</button>
		{/snippet}
	</ModalShell>
{/if}

{#if setDebitTimeOpen && credit}
	<ModalShell
		bind:open={setDebitTimeOpen}
		title={$_('credits.modal.autoPaymentTitle')}
		onclose={() => (setDebitTimeOpen = false)}
	>
		<div class="space-y-4">
			<div class="flex items-center justify-between gap-4">
				<p class="text-sm">{$_('credits.field.autoDebit')}</p>
				<ToggleSwitch
					checked={autoDebitEnabled}
					label={$_('credits.field.autoDebit')}
					onchange={() => {
						autoDebitEnabled = !autoDebitEnabled;
						if (autoDebitEnabled && !debitTimeLocal.trim())
							debitTimeLocal = defaultAutoDebitTimeLocal;
					}}
				/>
			</div>
			{#if autoDebitEnabled}
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('credits.field.debitTime')}</span
					>
					<input type="time" class="input w-full" bind:value={debitTimeLocal} />
				</label>
				<FieldHint text={$_('credits.field.debitTimeHint')} />
			{/if}
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={() => (setDebitTimeOpen = false)}>
				{$_('common.cancel')}
			</button>
			<button type="button" class="btn-primary" onclick={() => void submitDebitTime()}>
				{$_('common.save')}
			</button>
		{/snippet}
	</ModalShell>
{/if}

{#if changeNameOpen && credit}
	<ModalShell
		bind:open={changeNameOpen}
		title={$_('credits.action.changeName')}
		onclose={() => (changeNameOpen = false)}
	>
		<div class="space-y-4">
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.name')}</span>
				<input class="input w-full" bind:value={newCreditName} maxlength="128" />
			</label>
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={() => (changeNameOpen = false)}>
				{$_('common.cancel')}
			</button>
			<button type="button" class="btn-primary" onclick={() => void submitChangeName()}>
				{$_('common.save')}
			</button>
		{/snippet}
	</ModalShell>
{/if}

{#if changeBankOpen && credit}
	<ModalShell
		bind:open={changeBankOpen}
		title={$_('credits.action.changeBank')}
		onclose={() => (changeBankOpen = false)}
	>
		<div class="space-y-4">
			<Select
				label={$_('credits.field.bank')}
				bind:value={newBankId}
				options={bankOptions}
				usePortal
			/>
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={() => (changeBankOpen = false)}>
				{$_('common.cancel')}
			</button>
			<button type="button" class="btn-primary" onclick={() => void submitChangeBank()}>
				{$_('common.save')}
			</button>
		{/snippet}
	</ModalShell>
{/if}
