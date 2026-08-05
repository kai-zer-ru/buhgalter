<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		attachSubscriptionTransactions,
		createSubscription,
		deleteSubscription,
		getTransaction,
		getSubscriptionsSummary,
		listAccounts,
		listSubscriptions,
		updateSubscription,
		type Account,
		type Subscription,
		type SubscriptionPeriod,
		type SubscriptionSummary
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import CategoryIconPicker from '$lib/components/CategoryIconPicker.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import Select from '$lib/components/Select.svelte';
	import { confirm } from '$lib/confirm';
	import {
		todayDateLocal,
		fromDatetimeLocalValue,
		toDatetimeLocalValue,
		formatAPIOperationDateTimeForDisplay
	} from '$lib/dates';
	import { formatMoneyForInput, toAPIAmount } from '$lib/money';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { defaultAccountId } from '$lib/accounts';
	import { accountSelectOptions } from '$lib/select-options';
	import { user } from '$lib/stores/auth';

	let items = $state<Subscription[]>([]);
	let summary = $state<SubscriptionSummary | null>(null);
	let accounts = $state<Account[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let saving = $state(false);
	let editId = $state<string | null>(null);
	let formOpen = $state(false);
	let attachTxId = $state<string | null>(null);
	let attaching = $state(false);

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

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');
	const accountOptions = $derived(accountSelectOptions(accounts));

	onMount(() => {
		startDate = todayDateLocal(tz).slice(0, 10);
		syncDayOfMonthFromStartDate('month', startDate);
		void loadAll();
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

	async function loadAll() {
		loading = true;
		try {
			const [subs, sum, accs] = await Promise.all([
				listSubscriptions(),
				getSubscriptionsSummary({ upcoming_days: 14 }),
				listAccounts('active')
			]);
			items = subs;
			summary = sum;
			accounts = accs;
			if (!accountId) accountId = defaultAccountId(accounts);
			await prefillFromQueryTransaction();
			await handleAttachQuery();
			await handleEditQuery();
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: items.length > 0 });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	async function prefillFromQueryTransaction() {
		const txID = $page.url.searchParams.get('from_tx');
		if (!txID) return;
		try {
			const tx = await getTransaction(txID);
			if (tx.type !== 'expense') return;
			name = (tx.description ?? tx.category_name ?? '').trim() || $_('subscriptions.title');
			amount = formatMoneyForInput(tx.amount_display);
			description = tx.description ?? '';
			accountId = tx.account_id;
			const local = toDatetimeLocalValue(tx.transaction_date, tz);
			startDate = (local.split('T')[0] ?? todayDateLocal(tz).slice(0, 10)) as string;
			syncDayOfMonthFromStartDate(period, startDate);
			active = true;
			attachTransactionId = tx.id;
			formOpen = true;
			await goto(resolve('/subscriptions'), {
				replaceState: true,
				noScroll: true,
				keepFocus: true
			});
			toast($_('subscriptions.prefilled'));
		} catch {
			// Ignore optional prefill failures.
		}
	}

	async function handleAttachQuery() {
		const txID = $page.url.searchParams.get('attach_tx');
		if (!txID) {
			attachTxId = null;
			return;
		}
		attachTxId = txID;
		await goto(resolve('/subscriptions'), {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
	}

	async function handleEditQuery() {
		const id = $page.url.searchParams.get('edit');
		if (!id) return;
		const item = items.find((sub) => sub.id === id);
		if (item) beginEdit(item);
		await goto(resolve('/subscriptions'), {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
	}

	async function attachToSubscription(item: Subscription) {
		if (!attachTxId) return;
		attaching = true;
		try {
			await attachSubscriptionTransactions(item.id, [attachTxId]);
			toast($_('subscriptions.attached'));
			attachTxId = null;
			await loadAll();
		} catch (err) {
			toast.fromError(err);
		} finally {
			attaching = false;
		}
	}

	function resetForm() {
		editId = null;
		name = '';
		amount = '';
		description = '';
		icon = 'subscription';
		websiteUrl = '';
		accountId = defaultAccountId(accounts);
		period = 'month';
		weekday = '1';
		dayOfMonth = '1';
		startDate = todayDateLocal(tz).slice(0, 10);
		timeLocal = '08:00';
		active = true;
		attachTransactionId = null;
	}

	function beginEdit(item: Subscription) {
		if (editId === item.id) {
			resetForm();
			return;
		}
		formOpen = false;
		editId = item.id;
		name = item.name;
		amount = formatMoneyForInput(item.amount_display);
		description = item.description ?? '';
		icon = item.icon || 'default';
		websiteUrl = item.website_url ?? '';
		accountId = item.account_id;
		period = item.period;
		weekday = String(item.weekday ?? 1);
		dayOfMonth = String(item.day_of_month ?? 1);
		startDate = toDatetimeLocalValue(item.start_date, tz).slice(0, 10);
		syncDayOfMonthFromStartDate(item.period, startDate);
		timeLocal = item.time_local || '08:00';
		active = item.active;
		attachTransactionId = null;
	}

	async function remove(item: Subscription) {
		const ok = await confirm({
			message: $_('subscriptions.confirmDelete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteSubscription(item.id);
			toast($_('common.deleted'));
			await loadAll();
			if (editId === item.id) resetForm();
		} catch (err) {
			toast.fromError(err);
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
				attach_transaction_id: !editId && attachTransactionId ? attachTransactionId : undefined
			};
			if (editId) {
				await updateSubscription(editId, payload);
			} else {
				await createSubscription(payload);
			}
			toast($_('common.saved'));
			await loadAll();
			resetForm();
			formOpen = false;
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}

	function toggleForm() {
		if (formOpen && !editId) {
			formOpen = false;
			return;
		}
		formOpen = true;
		if (editId) resetForm();
	}

	function onPeriodChange(nextPeriod: SubscriptionPeriod) {
		period = nextPeriod;
		syncDayOfMonthFromStartDate(nextPeriod, startDate);
	}

	function periodLabel(itemPeriod: SubscriptionPeriod): string {
		switch (itemPeriod) {
			case 'week':
				return $_('recurring.period.week');
			case 'two_weeks':
				return $_('recurring.period.twoWeeks');
			case 'month':
				return $_('recurring.period.month');
			case 'quarter':
				return $_('subscriptions.period.quarter');
			case 'half_year':
				return $_('subscriptions.period.halfYear');
			default:
				return $_('recurring.period.year');
		}
	}

	function rowActions(item: Subscription): RowAction[] {
		return [
			{
				icon: 'create',
				label: $_('subscriptions.findTransactions'),
				onclick: () =>
					void goto(resolve(`/subscriptions/${encodeURIComponent(item.id)}/find-transactions`))
			},
			{
				icon: 'edit',
				label: $_('common.edit'),
				onclick: () => beginEdit(item)
			},
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void remove(item)
			}
		];
	}

	$effect(() => {
		syncDayOfMonthFromStartDate(period, startDate);
	});
</script>

{#snippet subscriptionForm(formPrefix: 'create' | 'edit' | 'edit-mobile')}
	<form class="space-y-4" onsubmit={submit}>
		<div class="grid gap-3 md:grid-cols-2">
			<div>
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="subscription-name-{formPrefix}"
				>
					{$_('subscriptions.field.name')}
				</label>
				<input
					id="subscription-name-{formPrefix}"
					class="input w-full"
					bind:value={name}
					placeholder={$_('subscriptions.field.name')}
					maxlength="120"
					required
				/>
			</div>
			<div>
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="subscription-amount-{formPrefix}"
				>
					{$_('transactions.field.amount')}
				</label>
				<MoneyInput id="subscription-amount-{formPrefix}" bind:value={amount} />
			</div>
		</div>

		<div class="grid gap-3 md:grid-cols-2">
			<div>
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="subscription-description-{formPrefix}"
				>
					{$_('subscriptions.field.description')}
				</label>
				<input
					id="subscription-description-{formPrefix}"
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
					for="subscription-website-{formPrefix}"
				>
					{$_('subscriptions.field.website')}
				</label>
				<input
					id="subscription-website-{formPrefix}"
					class="input w-full"
					type="url"
					bind:value={websiteUrl}
					placeholder="https://"
					maxlength="500"
				/>
			</div>
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

		<div class="grid gap-3 md:grid-cols-3">
			<Select
				label={$_('transactions.field.account')}
				bind:value={accountId}
				options={accountOptions}
				usePortal
			/>
			<div>
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="subscription-period-{formPrefix}">{$_('recurring.period')}</label
				>
				<select
					id="subscription-period-{formPrefix}"
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
			<div>
				<DateTimePicker
					id="subscription-start-date-{formPrefix}"
					label={$_('recurring.startDate')}
					bind:value={startDate}
					{...dateOnlyPicker}
					usePortal
					required
				/>
			</div>
		</div>

		{#if period === 'week' || period === 'two_weeks'}
			<div class="max-w-xs">
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="subscription-weekday-{formPrefix}">{$_('recurring.weekday')}</label
				>
				<select id="subscription-weekday-{formPrefix}" class="input w-full" bind:value={weekday}>
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
			<div class="max-w-xs">
				<label
					class="mb-1 block text-sm"
					style:color="var(--text-muted)"
					for="subscription-day-of-month-{formPrefix}">{$_('recurring.dayOfMonth')}</label
				>
				<input
					id="subscription-day-of-month-{formPrefix}"
					class="input w-full"
					type="number"
					min="1"
					max="31"
					bind:value={dayOfMonth}
				/>
			</div>
		{/if}

		<details>
			<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
				{$_('recurring.timeAdvanced')}
			</summary>
			<div class="mt-2 max-w-xs">
				<input class="input w-full" type="time" bind:value={timeLocal} step="60" />
			</div>
		</details>
		<label class="inline-flex items-center gap-2 text-sm">
			<input type="checkbox" bind:checked={active} />
			{$_('subscriptions.active')}
		</label>
		<div class="flex flex-wrap gap-2">
			<button type="submit" class="btn-primary" disabled={saving}>
				{saving
					? $_('common.loading')
					: formPrefix === 'create'
						? $_('common.create')
						: $_('common.save')}
			</button>
			{#if formPrefix !== 'create'}
				<button type="button" class="btn-ghost" onclick={resetForm}>{$_('common.cancel')}</button>
			{/if}
		</div>
	</form>
{/snippet}

<svelte:head>
	<title>{$_('subscriptions.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-5">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/subscriptions', label: $_('subscriptions.title') }
		]}
	/>

	<SectionHeader title={$_('subscriptions.title')}>
		{#snippet actions()}
			<button type="button" class="btn-primary shrink-0" onclick={toggleForm}>
				{formOpen && !editId ? $_('common.cancel') : $_('subscriptions.add')}
			</button>
		{/snippet}
	</SectionHeader>

	{#if attachTxId}
		<div class="card space-y-3">
			<h2 class="text-base font-semibold">{$_('subscriptions.selectToAttach')}</h2>
			{#if items.length === 0}
				<p class="text-sm" style:color="var(--text-muted)">{$_('subscriptions.empty')}</p>
			{:else}
				<ul class="space-y-2">
					{#each items as item (item.id)}
						<li>
							<button
								type="button"
								class="btn-ghost w-full justify-start text-left"
								disabled={attaching}
								onclick={() => void attachToSubscription(item)}
							>
								<span class="inline-flex items-center gap-2">
									<CategoryIcon icon={item.icon || 'default'} size={24} />
									<span class="font-medium">{item.name}</span>
									<span class="text-sm" style:color="var(--text-muted)">
										<MoneyDisplay value={item.amount_display} {currency} class="" />
									</span>
								</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
			<button type="button" class="btn-ghost" onclick={() => (attachTxId = null)}>
				{$_('common.cancel')}
			</button>
		</div>
	{/if}

	{#if summary}
		<div class="grid gap-3 sm:grid-cols-2">
			<div class="card p-4">
				<p class="text-sm" style:color="var(--text-muted)">{$_('subscriptions.summary.monthly')}</p>
				<p class="mt-1 text-xl font-semibold tabular-nums">
					<MoneyDisplay value={summary.monthly_total_display} {currency} class="" />
				</p>
			</div>
			<div class="card p-4">
				<p class="text-sm" style:color="var(--text-muted)">{$_('subscriptions.summary.yearly')}</p>
				<p class="mt-1 text-xl font-semibold tabular-nums">
					<MoneyDisplay value={summary.yearly_total_display} {currency} class="" />
				</p>
			</div>
		</div>
		{#if summary.upcoming.length > 0}
			<div class="card space-y-3 p-4">
				<h2 class="text-base font-semibold">{$_('subscriptions.upcoming')}</h2>
				<ul class="space-y-2 text-sm">
					{#each summary.upcoming as item (item.id)}
						<li class="flex flex-wrap items-center justify-between gap-2">
							<span class="inline-flex min-w-0 items-center gap-2">
								<CategoryIcon icon={item.icon || 'default'} size={24} />
								<span class="truncate font-medium">{item.name}</span>
							</span>
							<span class="shrink-0 tabular-nums" style:color="var(--text-muted)">
								{formatAPIOperationDateTimeForDisplay(item.next_run_at, tz)}
								·
								<MoneyDisplay value={item.amount_display} {currency} class="" />
							</span>
						</li>
					{/each}
				</ul>
			</div>
		{/if}
	{/if}

	{#if formOpen && !editId}
		<div class="card">
			{@render subscriptionForm('create')}
		</div>
	{/if}

	<PageLoadGate {loading} error={loadError} onretry={() => void loadAll()} inline>
		{#if items.length === 0 && !formOpen}
			<EmptyStateCard message={$_('subscriptions.empty')} />
		{:else if items.length > 0}
			<div class="card md:overflow-x-auto">
				<div class="hidden md:block">
					<table class="w-full text-left text-sm">
						<thead>
							<tr style:color="var(--text-muted)">
								<th class="p-3">{$_('subscriptions.field.name')}</th>
								<th class="p-3">{$_('recurring.period')}</th>
								<th class="p-3">{$_('transactions.field.account')}</th>
								<th class="p-3">{$_('recurring.nextRun')}</th>
								<th class="p-3"></th>
							</tr>
						</thead>
						<tbody>
							{#each items as item (item.id)}
								<tr class="border-t" style:border-color="var(--border)">
									<td class="p-3">
										<div class="flex items-center gap-2">
											<CategoryIcon icon={item.icon || 'default'} size={28} />
											<div class="min-w-0">
												<div class="font-medium">
													{item.name}
													{#if !item.active}
														<span class="text-xs font-normal" style:color="var(--text-muted)">
															({$_('subscriptions.inactive')})
														</span>
													{/if}
												</div>
												<div class="text-xs" style:color="var(--text-muted)">
													{#if item.description}
														{item.description} ·
													{/if}
													<MoneyDisplay value={item.amount_display} {currency} class="" />
												</div>
											</div>
										</div>
									</td>
									<td class="p-3">{periodLabel(item.period)}</td>
									<td class="p-3">{item.account_name}</td>
									<td class="p-3">{formatAPIOperationDateTimeForDisplay(item.next_run_at, tz)}</td>
									<td class="p-3 text-right">
										<RowActionsMenu actions={rowActions(item)} />
									</td>
								</tr>
								{#if editId === item.id}
									<tr class="border-t" style:border-color="var(--border)">
										<td colspan="5" class="p-3">
											{@render subscriptionForm('edit')}
										</td>
									</tr>
								{/if}
							{/each}
						</tbody>
					</table>
				</div>

				<div class="space-y-3 p-3 md:hidden">
					{#each items as item (item.id)}
						<article class="rounded-xl border p-4" style:border-color="var(--border)">
							<div class="flex items-start justify-between gap-3">
								<div class="flex min-w-0 items-start gap-2">
									<CategoryIcon icon={item.icon || 'default'} size={28} />
									<div class="min-w-0">
										<p class="font-medium">{item.name}</p>
										{#if item.description}
											<p class="mt-1 text-xs" style:color="var(--text-muted)">{item.description}</p>
										{/if}
									</div>
								</div>
								<p class="shrink-0 text-sm font-semibold tabular-nums">
									<MoneyDisplay value={item.amount_display} {currency} class="" />
								</p>
							</div>
							<dl class="mt-3 grid gap-2 text-sm">
								<div class="flex justify-between gap-2">
									<dt style:color="var(--text-muted)">{$_('recurring.period')}</dt>
									<dd>{periodLabel(item.period)}</dd>
								</div>
								<div class="flex justify-between gap-2">
									<dt style:color="var(--text-muted)">{$_('transactions.field.account')}</dt>
									<dd class="text-right">{item.account_name}</dd>
								</div>
								<div class="flex justify-between gap-2">
									<dt style:color="var(--text-muted)">{$_('recurring.nextRun')}</dt>
									<dd class="text-right">
										{formatAPIOperationDateTimeForDisplay(item.next_run_at, tz)}
									</dd>
								</div>
							</dl>
							<div class="mt-3 flex justify-end">
								<RowActionsMenu actions={rowActions(item)} />
							</div>
						</article>
						{#if editId === item.id}
							<div class="rounded-xl border p-4" style:border-color="var(--border)">
								{@render subscriptionForm('edit-mobile')}
							</div>
						{/if}
					{/each}
				</div>
			</div>
		{/if}
	</PageLoadGate>
</div>
