<script lang="ts">
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import {
		getTransaction,
		listAccounts,
		type Account,
		type Subscription,
		type SubscriptionPeriod
	} from '$lib/api/client';
	import { createSubscription, updateSubscription } from '$lib/offline/subscription-api';
	import CategoryIconPicker from '$lib/components/CategoryIconPicker.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import { todayDateLocal, fromDatetimeLocalValue, toDatetimeLocalValue } from '$lib/dates';
	import { defaultAccountId } from '$lib/accounts';
	import { formatMoneyForInput, toAPIAmount } from '$lib/money';
	import { accountSelectOptions } from '$lib/select-options';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import {
		fetchUpcomingLocalDates,
		seedUpcomingLocalDates,
		upcomingLocalToAPI,
		upcomingToLocalDates
	} from '$lib/subscription-upcoming';

	type Props = {
		backHref?: string;
		subscription?: Subscription | null;
		fromTxId?: string;
		onclose: () => void;
		onsaved: () => void;
	};

	let {
		backHref = '/subscriptions',
		subscription = null,
		fromTxId = '',
		onclose,
		onsaved
	}: Props = $props();

	let accounts = $state<Account[]>([]);
	let name = $state('');
	let amount = $state('');
	let description = $state('');
	let icon = $state('subscription');
	let websiteUrl = $state('');
	let accountId = $state('');
	let period = $state<SubscriptionPeriod>('month');
	let weekday = $state('1');
	let dayOfMonth = $state('1');
	let startDate = $state('');
	let timeLocal = $state('08:00');
	let active = $state(true);
	let attachTransactionId = $state<string | null>(null);
	let upcomingDates = $state<string[]>(['', '', '']);
	let upcomingLoading = $state(false);
	let saving = $state(false);
	let ready = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const isEdit = $derived(Boolean(subscription?.id));
	const accountOptions = $derived(accountSelectOptions(accounts));
	const pageTitle = $derived.by(() => {
		void $locale;
		return isEdit ? tr('common.edit') : tr('subscriptions.add');
	});

	$effect(() => {
		void subscription?.id;
		void fromTxId;
		void init();
	});

	function dayFromDate(value: string): string {
		const day = Number((value || '').split('T')[0]?.split('-')[2] ?? '');
		if (!Number.isFinite(day) || day < 1 || day > 31) return '1';
		return String(day);
	}

	function dateOnly(value: string): string {
		return (value || '').split('T')[0] ?? '';
	}

	function usesDayOfMonth(p: SubscriptionPeriod): boolean {
		return p === 'month' || p === 'quarter' || p === 'half_year' || p === 'year';
	}

	function syncDayOfMonthFromStartDate(nextPeriod: SubscriptionPeriod, nextStartDate: string) {
		if (nextPeriod !== 'month' && nextPeriod !== 'quarter' && nextPeriod !== 'half_year') return;
		dayOfMonth = dayFromDate(nextStartDate);
	}

	function onPeriodChange(nextPeriod: SubscriptionPeriod) {
		period = nextPeriod;
		syncDayOfMonthFromStartDate(nextPeriod, startDate);
		void refreshUpcoming();
	}

	function previewPayload() {
		return {
			period,
			weekday: period === 'week' || period === 'two_weeks' ? Number(weekday) : undefined,
			day_of_month: usesDayOfMonth(period)
				? period === 'year'
					? Number(dayOfMonth)
					: Number(dayFromDate(startDate))
				: undefined,
			start_date: fromDatetimeLocalValue(`${dateOnly(startDate)}T00:00`, tz),
			time_local: timeLocal || '08:00'
		};
	}

	async function refreshUpcoming() {
		if (!startDate) return;
		upcomingLoading = true;
		try {
			upcomingDates = await fetchUpcomingLocalDates(previewPayload(), tz);
		} finally {
			upcomingLoading = false;
		}
	}

	async function init() {
		ready = false;
		const current = subscription;
		const txId = fromTxId;
		try {
			const list = await listAccounts('active');
			accounts = list;

			if (current) {
				name = current.name;
				amount = formatMoneyForInput(current.amount_display);
				description = current.description ?? '';
				icon = current.icon || 'subscription';
				websiteUrl = current.website_url ?? '';
				accountId = current.account_id;
				period = current.period;
				weekday = String(current.weekday ?? 1);
				dayOfMonth = String(current.day_of_month ?? 1);
				startDate = toDatetimeLocalValue(current.start_date, tz).slice(0, 10);
				syncDayOfMonthFromStartDate(current.period, startDate);
				timeLocal = current.time_local || '08:00';
				active = current.active;
				attachTransactionId = null;
				upcomingDates = upcomingToLocalDates(current.upcoming_run_ats, tz);
				if (!upcomingDates[0]) await refreshUpcoming();
			} else {
				name = '';
				amount = '';
				description = '';
				icon = 'subscription';
				websiteUrl = '';
				accountId = defaultAccountId(list);
				period = 'month';
				weekday = '1';
				dayOfMonth = '1';
				startDate = todayDateLocal(tz).slice(0, 10);
				syncDayOfMonthFromStartDate('month', startDate);
				timeLocal = '08:00';
				active = true;
				attachTransactionId = null;
				upcomingDates = [...seedUpcomingLocalDates(startDate, 'month')];
				if (txId) {
					await prefillFromTransaction(txId);
				}
				await refreshUpcoming();
			}
		} catch (err) {
			toast.fromError(err);
		} finally {
			ready = true;
		}
	}

	async function prefillFromTransaction(txID: string) {
		try {
			const tx = await getTransaction(txID);
			if (tx.type !== 'expense') return;
			name = (tx.description ?? tx.category_name ?? '').trim() || tr('subscriptions.title');
			amount = formatMoneyForInput(tx.amount_display);
			description = tx.description ?? '';
			accountId = tx.account_id;
			const local = toDatetimeLocalValue(tx.transaction_date, tz);
			startDate = (local.split('T')[0] ?? todayDateLocal(tz).slice(0, 10)) as string;
			syncDayOfMonthFromStartDate(period, startDate);
			active = true;
			attachTransactionId = tx.id;
			await refreshUpcoming();
			toast(tr('subscriptions.prefilled'));
		} catch {
			// Ignore optional prefill failures.
		}
	}

	async function submit(e: Event) {
		e.preventDefault();
		saving = true;
		try {
			const payload = {
				name: name.trim(),
				amount: toAPIAmount(amount),
				description: description.trim() || undefined,
				icon: icon.trim() || undefined,
				website_url: websiteUrl.trim() || undefined,
				account_id: accountId,
				period,
				weekday: period === 'week' || period === 'two_weeks' ? Number(weekday) : undefined,
				day_of_month: usesDayOfMonth(period)
					? period === 'year'
						? Number(dayOfMonth)
						: Number(dayFromDate(startDate))
					: undefined,
				start_date: fromDatetimeLocalValue(`${dateOnly(startDate)}T00:00`, tz),
				time_local: timeLocal || '08:00',
				active,
				upcoming_run_ats: upcomingLocalToAPI(upcomingDates, timeLocal || '08:00', tz),
				attach_transaction_id: !isEdit && attachTransactionId ? attachTransactionId : undefined
			};
			if (isEdit && subscription) {
				await updateSubscription(subscription.id, payload);
			} else {
				await createSubscription(payload);
			}
			toast(tr('common.saved'));
			onsaved();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}
</script>

{#snippet formBody()}
	{#if ready}
		<form id="subscription-form" class="space-y-4" onsubmit={submit}>
			<div>
				<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="subscription-name">
					{$_('subscriptions.field.name')}
				</label>
				<input
					id="subscription-name"
					class="input w-full"
					bind:value={name}
					placeholder={$_('subscriptions.field.name')}
					maxlength="120"
					required
				/>
			</div>
			<div>
				<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="subscription-amount">
					{$_('transactions.field.amount')}
				</label>
				<MoneyInput id="subscription-amount" bind:value={amount} />
			</div>
			<div>
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="subscription-description"
				>
					{$_('subscriptions.field.description')}
				</label>
				<input
					id="subscription-description"
					class="input w-full"
					bind:value={description}
					placeholder={$_('subscriptions.field.description')}
					maxlength="160"
				/>
			</div>
			<div>
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="subscription-website"
				>
					{$_('subscriptions.field.website')}
				</label>
				<input
					id="subscription-website"
					class="input w-full"
					type="url"
					bind:value={websiteUrl}
					placeholder="https://"
					maxlength="500"
				/>
			</div>
			<div>
				<p class="mb-1 text-sm" style:color="var(--text-muted)">{$_('subscriptions.field.icon')}</p>
				<CategoryIconPicker
					bind:value={icon}
					bind:categoryName={name}
					categoryType="expense"
					lockName={true}
					variant="button"
				/>
			</div>
			<Select
				label={$_('transactions.field.account')}
				bind:value={accountId}
				options={accountOptions}
				usePortal
			/>
			<div>
				<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="subscription-period"
					>{$_('recurring.period')}</label
				>
				<select
					id="subscription-period"
					class="input w-full"
					bind:value={period}
					onchange={(e) =>
						onPeriodChange((e.currentTarget as HTMLSelectElement).value as SubscriptionPeriod)}
				>
					<option value="week">{$_('recurring.period.week')}</option>
					<option value="two_weeks">{$_('recurring.period.twoWeeks')}</option>
					<option value="month">{$_('recurring.period.month')}</option>
					<option value="quarter">{$_('subscriptions.period.quarter')}</option>
					<option value="half_year">{$_('subscriptions.period.halfYear')}</option>
					<option value="year">{$_('recurring.period.year')}</option>
				</select>
			</div>
			<DateTimePicker
				id="subscription-start-date"
				label={$_('recurring.startDate')}
				bind:value={startDate}
				{...dateOnlyPicker}
				usePortal
				required
			/>
			{#if period === 'week' || period === 'two_weeks'}
				<div>
					<label
						class="mb-1 block text-sm"
						style:color="var(--text-muted)"
						for="subscription-weekday">{$_('recurring.weekday')}</label
					>
					<select
						id="subscription-weekday"
						class="input w-full"
						bind:value={weekday}
						onchange={() => void refreshUpcoming()}
					>
						<option value="1">{$_('datetime.weekday.mon')}</option>
						<option value="2">{$_('datetime.weekday.tue')}</option>
						<option value="3">{$_('datetime.weekday.wed')}</option>
						<option value="4">{$_('datetime.weekday.thu')}</option>
						<option value="5">{$_('datetime.weekday.fri')}</option>
						<option value="6">{$_('datetime.weekday.sat')}</option>
						<option value="7">{$_('datetime.weekday.sun')}</option>
					</select>
				</div>
			{:else if period === 'year'}
				<div>
					<label
						class="mb-1 block text-sm"
						style:color="var(--text-muted)"
						for="subscription-day-of-month">{$_('recurring.dayOfMonth')}</label
					>
					<input
						id="subscription-day-of-month"
						class="input w-full"
						type="number"
						min="1"
						max="31"
						bind:value={dayOfMonth}
						onchange={() => void refreshUpcoming()}
					/>
				</div>
			{/if}
			<div class="space-y-2">
				<div class="flex flex-wrap items-center justify-between gap-2">
					<p class="text-sm font-medium">{$_('subscriptions.upcomingDates')}</p>
					<button
						type="button"
						class="btn-ghost text-sm"
						disabled={upcomingLoading}
						onclick={() => void refreshUpcoming()}
					>
						{$_('subscriptions.upcomingReset')}
					</button>
				</div>
				<p class="text-xs" style:color="var(--text-muted)">{$_('subscriptions.upcomingHint')}</p>
				<DateTimePicker
					id="subscription-upcoming-0"
					label={$_('subscriptions.upcoming1')}
					bind:value={upcomingDates[0]}
					{...dateOnlyPicker}
					usePortal
					required
				/>
				<DateTimePicker
					id="subscription-upcoming-1"
					label={$_('subscriptions.upcoming2')}
					bind:value={upcomingDates[1]}
					{...dateOnlyPicker}
					usePortal
					required
				/>
				<DateTimePicker
					id="subscription-upcoming-2"
					label={$_('subscriptions.upcoming3')}
					bind:value={upcomingDates[2]}
					{...dateOnlyPicker}
					usePortal
					required
				/>
			</div>
			<details>
				<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
					{$_('recurring.timeAdvanced')}
				</summary>
				<div class="mt-2">
					<input
						class="input w-full"
						type="time"
						bind:value={timeLocal}
						step="60"
						onchange={() => void refreshUpcoming()}
					/>
				</div>
			</details>
			<label class="inline-flex items-center gap-2 text-sm">
				<input type="checkbox" bind:checked={active} />
				{$_('subscriptions.active')}
			</label>
		</form>
	{:else}
		<p class="text-sm" style:color="var(--text-muted)">{$_('common.loading')}</p>
	{/if}
{/snippet}

{#snippet formFooter()}
	<button type="button" class="btn-ghost" onclick={onclose}>{$_('common.cancel')}</button>
	<button type="submit" form="subscription-form" class="btn-primary" disabled={saving || !ready}>
		{saving ? $_('common.loading') : isEdit ? $_('common.save') : $_('common.create')}
	</button>
{/snippet}

<FormPageShell title={pageTitle} {backHref} onback={onclose}>
	{@render formBody()}
	{#snippet footer()}
		{@render formFooter()}
	{/snippet}
</FormPageShell>
