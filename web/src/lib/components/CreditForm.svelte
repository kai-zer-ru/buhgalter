<script lang="ts">
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import {
		ApiError,
		createCredit,
		listAccounts,
		previewCreditSchedule,
		type Account
	} from '$lib/api/client';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { toast } from '$lib/toast';
	import {
		fromDatetimeLocalValue,
		dateOnlyLocalValue,
		isFutureDatetimeLocal,
		todayDateLocal,
		toDatetimeLocalValue
	} from '$lib/dates';
	import { fromCents, toAPIAmount, toCents } from '$lib/money';
	import { user } from '$lib/stores/auth';

	type Props = {
		open: boolean;
		onclose: () => void;
		onsaved: () => void;
	};

	type Interval = 'month' | 'week' | 'two_weeks' | 'manual';
	type ScheduleRow = { date: string; amount: string };

	let { open = $bindable(), onclose, onsaved }: Props = $props();

	let productType = $state<'credit' | 'installment'>('credit');
	let name = $state('');
	let principal = $state('');
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
	let accounts = $state<Account[]>([]);
	let scheduleRows = $state<ScheduleRow[]>([]);
	let scheduleLoading = $state(false);
	let lastScheduleKey = $state('');
	let saving = $state(false);
	let error = $state('');

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const isManualInterval = $derived(interval === 'manual');
	const termCount = $derived(Math.max(1, Number(termMonths) || 1));
	const accountOptions = $derived(accounts.map((acc) => ({ value: acc.id, label: acc.name })));
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
		if (!principal.trim()) return '—';
		try {
			return fromCents(Math.floor(toCents(principal) / termCount));
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
		return [principal, termMonths, issueDateLocal, interval, productType, interestRate].join('|');
	}

	function scheduleParamsKey(): string {
		return [baseScheduleParamsKey(), paymentOverride ?? ''].join('|');
	}

	async function applyPaymentEdit() {
		paymentOverride = paymentDraft.trim() ? paymentDraft : null;
		editingPayment = false;
		lastScheduleKey = '';
		await refreshSchedule(scheduleParamsKey());
	}

	function startPaymentEdit() {
		paymentDraft = paymentOverride ?? calculatedPayment;
		editingPayment = true;
	}

	function cancelPaymentEdit() {
		editingPayment = false;
	}

	$effect(() => {
		if (open) {
			lastScheduleKey = '';
			lastBaseScheduleKey = '';
			scheduleRows = [];
			paymentOverride = null;
			editingPayment = false;
			paymentDraft = '';
			createTransactions = true;
			void loadAccounts();
			issueDateLocal = todayDateLocal(tz);
		}
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
		if (!isManualInterval || !principal.trim() || scheduleRows.length !== termCount) return;
		if (!scheduleRows.every((r) => !r.amount.trim())) return;
		let total: number;
		try {
			total = toCents(principal);
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
		if (!principal.trim() || !termMonths) return;
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
		} catch {
			accounts = [];
		}
	}

	async function refreshSchedule(expectedKey: string) {
		if (!principal.trim() || !termMonths || isManualInterval) return;
		const hadRows = scheduleRows.length > 0;
		scheduleLoading = true;
		try {
			const res = await previewCreditSchedule({
				principal: toAPIAmount(principal),
				term: Number(termMonths),
				interest_rate: productType === 'installment' ? 0 : Number(interestRate) || 0,
				payment_interval: interval,
				issue_date: fromDatetimeLocalValue(issueDateLocal, tz),
				monthly_payment: paymentOverride ? toAPIAmount(paymentOverride) : null
			});
			if (scheduleParamsKey() !== expectedKey) return;
			const nextRows = res.schedule_preview.map((row) => ({
				date: dateOnlyLocalValue(toDatetimeLocalValue(row.payment_date, tz)),
				amount: row.amount_display ?? fromCents(row.amount)
			}));
			if (nextRows.length === scheduleRows.length && scheduleRows.length > 0) {
				scheduleRows = scheduleRows.map((row, i) => ({
					date: nextRows[i].date,
					amount: nextRows[i].amount
				}));
			} else {
				scheduleRows = nextRows;
			}
			calculatedPayment = res.calculated_monthly_payment_display;
			lastScheduleKey = expectedKey;
		} catch {
			if (scheduleParamsKey() === expectedKey && !hadRows) {
				scheduleRows = [];
			}
		} finally {
			scheduleLoading = false;
		}
	}

	function buildSchedulePayload() {
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

	function rowStatus(row: ScheduleRow): 'retroactive' | 'pending' | null {
		if (!row.date.trim()) return null;
		if (!retroactive) return 'pending';
		return isFutureDatetimeLocal(row.date, tz) ? 'pending' : 'retroactive';
	}

	const hasUnpaidPayments = $derived(
		scheduleRows.some((row) => row.date.trim() && rowStatus(row) === 'pending')
	);

	const showCreateTransactions = $derived(!retroactive && hasUnpaidPayments);

	$effect(() => {
		if (retroactive) createTransactions = false;
	});

	async function submit() {
		error = '';
		if (!debitAccountId) {
			error = $_('credits.error.noAccount');
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
				principal_amount: toAPIAmount(principal),
				issue_date: fromDatetimeLocalValue(issueDateLocal, tz),
				term_months: Number(termMonths),
				interest_rate: productType === 'installment' ? 0 : Number(interestRate) || 0,
				payment_interval: interval,
				paid_amount: '0',
				monthly_payment: !isManualInterval && paymentOverride ? toAPIAmount(paymentOverride) : null,
				debit_account_id: debitAccountId,
				added_retroactively: retroactive,
				create_transactions: showCreateTransactions && createTransactions,
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
		</fieldset>

		<div class="grid gap-4 sm:grid-cols-2">
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.principal')}</span>
				<MoneyInput bind:value={principal} />
			</label>
			<DateTimePicker
				label={$_('credits.field.issueDate')}
				bind:value={issueDateLocal}
				timeMode="hidden"
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
			<Select
				label={$_('credits.field.interval')}
				bind:value={interval}
				options={intervalOptions}
				usePortal
				onchange={() => {
					lastScheduleKey = '';
				}}
			/>
		</div>

		<div class="flex flex-wrap items-center gap-2 text-sm">
			{#if !isManualInterval && editingPayment}
				<MoneyInput bind:value={paymentDraft} class="input w-40 tabular-nums" />
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

		<div class="space-y-1">
			<label class="flex items-start gap-2">
				<input type="checkbox" bind:checked={retroactive} class="mt-1" />
				<span class="text-sm">{$_('credits.field.retroactive')}</span>
			</label>
			<FieldHint text={$_('credits.field.retroactiveHint')} />
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
			<div class="overflow-x-auto rounded border text-sm" style:border-color="var(--border)">
				{#if scheduleRows.length === 0}
					<p class="p-4 text-sm" style:color="var(--text-muted)">
						{scheduleLoading ? $_('credits.schedule.loading') : $_('credits.schedule.empty')}
					</p>
				{:else}
					<table
						class="w-full border-separate border-spacing-0 transition-opacity duration-150"
						class:opacity-50={scheduleLoading}
					>
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
									{$_('transactions.col.status')}
								</th>
							</tr>
						</thead>
						<tbody>
							{#each scheduleRows as row, i (i)}
								{@const status = rowStatus(row)}
								<tr class="border-b last:border-b-0" style:border-color="var(--border)">
									<td class="p-2 align-middle" style:color="var(--text-muted)">{i + 1}</td>
									<td class="p-2 align-middle">
										<DateTimePicker
											id={`credit-schedule-${i}`}
											bind:value={row.date}
											timeMode="hidden"
											usePortal
										/>
									</td>
									<td class="p-2 align-middle">
										<MoneyInput bind:value={row.amount} />
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
				{/if}
			</div>
		</div>

		<Select
			label={$_('credits.field.debitAccount')}
			bind:value={debitAccountId}
			options={accountOptions}
			usePortal
		/>

		{#if showCreateTransactions}
			<div class="space-y-1">
				<label class="flex items-center gap-2">
					<input type="checkbox" bind:checked={createTransactions} />
					<span class="text-sm">{$_('credits.field.createTransactions')}</span>
				</label>
				<FieldHint text={$_('credits.field.createTransactionsHint')} />
			</div>
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
