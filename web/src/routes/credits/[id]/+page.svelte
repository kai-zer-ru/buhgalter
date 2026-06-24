<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		addCreditPayment,
		completeCredit,
		deleteCredit,
		deleteCreditPayment,
		getCredit,
		listAccounts,
		updateCredit,
		type Account,
		type Credit,
		type CreditPayment
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import Select from '$lib/components/Select.svelte';
	import { toast } from '$lib/toast';
	import { confirm } from '$lib/confirm';
	import TransactionContextStats from '$lib/components/TransactionContextStats.svelte';
	import {
		formatAPIDateForDisplay,
		formatAPIDateTimeForDisplay,
		dateOnlyLocalValue,
		fromDatetimeLocalValue,
		todayDateLocal,
		toDatetimeLocalValue
	} from '$lib/dates';
	import { formatBalance } from '$lib/finance';
	import { toAPIAmount, fromCents } from '$lib/money';
	import { user } from '$lib/stores/auth';

	const id = $derived($page.params.id ?? '');
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');

	let credit = $state<Credit | null>(null);
	let accounts = $state<Account[]>([]);
	let loading = $state(true);
	let error = $state('');
	let payOpen = $state(false);
	let payAmount = $state('');
	let payDateLocal = $state('');
	let changeAccountOpen = $state(false);
	let newAccountId = $state('');
	let completeOpen = $state(false);
	let completeDateLocal = $state('');
	let completeMode = $state<'account' | 'skip'>('account');
	let payError = $state('');
	let completeError = $state('');
	let changeAccountError = $state('');

	const accountOptions = $derived(accounts.map((acc) => ({ value: acc.id, label: acc.name })));

	onMount(() => void load());

	async function load() {
		loading = true;
		error = '';
		try {
			const [c, accs] = await Promise.all([getCredit(id), listAccounts()]);
			credit = c;
			accounts = accs.filter((a) => a.status === 'active');
			newAccountId = c.debit_account_id;
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}

	function creditName(c: Credit): string {
		return c.name?.trim() || $_('credits.title');
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
		payError = '';
		try {
			await addCreditPayment(credit.id, {
				amount: toAPIAmount(payAmount),
				payment_date: fromDatetimeLocalValue(payDateLocal, tz)
			});
			payOpen = false;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			payError = err instanceof ApiError ? err.message : $_('common.error');
		}
	}

	async function submitChangeAccount() {
		if (!credit) return;
		changeAccountError = '';
		try {
			await updateCredit(credit.id, { debit_account_id: newAccountId });
			changeAccountOpen = false;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			changeAccountError = err instanceof ApiError ? err.message : $_('common.error');
		}
	}

	async function submitComplete() {
		if (!credit) return;
		completeError = '';
		try {
			await completeCredit(credit.id, {
				affects_balance: completeMode === 'account',
				payment_date: fromDatetimeLocalValue(completeDateLocal, tz)
			});
			completeOpen = false;
			toast($_('common.saved'));
			await load();
		} catch (err) {
			completeError = err instanceof ApiError ? err.message : $_('common.error');
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
		error = '';
		try {
			await deleteCredit(credit.id, mode);
			window.location.href = resolve('/credits');
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
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

	function setPayDateToday() {
		payDateLocal = todayDateLocal(tz);
	}

	function openPay() {
		if (!credit) return;
		payError = '';
		payAmount = defaultPayAmount(credit);
		payDateLocal = defaultPayDate(credit);
		payOpen = true;
	}

	function openComplete() {
		if (!credit) return;
		completeError = '';
		completeDateLocal = todayDateLocal(tz);
		completeMode = credit.remaining_amount > 0 ? 'account' : 'skip';
		completeOpen = true;
	}

	function openChangeAccount() {
		if (!credit) return;
		changeAccountError = '';
		newAccountId = credit.debit_account_id;
		changeAccountOpen = true;
	}

	function canDeletePayment(p: CreditPayment): boolean {
		return credit?.status === 'active' && p.kind !== 'retroactive';
	}

	type ScheduleGroup = 'pending' | 'applied' | 'retroactive';

	function scheduleGroup(p: CreditPayment): ScheduleGroup {
		if (p.kind === 'retroactive') return 'retroactive';
		if (!p.is_applied) return 'pending';
		return 'applied';
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
		return empty;
	});

	const creditIsActive = $derived(credit?.status === 'active');

	async function doDeletePayment(p: CreditPayment) {
		if (!credit) return;
		const ok = await confirm({
			message: $_('credits.confirm.deletePayment'),
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
	<BackLink href="/credits" label={$_('credits.title')} />

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if error}
		<p style:color="var(--danger)">{error}</p>
	{:else if credit}
		<div class="flex flex-wrap items-start justify-between gap-3">
			<div>
				<h1 class="text-2xl font-semibold">{creditName(credit)}</h1>
				<div class="mt-2 flex flex-wrap items-center gap-2">
					{#if credit.is_installment}
						<span class="badge">{$_('credits.badge.installment')}</span>
					{/if}
					{#if credit.added_retroactively}
						<span class="badge">{$_('credits.badge.retroactive')}</span>
					{/if}
					{#if credit.status === 'closed'}
						<span class="badge badge-success">{$_('credits.badge.closed')}</span>
					{/if}
				</div>
				{#if credit.added_retroactively}
					<dl class="mt-3 space-y-1 text-sm" style:color="var(--text-muted)">
						<div class="flex flex-wrap gap-x-2 gap-y-0.5">
							<dt class="shrink-0">{$_('credits.field.issueDate')}:</dt>
							<dd>{formatAPIDateForDisplay(credit.issue_date, tz)}</dd>
						</div>
						<div class="flex flex-wrap gap-x-2 gap-y-0.5">
							<dt class="shrink-0">{$_('credits.field.recordedAt')}:</dt>
							<dd>{formatAPIDateTimeForDisplay(credit.recorded_at, tz)}</dd>
						</div>
					</dl>
				{/if}
			</div>
			{#if credit.status === 'active'}
				<div class="flex flex-wrap gap-2">
					{#if nextPendingPayment(credit)}
						<button type="button" class="btn-primary" onclick={openPay}>
							{$_('credits.action.pay')}
						</button>
					{/if}
					<button type="button" class="btn-ghost" onclick={openChangeAccount}>
						{$_('credits.action.changeAccount')}
					</button>
					<button type="button" class="btn-ghost" onclick={openComplete}>
						{$_('credits.action.complete')}
					</button>
					<button type="button" class="btn-ghost" onclick={() => void doDelete()}>
						{$_('common.delete')}
					</button>
				</div>
			{/if}
		</div>

		<div class="card grid gap-3 p-4 text-sm sm:grid-cols-2">
			<div>
				<span style:color="var(--text-muted)">{$_('credits.field.principal')}</span>
				<p class="font-medium">{credit.principal_amount_display}</p>
			</div>
			<div>
				<span style:color="var(--text-muted)">{$_('credits.col.remaining')}</span>
				<p class="font-medium">{credit.remaining_amount_display}</p>
			</div>
			<div>
				<span style:color="var(--text-muted)">{$_('credits.field.payment')}</span>
				<p class="font-medium">{credit.monthly_payment_display}</p>
			</div>
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
		</div>

		<TransactionContextStats params={{ credit_id: id }} />

		{#if credit.schedule?.length}
			<div class="card overflow-hidden">
				<h2 class="border-b px-4 py-3 text-sm font-semibold" style:border-color="var(--border)">
					{$_('credits.schedule.title')}
				</h2>

				{#snippet paymentTable(payments: CreditPayment[])}
					<div class="hidden overflow-x-auto md:block">
						<table class="w-full text-left text-sm">
							<thead>
								<tr style:color="var(--text-muted)">
									<th class="p-3">{$_('credits.pay.date')}</th>
									<th class="p-3">{$_('transactions.col.amount')}</th>
									<th class="p-3">{$_('transactions.col.status')}</th>
									{#if creditIsActive}
										<th class="p-3"></th>
									{/if}
								</tr>
							</thead>
							<tbody>
								{#each payments as p (p.id)}
									<tr class="border-t" style:border-color="var(--border)">
										<td class="p-3">{formatAPIDateTimeForDisplay(p.payment_date, tz)}</td>
										<td class="p-3">{p.amount_display}</td>
										<td class="p-3">
											{paymentStatus(p)}
											{#if paymentStatusExtra(p)}
												<span class="badge ml-2">{paymentStatusExtra(p)}</span>
											{/if}
										</td>
										{#if creditIsActive}
											<td class="p-3 text-right">
												{#if canDeletePayment(p)}
													<button
														type="button"
														class="btn-ghost text-sm"
														onclick={() => void doDeletePayment(p)}
													>
														{$_('credits.payment.delete')}
													</button>
												{/if}
											</td>
										{/if}
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					<div class="space-y-3 p-3 md:hidden">
						{#each payments as p (p.id)}
							<article class="rounded-xl border p-3" style:border-color="var(--border)">
								<dl class="grid gap-2 text-sm">
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">{$_('credits.pay.date')}</dt>
										<dd>{formatAPIDateTimeForDisplay(p.payment_date, tz)}</dd>
									</div>
									<div class="flex justify-between gap-2">
										<dt style:color="var(--text-muted)">{$_('transactions.col.amount')}</dt>
										<dd class="font-medium tabular-nums">{p.amount_display}</dd>
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
								{#if creditIsActive && canDeletePayment(p)}
									<button
										type="button"
										class="btn-ghost mt-3 w-full text-sm"
										onclick={() => void doDeletePayment(p)}
									>
										{$_('credits.payment.delete')}
									</button>
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
							{$_('credits.schedule.group.pending')}
							<span class="ml-1 font-normal tabular-nums" style:color="var(--text-muted)">
								({scheduleGroups.pending.length})
							</span>
						</summary>
						{@render paymentTable(scheduleGroups.pending)}
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
						{@render paymentTable(scheduleGroups.applied)}
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
						{@render paymentTable(scheduleGroups.retroactive)}
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
			<div class="space-y-1">
				<div class="flex gap-2">
					<div class="min-w-0 flex-1">
						<DateTimePicker
							label={$_('credits.pay.date')}
							bind:value={payDateLocal}
							timeMode="hidden"
							usePortal
						/>
					</div>
					<button
						type="button"
						class="btn-ghost mt-6 shrink-0 self-start"
						onclick={setPayDateToday}
					>
						{$_('credits.pay.today')}
					</button>
				</div>
			</div>
			{#if payRemaining() !== null}
				<p class="text-sm" style:color="var(--text-muted)">
					{$_('credits.pay.preview')}: {formatBalance(
						fromCents(payRemaining()!),
						$user?.currency ?? 'RUB'
					)}
				</p>
			{/if}
			<FormFeedback error={payError} />
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
									amount: activeCredit.remaining_amount_display,
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
				timeMode="hidden"
				usePortal
			/>
			<FormFeedback error={completeError} />
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
			<FormFeedback error={changeAccountError} />
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
