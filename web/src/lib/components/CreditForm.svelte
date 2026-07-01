<script lang="ts">
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import {
		ApiError,
		createCredit,
		listAccounts,
		listBanks,
		previewCreditSchedule,
		type Account,
		type Bank
	} from '$lib/api/client';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker, defaultAutoDebitTimeLocal } from '$lib/datetime-picker-standards';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import TransactionPagination from '$lib/components/TransactionPagination.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { toast } from '$lib/toast';
	import {
		fromDatetimeLocalValue,
		dateOnlyLocalValue,
		isFutureDatetimeLocal,
		todayDateLocal,
		toDatetimeLocalValue
	} from '$lib/dates';
	import { fromCents, toAPIAmount, toCents, formatMoneyForInput } from '$lib/money';
	import { user } from '$lib/stores/auth';

	type Props = {
		open: boolean;
		onclose: () => void;
		onsaved: () => void;
	};

	type Interval = 'month' | 'week' | 'two_weeks' | 'manual';
	type ScheduleRow = { date: string; amount: string };

	let { open = $bindable(), onclose, onsaved }: Props = $props();

	let productType = $state<'credit' | 'installment' | 'mortgage'>('credit');
	let name = $state('');
	let principal = $state('');
	let propertyPrice = $state('');
	let downPayment = $state('');
	let downPaymentAffectsBalance = $state(false);
	let downPaymentAccountId = $state('');
	let issueDateLocal = $state('');
	let termMonths = $state('12');
	let interestRate = $state('12');
	let interval = $state<Interval>('month');
	let calculatedPayment = $state('');
	/** Сохранённая пользователем сумма платежа (после «Сохранить»). */
	let paymentOverride = $state<string | null>(null);
	let editingPayment = $state(false);
	let paymentDraft = $state('');
	let debitAccountId = $state('');
	let createTransactions = $state(true);
	let retroactive = $state(false);
	let retroactiveDebitCount = $state(0);
	let principalAffectsBalance = $state(false);
	let accounts = $state<Account[]>([]);
	let banks = $state<Bank[]>([]);
	let bankId = $state('');
	let debitTimeLocal = $state('');
	let scheduleRows = $state<ScheduleRow[]>([]);
	let scheduleLoading = $state(false);
	let scheduleError = $state('');
	let lastScheduleKey = $state('');
	let saving = $state(false);
	let error = $state('');
	let schedulePage = $state(1);

	const schedulePageSize = 10;

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const principalIncomeBlocked = $derived.by(() => {
		if (scheduleRows.length === 0) return false;
		const todayDay = todayDateLocal(tz).slice(0, 10);
		return scheduleRows.some((row) => {
			if (!row.date.trim()) return false;
			return dateOnlyLocalValue(row.date).slice(0, 10) < todayDay;
		});
	});
	const isManualInterval = $derived(interval === 'manual');
	const hasDownPayment = $derived.by(() => {
		if (!downPayment.trim()) return false;
		try {
			return toCents(downPayment) > 0;
		} catch {
			return false;
		}
	});
	const effectivePrincipal = $derived.by(() => {
		if (productType !== 'mortgage') return principal;
		if (!propertyPrice.trim()) return '';
		try {
			const property = toCents(propertyPrice);
			const down = downPayment.trim() ? toCents(downPayment) : 0;
			if (down >= property) return '';
			return fromCents(property - down);
		} catch {
			return '';
		}
	});
	const termCount = $derived(Math.max(1, Number(termMonths) || 1));
	const accountOptions = $derived(accounts.map((acc) => ({ value: acc.id, label: acc.name })));
	const bankOptions = $derived([
		{ value: '', label: $_('credits.field.bankNotSelected') },
		...banks.map((bank) => ({ value: bank.id, label: bank.name }))
	]);
	const intervalOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: 'month', label: tr('credits.interval.month') },
			{ value: 'week', label: tr('credits.interval.week') },
			{ value: 'two_weeks', label: tr('credits.interval.two_weeks') },
			{ value: 'manual', label: tr('credits.interval.manual') }
		];
	});

	function averageFromScheduleRows(rows: ScheduleRow[]): string {
		const cents: number[] = [];
		for (const r of rows) {
			if (!r.amount.trim()) continue;
			try {
				cents.push(toCents(r.amount));
			} catch {
				/* skip invalid */
			}
		}
		if (cents.length > 0) {
			const sum = cents.reduce((a, b) => a + b, 0);
			return fromCents(Math.round(sum / cents.length));
		}
		if (!effectivePrincipal.trim()) return '—';
		try {
			return fromCents(Math.floor(toCents(effectivePrincipal) / termCount));
		} catch {
			return '—';
		}
	}

	const displayedPayment = $derived(
		isManualInterval
			? averageFromScheduleRows(scheduleRows)
			: (paymentOverride ?? calculatedPayment) || '—'
	);

	function baseScheduleParamsKey(): string {
		return [
			effectivePrincipal,
			termMonths,
			issueDateLocal,
			interval,
			productType,
			interestRate
		].join('|');
	}

	function scheduleParamsKey(): string {
		return [baseScheduleParamsKey(), paymentOverride ?? ''].join('|');
	}

	function paymentsClose(a: string, b: string, toleranceCents = 100): boolean {
		try {
			return Math.abs(toCents(a) - toCents(b)) <= toleranceCents;
		} catch {
			return false;
		}
	}

	async function applyPaymentEdit() {
		const draft = paymentDraft.trim();
		if (!draft) {
			paymentOverride = null;
		} else if (calculatedPayment && paymentsClose(draft, calculatedPayment)) {
			paymentOverride = null;
		} else {
			paymentOverride = draft;
		}
		editingPayment = false;
		lastScheduleKey = '';
		await refreshSchedule(scheduleParamsKey());
	}

	function startPaymentEdit() {
		paymentDraft = formatMoneyForInput(paymentOverride ?? calculatedPayment);
		editingPayment = true;
	}

	function cancelPaymentEdit() {
		editingPayment = false;
	}

	function resetForm() {
		productType = 'credit';
		name = '';
		principal = '';
		propertyPrice = '';
		downPayment = '';
		downPaymentAffectsBalance = false;
		termMonths = '12';
		interestRate = '12';
		interval = 'month';
		calculatedPayment = '';
		paymentOverride = null;
		editingPayment = false;
		paymentDraft = '';
		retroactive = false;
		retroactiveDebitCount = 0;
		principalAffectsBalance = false;
		downPaymentAccountId = '';
		bankId = '';
		debitTimeLocal = '';
		banks = [];
		scheduleRows = [];
		lastScheduleKey = '';
		lastBaseScheduleKey = '';
		scheduleError = '';
		error = '';
		schedulePage = 1;
	}

	$effect(() => {
		if (principalIncomeBlocked) principalAffectsBalance = false;
	});

	$effect(() => {
		if (productType !== 'credit') principalAffectsBalance = false;
	});

	let formWasOpen = $state(false);

	$effect(() => {
		if (!open) {
			formWasOpen = false;
			return;
		}
		if (formWasOpen) return;
		formWasOpen = true;
		resetForm();
		createTransactions = true;
		void loadAccounts();
		void loadBanks();
		issueDateLocal = todayDateLocal(tz);
	});

	let lastBaseScheduleKey = $state('');

	$effect(() => {
		if (!open) return;
		const base = baseScheduleParamsKey();
		if (lastBaseScheduleKey && base !== lastBaseScheduleKey) {
			paymentOverride = null;
			lastScheduleKey = '';
		}
		lastBaseScheduleKey = base;
	});

	$effect(() => {
		if (!isManualInterval) return;
		const n = termCount;
		if (scheduleRows.length === n) return;
		if (scheduleRows.length < n) {
			const next = [...scheduleRows];
			while (next.length < n) {
				next.push({ date: '', amount: '' });
			}
			scheduleRows = next;
		} else {
			scheduleRows = scheduleRows.slice(0, n);
		}
	});

	$effect(() => {
		if (!isManualInterval || !effectivePrincipal.trim() || scheduleRows.length !== termCount)
			return;
		if (!scheduleRows.every((r) => !r.amount.trim())) return;
		let total: number;
		try {
			total = toCents(effectivePrincipal);
		} catch {
			return;
		}
		const n = termCount;
		const base = Math.floor(total / n);
		const lastAmt = total - base * (n - 1);
		scheduleRows = scheduleRows.map((r, i) => ({
			...r,
			amount: fromCents(i === n - 1 ? lastAmt : base)
		}));
	});

	$effect(() => {
		if (!open || isManualInterval || editingPayment) return;
		if (!effectivePrincipal.trim() || !termMonths) return;
		const key = scheduleParamsKey();
		if (key === lastScheduleKey) return;

		const timer = setTimeout(() => {
			void refreshSchedule(key);
		}, 300);
		return () => clearTimeout(timer);
	});

	async function loadAccounts() {
		try {
			accounts = (await listAccounts()).filter((a) => a.status === 'active');
			debitAccountId = defaultAccountId(accounts, debitAccountId);
			downPaymentAccountId = defaultAccountId(accounts, downPaymentAccountId || debitAccountId);
		} catch {
			accounts = [];
		}
	}

	async function loadBanks() {
		try {
			banks = await listBanks();
		} catch {
			banks = [];
		}
	}

	async function refreshSchedule(expectedKey: string) {
		if (!effectivePrincipal.trim() || !termMonths || isManualInterval) return;
		scheduleLoading = true;
		scheduleError = '';
		try {
			const res = await previewCreditSchedule({
				principal: toAPIAmount(effectivePrincipal),
				term: Number(termMonths),
				interest_rate: productType === 'installment' ? 0 : Number(interestRate) || 0,
				payment_interval: interval,
				issue_date: fromDatetimeLocalValue(issueDateLocal, tz),
				credit_kind: productType === 'mortgage' ? 'mortgage' : 'consumer',
				monthly_payment: paymentOverride ? toAPIAmount(paymentOverride) : null
			});
			if (scheduleParamsKey() !== expectedKey) return;
			scheduleRows = (res.schedule_preview ?? []).map((row) => ({
				date: dateOnlyLocalValue(toDatetimeLocalValue(row.payment_date, tz)),
				amount: formatMoneyForInput(row.amount_display ?? fromCents(row.amount))
			}));
			calculatedPayment = res.calculated_monthly_payment_display;
			lastScheduleKey = expectedKey;
			schedulePage = 1;
		} catch (e) {
			if (scheduleParamsKey() !== expectedKey) return;
			scheduleError = e instanceof ApiError ? e.message : 'Не удалось рассчитать график';
			lastScheduleKey = '';
		} finally {
			scheduleLoading = false;
		}
	}

	function buildSchedulePayload() {
		if (!isManualInterval) {
			return [];
		}
		return scheduleRows
			.filter((r) => r.date && r.amount)
			.map((r) => ({
				payment_date: fromDatetimeLocalValue(r.date, tz),
				amount: toAPIAmount(r.amount)
			}));
	}

	function scheduleRowsComplete(): boolean {
		return scheduleRows.length > 0 && scheduleRows.every((r) => r.date.trim() && r.amount.trim());
	}

	function retroRowIndices(): number[] {
		const out: number[] = [];
		for (let i = 0; i < scheduleRows.length; i++) {
			if (rowStatus(scheduleRows[i]) === 'retroactive') out.push(i);
		}
		return out;
	}

	function isRetroDebited(rowIndex: number): boolean {
		const indices = retroRowIndices();
		const pos = indices.indexOf(rowIndex);
		if (pos < 0) return false;
		return pos >= indices.length - retroactiveDebitCount;
	}

	function canToggleRetroDebit(rowIndex: number): boolean {
		const indices = retroRowIndices();
		const pos = indices.indexOf(rowIndex);
		if (pos < 0) return false;
		const len = indices.length;
		const n = retroactiveDebitCount;
		if (n > 0 && pos === len - n) return true;
		if (n < len && pos === len - n - 1) return true;
		return false;
	}

	function toggleRetroDebit(rowIndex: number) {
		const indices = retroRowIndices();
		const pos = indices.indexOf(rowIndex);
		if (pos < 0 || !canToggleRetroDebit(rowIndex)) return;
		const len = indices.length;
		const n = retroactiveDebitCount;
		if (n > 0 && pos === len - n) {
			retroactiveDebitCount--;
		} else if (n < len && pos === len - n - 1) {
			retroactiveDebitCount++;
		}
	}

	function rowStatus(row: ScheduleRow): 'retroactive' | 'pending' | null {
		if (!row.date.trim()) return null;
		if (!retroactive) return 'pending';
		return isFutureDatetimeLocal(row.date, tz) ? 'pending' : 'retroactive';
	}

	$effect(() => {
		if (retroactive) return;
		retroactiveDebitCount = 0;
	});

	$effect(() => {
		if (!retroactive) return;
		const len = retroRowIndices().length;
		if (retroactiveDebitCount > len) retroactiveDebitCount = len;
	});

	const showCreateTransactions = $derived(true);
	const showDebitTimeField = $derived(showCreateTransactions && createTransactions);
	const scheduleTotalPages = $derived(
		Math.max(1, Math.ceil(scheduleRows.length / schedulePageSize))
	);
	const schedulePageSafe = $derived(Math.min(Math.max(1, schedulePage), scheduleTotalPages));
	const visibleScheduleRows = $derived(
		scheduleRows
			.slice((schedulePageSafe - 1) * schedulePageSize, schedulePageSafe * schedulePageSize)
			.map((row, offset) => ({ row, index: (schedulePageSafe - 1) * schedulePageSize + offset }))
	);

	$effect(() => {
		if (schedulePage > scheduleTotalPages) {
			schedulePage = scheduleTotalPages;
		}
	});

	$effect(() => {
		if (!showDebitTimeField) {
			debitTimeLocal = '';
		}
	});

	$effect(() => {
		if (showDebitTimeField && !debitTimeLocal.trim()) {
			debitTimeLocal = defaultAutoDebitTimeLocal;
		}
	});

	$effect(() => {
		if (!downPaymentAffectsBalance) return;
		if (!downPaymentAccountId) {
			downPaymentAccountId = debitAccountId;
		}
	});

	async function submit() {
		error = '';
		if (!debitAccountId) {
			error = $_('credits.error.noAccount');
			return;
		}
		if (!effectivePrincipal.trim()) {
			error = $_('credits.error.mortgageFields');
			return;
		}
		if (!scheduleRowsComplete()) {
			error = isManualInterval
				? $_('credits.error.manualIncomplete')
				: $_('credits.schedule.empty');
			return;
		}
		saving = true;
		try {
			const seed = buildSchedulePayload();
			await createCredit({
				name: name.trim() || null,
				credit_kind: productType === 'mortgage' ? 'mortgage' : 'consumer',
				principal_amount: toAPIAmount(effectivePrincipal),
				property_price: productType === 'mortgage' ? toAPIAmount(propertyPrice) : null,
				down_payment: productType === 'mortgage' ? toAPIAmount(downPayment || '0') : null,
				down_payment_affects_balance:
					productType === 'mortgage' ? downPaymentAffectsBalance : false,
				down_payment_account_id:
					productType === 'mortgage' && downPaymentAffectsBalance
						? downPaymentAccountId || debitAccountId
						: null,
				issue_date: fromDatetimeLocalValue(issueDateLocal, tz),
				term_months: Number(termMonths),
				interest_rate: productType === 'installment' ? 0 : Number(interestRate) || 0,
				payment_interval: interval,
				paid_amount: '0',
				monthly_payment: !isManualInterval && paymentOverride ? toAPIAmount(paymentOverride) : null,
				debit_account_id: debitAccountId,
				debit_time_local: showDebitTimeField
					? debitTimeLocal.trim() || defaultAutoDebitTimeLocal
					: null,
				bank_id: bankId || null,
				added_retroactively: retroactive,
				retroactive_debit_count: retroactive ? retroactiveDebitCount : 0,
				principal_affects_balance: productType === 'credit' ? principalAffectsBalance : false,
				create_transactions: createTransactions,
				schedule_seed: seed
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
</script>

<ModalShell bind:open title={$_('credits.new')} maxWidth="max-w-2xl" {onclose}>
	<div class="space-y-4">
		<label class="block space-y-1">
			<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.name')}</span>
			<input type="text" class="input w-full" bind:value={name} />
		</label>

		<fieldset class="flex flex-wrap gap-4">
			<legend class="sr-only">{$_('credits.field.productType')}</legend>
			<label class="flex items-center gap-2">
				<input
					type="radio"
					name="productType"
					value="credit"
					checked={productType === 'credit'}
					onchange={() => (productType = 'credit')}
				/>
				{$_('credits.field.credit')}
			</label>
			<label class="flex items-center gap-2">
				<input
					type="radio"
					name="productType"
					value="installment"
					checked={productType === 'installment'}
					onchange={() => (productType = 'installment')}
				/>
				{$_('credits.field.installment')}
			</label>
			<label class="flex items-center gap-2">
				<input
					type="radio"
					name="productType"
					value="mortgage"
					checked={productType === 'mortgage'}
					onchange={() => {
						productType = 'mortgage';
						interval = 'month';
					}}
				/>
				{$_('credits.field.mortgage')}
			</label>
		</fieldset>

		<div class="grid gap-4 sm:grid-cols-2">
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
					<input class="input w-full" value={effectivePrincipal} readonly />
				</label>
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.rate')}</span>
					<input type="number" step="0.1" min="0" class="input w-full" bind:value={interestRate} />
				</label>
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
				usePortal
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
					usePortal
					onchange={() => {
						lastScheduleKey = '';
					}}
				/>
			{/if}
		</div>
		{#if productType === 'mortgage' && hasDownPayment}
			<div class="space-y-1">
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
					<div class="mt-2">
						<Select
							label={$_('credits.field.downPaymentAccount')}
							bind:value={downPaymentAccountId}
							options={accountOptions}
							usePortal
						/>
					</div>
				{/if}
			</div>
		{/if}

		<div class="flex flex-wrap items-center gap-2 text-sm">
			{#if !isManualInterval && editingPayment}
				<MoneyInput bind:value={paymentDraft} class="input w-40 tabular-nums" autoFocus />
				<button type="button" class="btn-ghost text-sm" onclick={() => void applyPaymentEdit()}>
					{$_('common.save')}
				</button>
				<button type="button" class="btn-ghost text-sm" onclick={cancelPaymentEdit}>
					{$_('common.cancel')}
				</button>
			{:else}
				<span class="font-medium">
					{$_('credits.field.paymentSum', { values: { amount: displayedPayment } })}
				</span>
				{#if !isManualInterval}
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
						disabled={principalIncomeBlocked}
						label={$_('credits.field.principalAffectsBalance')}
						onchange={() => (principalAffectsBalance = !principalAffectsBalance)}
					/>
				</div>
				{#if principalIncomeBlocked}
					<FieldHint text={$_('credits.field.principalAffectsBalancePastPaymentBlocked')} />
				{/if}
			</div>
		{/if}

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
			{#if retroactive && retroRowIndices().length > 0}
				<FieldHint text={$_('credits.field.retroactiveDebitHint')} />
			{/if}
		</div>

		<div class="space-y-2">
			<div class="flex items-center gap-2">
				<p class="text-sm font-medium">{$_('credits.schedule.title')}</p>
				{#if scheduleLoading}
					<span class="text-xs" style:color="var(--text-muted)">
						{$_('credits.schedule.loading')}
					</span>
				{/if}
			</div>
			{#if scheduleError}
				<p class="text-sm text-red-600">{scheduleError}</p>
			{/if}
			<div class="rounded border text-sm" style:border-color="var(--border)">
				{#if scheduleRows.length === 0}
					<p class="p-4 text-sm" style:color="var(--text-muted)">
						{scheduleLoading ? $_('credits.schedule.loading') : $_('credits.schedule.empty')}
					</p>
				{:else}
					<div
						class="hidden md:block overflow-x-auto transition-opacity duration-150"
						class:opacity-50={scheduleLoading}
					>
						<table class="w-full border-separate border-spacing-0">
							<thead>
								<tr>
									<th
										class="sticky top-0 z-10 border-b p-2 text-left font-medium"
										style:color="var(--text-muted)"
										style:background-color="var(--bg-elevated)"
										style:border-color="var(--border)"
									>
										#
									</th>
									<th
										class="sticky top-0 z-10 border-b p-2 text-left font-medium"
										style:color="var(--text-muted)"
										style:background-color="var(--bg-elevated)"
										style:border-color="var(--border)"
									>
										{$_('transactions.col.date')}
									</th>
									<th
										class="sticky top-0 z-10 border-b p-2 text-left font-medium"
										style:color="var(--text-muted)"
										style:background-color="var(--bg-elevated)"
										style:border-color="var(--border)"
									>
										{$_('transactions.col.amount')}
									</th>
									<th
										class="sticky top-0 z-10 border-b p-2 text-left font-medium"
										style:color="var(--text-muted)"
										style:background-color="var(--bg-elevated)"
										style:border-color="var(--border)"
									>
										{$_('credits.field.retroactiveDebit')}
									</th>
									<th
										class="sticky top-0 z-10 border-b p-2 text-left font-medium"
										style:color="var(--text-muted)"
										style:background-color="var(--bg-elevated)"
										style:border-color="var(--border)"
									>
										{$_('transactions.col.status')}
									</th>
								</tr>
							</thead>
							<tbody>
								{#each visibleScheduleRows as item (item.index)}
									{@const row = item.row}
									{@const i = item.index}
									{@const status = rowStatus(row)}
									<tr class="border-b last:border-b-0" style:border-color="var(--border)">
										<td class="p-2 align-middle" style:color="var(--text-muted)">{i + 1}</td>
										<td class="p-2 align-middle">
											<DateTimePicker
												id={`credit-schedule-${i}`}
												bind:value={row.date}
												{...dateOnlyPicker}
												usePortal
											/>
										</td>
										<td class="p-2 align-middle">
											<MoneyInput bind:value={row.amount} />
										</td>
										<td class="p-2 align-middle whitespace-nowrap">
											{#if status === 'retroactive'}
												<ToggleSwitch
													checked={isRetroDebited(i)}
													disabled={!canToggleRetroDebit(i)}
													label={$_('credits.field.retroactiveDebit')}
													onchange={() => toggleRetroDebit(i)}
												/>
											{/if}
										</td>
										<td class="p-2 align-middle whitespace-nowrap">
											{#if status === 'retroactive'}
												<span class="badge">{$_('credits.payment.status.retroactive')}</span>
											{:else if status === 'pending'}
												<span style:color="var(--text-muted)">
													{$_('credits.payment.status.pending')}
												</span>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					<div
						class="space-y-3 p-3 md:hidden transition-opacity duration-150"
						class:opacity-50={scheduleLoading}
					>
						{#each visibleScheduleRows as item (item.index)}
							{@const row = item.row}
							{@const i = item.index}
							{@const status = rowStatus(row)}
							<article class="rounded-xl border p-3" style:border-color="var(--border)">
								<p class="mb-2 text-xs font-medium" style:color="var(--text-muted)">
									{$_('credits.schedule.paymentNumber', { values: { n: i + 1 } })}
								</p>
								<dl class="grid gap-3 text-sm">
									<div class="space-y-1">
										<dt style:color="var(--text-muted)">{$_('transactions.col.date')}</dt>
										<dd>
											<DateTimePicker
												id={`credit-schedule-mobile-${i}`}
												bind:value={row.date}
												{...dateOnlyPicker}
												usePortal
											/>
										</dd>
									</div>
									<div class="space-y-1">
										<dt style:color="var(--text-muted)">{$_('transactions.col.amount')}</dt>
										<dd>
											<MoneyInput bind:value={row.amount} />
										</dd>
									</div>
									{#if status === 'retroactive'}
										<div class="flex justify-between gap-2">
											<dt style:color="var(--text-muted)">
												{$_('credits.field.retroactiveDebit')}
											</dt>
											<dd>
												<ToggleSwitch
													checked={isRetroDebited(i)}
													disabled={!canToggleRetroDebit(i)}
													label={$_('credits.field.retroactiveDebit')}
													onchange={() => toggleRetroDebit(i)}
												/>
											</dd>
										</div>
									{/if}
									{#if status}
										<div class="flex justify-between gap-2">
											<dt style:color="var(--text-muted)">{$_('transactions.col.status')}</dt>
											<dd>
												{#if status === 'retroactive'}
													<span class="badge">{$_('credits.payment.status.retroactive')}</span>
												{:else}
													<span style:color="var(--text-muted)">
														{$_('credits.payment.status.pending')}
													</span>
												{/if}
											</dd>
										</div>
									{/if}
								</dl>
							</article>
						{/each}
					</div>
					<div class="border-t" style:border-color="var(--border)">
						<TransactionPagination
							page={schedulePageSafe}
							limit={schedulePageSize}
							total={scheduleRows.length}
							onchange={(p) => (schedulePage = p)}
							class="px-3 py-2 text-sm"
						/>
					</div>
				{/if}
			</div>
		</div>

		<div class="grid gap-4 sm:grid-cols-2">
			<Select
				label={$_('credits.field.debitAccount')}
				bind:value={debitAccountId}
				options={accountOptions}
				usePortal
			/>
			<Select
				label={$_('credits.field.bank')}
				bind:value={bankId}
				options={bankOptions}
				usePortal
			/>
		</div>
		{#if showCreateTransactions}
			<div class="space-y-1">
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
			</div>
		{/if}

		{#if showDebitTimeField}
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.debitTime')}</span>
				<input type="time" class="input w-full" bind:value={debitTimeLocal} />
				<FieldHint text={$_('credits.field.debitTimeHint')} />
			</label>
		{/if}

		<FormFeedback {error} />
	</div>
	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={onclose}>{$_('common.cancel')}</button>
		<button type="button" class="btn-primary" disabled={saving} onclick={() => void submit()}>
			{$_('common.save')}
		</button>
	{/snippet}
</ModalShell>
