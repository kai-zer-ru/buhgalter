<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		getCredit,
		listBanks,
		type Bank,
		type Credit,
		type CreditPayment
	} from '$lib/api/client';
	import {
		deleteCredit,
		deleteCreditPayment,
		updateCreditSchedule
	} from '$lib/offline/credits-api';
	import { creditActionPath } from '$lib/android/form-routes';
	import { nextPendingPayment } from '$lib/credits/pay-helpers';
	import BackLink from '$lib/components/BackLink.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
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
		toDatetimeLocalValue,
		todayDateLocal
	} from '$lib/dates';
	import { bankIconUrl } from '$lib/finance';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { fromCents, formatMoneyForInput, toAPIAmount } from '$lib/money';
	import { user } from '$lib/stores/auth';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { refCacheReadyAny, refCacheUpdate } from '$lib/offline/ref-cache';
	import { refCachePathMatches } from '$lib/offline/ref-cache-watch';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { assignIfChanged } from '$lib/state-utils';

	const id = $derived($page.params.id ?? '');
	const creditPath = $derived(`/credits/${id}`);
	const creditApiPath = $derived(`/api/v1/credits/${id}`);
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	let credit = $state<Credit | null>(null);
	let banks = $state<Bank[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let scheduleEditing = $state(false);
	let scheduleEditRows = $state<{ id: string; amount: string }[]>([]);
	let scheduleSaving = $state(false);
	let refreshing = $state(false);
	let pendingPage = $state(1);
	let appliedPage = $state(1);
	let retroactivePage = $state(1);
	let loadedForID = $state('');
	let loadSeq = 0;

	const schedulePageSize = 10;

	$effect(() => {
		if (!id) {
			loading = false;
			return;
		}
		if (loadedForID === id) return;
		loadedForID = id;
		credit = null;
		loading = !refCacheReadyAny([creditApiPath, '/api/v1/banks']);
		void load();
	});

	// Path-aware only — do not listen to refCacheTick: assigning credit without
	// assignIfChanged re-fired the effect and spun "Обновляем данные…".
	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || !credit || loadedForID !== id) return;
		if (refCachePathMatches(update.path, [creditApiPath, '/api/v1/banks'])) {
			void load({ background: true });
		}
	});

	$effect(() => {
		const tick = $dataRefreshTick;
		if (tick === 0 || !id || loadedForID !== id) return;
		void load({ background: true });
	});

	async function load(options: { silent?: boolean; background?: boolean } = {}) {
		const silent = options.silent ?? false;
		const seq = ++loadSeq;
		if (silent && credit) {
			refreshing = true;
		} else if (!options.background && !credit) {
			loading = !refCacheReadyAny([creditApiPath, '/api/v1/banks']);
		}
		try {
			const [c, bankList] = await Promise.all([getCredit(id), listBanks()]);
			if (seq !== loadSeq) return;
			credit = options.background ? assignIfChanged(credit, c) : c;
			banks = options.background ? assignIfChanged(banks, bankList) : bankList;
			if (!options.background) {
				pendingPage = 1;
				appliedPage = 1;
				retroactivePage = 1;
			}
			loadError = null;
		} catch (err) {
			if (seq !== loadSeq) return;
			const msg = reportPageLoadFailure(err, {
				background: options.background,
				hasData: !!credit
			});
			if (msg) loadError = msg;
		} finally {
			if (seq === loadSeq) {
				loading = false;
				refreshing = false;
			}
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

	function nextPending(c: Credit): CreditPayment | undefined {
		return nextPendingPayment(c);
	}

	function paymentStatus(p: NonNullable<Credit['schedule']>[number]): string {
		if (p.kind === 'retroactive') return $_('credits.payment.status.retroactive');
		if (!p.is_applied) return $_('credits.payment.status.pending');
		if (p.transaction_kind === 'future') {
			return $_('credits.payment.status.future');
		}
		return $_('credits.payment.status.applied');
	}

	/** Badge colour: pending — warning, debited — success, the rest neutral. */
	function paymentTone(p: CreditPayment): 'warning' | 'success' | 'neutral' {
		if (!p.is_applied) return 'warning';
		if (p.kind !== 'retroactive' && p.transaction_kind !== 'future') return 'success';
		return 'neutral';
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

	function openPay() {
		if (!credit) return;
		void goto(resolve(creditActionPath(credit.id, 'pay', { from: creditPath })));
	}

	function openComplete() {
		if (!credit) return;
		void goto(resolve(creditActionPath(credit.id, 'complete', { from: creditPath })));
	}

	function openChangeAccount() {
		if (!credit) return;
		void goto(resolve(creditActionPath(credit.id, 'change-account', { from: creditPath })));
	}

	function openSetDebitTime() {
		if (!credit) return;
		void goto(resolve(creditActionPath(credit.id, 'debit-time', { from: creditPath })));
	}

	function openChangeName() {
		if (!credit) return;
		void goto(resolve(creditActionPath(credit.id, 'change-name', { from: creditPath })));
	}

	function openChangeBank() {
		if (!credit) return;
		void goto(resolve(creditActionPath(credit.id, 'change-bank', { from: creditPath })));
	}

	/** «сегодня / завтра / через N дн. / просрочен на N дн.» for the next-payment card. */
	function paymentDueLabel(p: CreditPayment): string {
		const today = todayDateLocal(tz).slice(0, 10);
		const due = p.payment_date.slice(0, 10);
		const days = Math.round(
			(Date.parse(`${due}T00:00:00Z`) - Date.parse(`${today}T00:00:00Z`)) / 86_400_000
		);
		if (days === 0) return $_('credits.next.today');
		if (days === 1) return $_('credits.next.tomorrow');
		if (days > 1) return $_('credits.next.inDays', { values: { n: days } });
		return $_('credits.next.overdue', { values: { n: -days } });
	}

	function paymentDateDisplay(p: CreditPayment): string {
		return formatCreditPaymentDateForDisplay(p.payment_date, tz, credit?.debit_time_local);
	}

	function canEditPayment(p: CreditPayment): boolean {
		return credit?.status === 'active' && !refreshing && !p.is_applied && p.kind === 'scheduled';
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
			!refreshing &&
			p.is_applied &&
			p.kind !== 'retroactive' &&
			p.id === latestDeletableAppliedPaymentId
		);
	}

	function canPayPayment(p: CreditPayment): boolean {
		return (
			credit?.status === 'active' &&
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
		void goto(
			resolve(
				creditActionPath(credit.id, 'pay', {
					from: creditPath,
					amount: fromCents(amountCents),
					date: dateOnlyLocalValue(toDatetimeLocalValue(p.payment_date, tz))
				})
			)
		);
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

<div class="space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/credits', label: $_('credits.title') },
			{ href: '/credits', label: credit ? creditName(credit) : $_('credits.title') }
		]}
	/>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if credit}
			{@const nextPayment = credit.status === 'active' ? nextPending(credit) : undefined}
			<div class="flex items-start justify-between gap-3">
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
				<div class="flex shrink-0 items-center gap-2">
					<RowActionsMenu actions={creditPageActions()} />
				</div>
			</div>

			<div class="space-y-8">
				<div class="space-y-3">
					<div class="card overflow-hidden p-0 sm:p-0">
						<div class="stat-grid grid-cols-2 text-sm md:grid-cols-3">
							<div>
								<p class="stat-label">{$_('credits.field.principal')}</p>
								<p class="font-medium">
									<MoneyDisplay value={credit.principal_amount_display} {currency} class="" />
								</p>
							</div>
							{#if credit.credit_kind === 'mortgage'}
								<div>
									<p class="stat-label">{$_('credits.field.propertyPrice')}</p>
									<p class="font-medium">
										{#if credit.property_price_display}
											<MoneyDisplay value={credit.property_price_display} {currency} class="" />
										{:else}
											—
										{/if}
									</p>
								</div>
								<div>
									<p class="stat-label">{$_('credits.field.downPayment')}</p>
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
									<p class="stat-label">{$_('credits.field.totalInterest')}</p>
									<p class="font-medium">
										<MoneyDisplay cents={totalInterestCents(credit)} {currency} class="" />
									</p>
								</div>
							{/if}
							<div>
								<p class="stat-label">{$_('credits.field.payment')}</p>
								<p class="font-medium">
									<MoneyDisplay value={credit.monthly_payment_display} {currency} class="" />
								</p>
							</div>
							{#if !credit.is_installment}
								<div>
									<p class="stat-label">{$_('credits.field.rate')}</p>
									<p class="font-medium">
										{$_('credits.header.rate', {
											values: { rate: formatInterestRate(credit.interest_rate) }
										})}
									</p>
								</div>
							{/if}
							<div>
								<p class="stat-label">{$_('credits.field.issueDate')}</p>
								<p class="font-medium">{formatAPIDateForDisplay(credit.issue_date, tz)}</p>
							</div>
							{#if credit.added_retroactively}
								<div>
									<p class="stat-label">{$_('credits.field.recordedAt')}</p>
									<p class="font-medium">
										{formatAPIOperationDateTimeForDisplay(credit.recorded_at, tz)}
									</p>
								</div>
							{/if}
							<div>
								<p class="stat-label">{$_('credits.field.debitAccount')}</p>
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
									<p class="stat-label">{$_('credits.field.debitTime')}</p>
									<p class="font-medium">{credit.debit_time_local}</p>
								</div>
							{/if}
							<div>
								<p class="stat-label">{$_('credits.field.bank')}</p>
								<p class="font-medium">{credit.bank_name || $_('credits.field.bankNotSelected')}</p>
							</div>
						</div>
					</div>
					{#if refreshing}
						<p class="text-sm" style:color="var(--text-muted)">
							{$_('credits.loading.updating')}
						</p>
					{/if}
				</div>

				{#if nextPayment}
					<section class="space-y-3">
						<GroupHeading title={$_('credits.next.title')} />
						<div class="card flex flex-wrap items-center gap-x-8 gap-y-3">
							<span class="pay-glyph pay-glyph-pending" aria-hidden="true">
								<svg
									class="h-5 w-5"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
									stroke-linecap="round"
									stroke-linejoin="round"
								>
									<circle cx="12" cy="12" r="9" />
									<path d="M12 7v5l3 2" />
								</svg>
							</span>
							<div class="space-y-0.5">
								<p class="stat-label">{$_('transactions.col.amount')}</p>
								<p class="text-xl font-bold tabular-nums">
									<MoneyDisplay value={nextPayment.amount_display} {currency} class="" />
								</p>
							</div>
							<div class="basis-full space-y-0.5 sm:basis-auto">
								<p class="stat-label">{$_('credits.pay.date')}</p>
								<p class="flex items-center gap-2 font-medium whitespace-nowrap">
									{paymentDateDisplay(nextPayment)}
									<span class="badge pay-badge-warning">{paymentDueLabel(nextPayment)}</span>
								</p>
							</div>
							<div class="min-w-0 basis-full space-y-0.5 sm:flex-1 sm:basis-auto">
								<p class="stat-label">{$_('credits.field.debitAccount')}</p>
								<p class="truncate font-medium">{credit.debit_account_name}</p>
							</div>
							<button type="button" class="btn-primary w-full sm:w-auto" onclick={openPay}>
								{$_('credits.action.pay')}
							</button>
						</div>
					</section>
				{/if}

				{#if credit.schedule?.length}
					<section class="space-y-3">
						<GroupHeading title={$_('credits.schedule.title')} />
						<div class="card relative overflow-hidden p-0 sm:p-0">
							{#if refreshing}
								<div
									class="absolute inset-0 z-20 flex items-center justify-center bg-[color-mix(in_srgb,var(--bg)_72%,transparent)]"
								>
									<span class="badge">{$_('credits.loading.updating')}</span>
								</div>
							{/if}

							{#snippet paymentTable(payments: CreditPayment[], editable = false)}
								<div class="hidden overflow-x-auto md:block">
									<table class="data-table w-full table-fixed text-left text-sm">
										<colgroup>
											<col class="w-[34%]" />
											<col class="w-[30%]" />
											<col />
											<col class="w-20" />
										</colgroup>
										<thead>
											<tr>
												<th class="p-3">{$_('credits.pay.date')}</th>
												<th class="p-3">{$_('transactions.col.status')}</th>
												<th class="p-3 text-right">{$_('transactions.col.amount')}</th>
												<th class="p-3"></th>
											</tr>
										</thead>
										<tbody>
											{#each payments as p (p.id)}
												{@const rowActions =
													creditIsActive && !editable ? paymentRowActions(p) : []}
												<tr>
													<td class="p-3">{paymentDateDisplay(p)}</td>
													<td class="p-3">
														<span class="inline-flex flex-wrap items-center gap-1.5">
															<span
																class="badge"
																class:badge-success={paymentTone(p) === 'success'}
																class:pay-badge-warning={paymentTone(p) === 'warning'}
																>{paymentStatus(p)}</span
															>
															{#if paymentStatusExtra(p)}
																<span class="badge">{paymentStatusExtra(p)}</span>
															{/if}
														</span>
													</td>
													<td class="p-3 text-right font-semibold tabular-nums">
														{#if editable && canEditPayment(p)}
															{@const editIdx = scheduleEditRows.findIndex(
																(row) => row.id === p.id
															)}
															{#if editIdx >= 0}
																<div class="ml-auto max-w-40">
																	<MoneyInput bind:value={scheduleEditRows[editIdx].amount} />
																</div>
															{/if}
														{:else}
															<MoneyDisplay value={p.amount_display} {currency} class="" />
														{/if}
													</td>
													<td class="p-3 text-right">
														{#if rowActions.length > 0}
															<RowActionsMenu actions={rowActions} />
														{:else}
															<!-- Same footprint as the menu button keeps every row the same height. -->
															<span class="inline-block h-11 w-11 align-middle" aria-hidden="true"
															></span>
														{/if}
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
								<div class="list-divided md:hidden">
									{#each payments as p (p.id)}
										{@const paymentActions = paymentRowActions(p)}
										{@const editIdx =
											editable && canEditPayment(p)
												? scheduleEditRows.findIndex((row) => row.id === p.id)
												: -1}
										<article class="flex items-center gap-3 px-4 py-3">
											<span
												class="pay-glyph"
												class:pay-glyph-pending={!p.is_applied}
												aria-hidden="true"
											>
												<svg
													class="h-5 w-5"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
													stroke-linecap="round"
													stroke-linejoin="round"
												>
													{#if p.is_applied}
														<path d="M20 6 9 17l-5-5" />
													{:else}
														<circle cx="12" cy="12" r="9" />
														<path d="M12 7v5l3 2" />
													{/if}
												</svg>
											</span>
											<div class="min-w-0 flex-1">
												<p class="truncate text-sm font-medium">{paymentDateDisplay(p)}</p>
												<p
													class="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs"
													style:color="var(--text-muted)"
												>
													{paymentStatus(p)}
													{#if paymentStatusExtra(p)}
														<span class="badge">{paymentStatusExtra(p)}</span>
													{/if}
												</p>
											</div>
											{#if editIdx >= 0}
												<div class="w-32 shrink-0">
													<MoneyInput bind:value={scheduleEditRows[editIdx].amount} />
												</div>
											{:else}
												<span class="shrink-0 font-semibold tabular-nums">
													<MoneyDisplay value={p.amount_display} {currency} class="" />
												</span>
											{/if}
											{#if creditIsActive && !editable && paymentActions.length > 0}
												<RowActionsMenu actions={paymentActions} />
											{/if}
										</article>
									{/each}
								</div>
							{/snippet}

							{#if scheduleGroups.pending.length > 0}
								<details class="border-b" style:border-color="var(--border)">
									<summary
										class="cursor-pointer list-none px-4 py-3 text-sm font-medium select-none [&::-webkit-details-marker]:hidden"
									>
										<div class="flex flex-wrap items-center justify-between gap-2">
											<span class="inline-flex items-center gap-1">
												{$_('credits.schedule.group.pending')}
												<span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)"
													>· {scheduleGroups.pending.length}</span
												><svg
													class="spoiler-chevron"
													aria-hidden="true"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"><path d="m6 9 6 6 6-6" /></svg
												>
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
										class="flex items-center gap-1 cursor-pointer list-none px-4 py-3 text-sm font-medium select-none [&::-webkit-details-marker]:hidden"
									>
										{$_('credits.schedule.group.applied')}
										<span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)"
											>· {scheduleGroups.applied.length}</span
										><svg
											class="spoiler-chevron"
											aria-hidden="true"
											viewBox="0 0 24 24"
											fill="none"
											stroke="currentColor"
											stroke-width="2"><path d="m6 9 6 6 6-6" /></svg
										>
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
										class="flex items-center gap-1 cursor-pointer list-none px-4 py-3 text-sm font-medium select-none [&::-webkit-details-marker]:hidden"
									>
										{$_('credits.schedule.group.retroactive')}
										<span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)"
											>· {scheduleGroups.retroactive.length}</span
										><svg
											class="spoiler-chevron"
											aria-hidden="true"
											viewBox="0 0 24 24"
											fill="none"
											stroke="currentColor"
											stroke-width="2"><path d="m6 9 6 6 6-6" /></svg
										>
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
					</section>
				{/if}
			</div>
		{/if}
	</PageLoadGate>
</div>

<style>
	.pay-glyph {
		display: inline-flex;
		height: 2.5rem;
		width: 2.5rem;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		border-radius: 0.75rem;
		color: var(--primary);
		background-color: color-mix(in srgb, var(--primary) 12%, transparent);
	}

	.pay-badge-warning {
		color: var(--warning);
		border-color: color-mix(in srgb, var(--warning) 30%, transparent);
		background-color: color-mix(in srgb, var(--warning) 14%, transparent);
	}

	.pay-glyph-pending {
		color: var(--warning);
		background-color: color-mix(in srgb, var(--warning) 14%, transparent);
	}
</style>
