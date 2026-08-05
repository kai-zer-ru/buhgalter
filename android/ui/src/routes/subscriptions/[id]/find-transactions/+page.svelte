<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		attachSubscriptionTransactions,
		getUIMeta,
		listSubscriptionCandidateTransactions,
		listSubscriptions,
		type Account,
		type Subscription,
		type SubscriptionCandidateTransaction
	} from '$lib/api/client';
	import { subscriptionEditPath } from '$lib/android/form-routes';
	import BackLink from '$lib/components/BackLink.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import Select from '$lib/components/Select.svelte';
	import { formatAPIOperationDateTimeForDisplay } from '$lib/dates';
	import { toCents } from '$lib/money';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { accountSelectOptions, accountsFromUIMeta } from '$lib/select-options';
	import { user } from '$lib/stores/auth';
	import { requireOnline } from '$lib/offline/require-online';

	let subscription = $state<Subscription | null>(null);
	let accounts = $state<Account[]>([]);
	let items = $state<SubscriptionCandidateTransaction[]>([]);
	let total = $state(0);
	let loading = $state(true);
	let searching = $state(false);
	let attaching = $state(false);
	let loadError = $state<string | null>(null);
	let selected = $state<Record<string, boolean>>({});

	let mode = $state<'auto' | 'all'>('auto');
	let nextAfterAttach = $state<'edit' | null>(null);
	let q = $state('');
	let accountId = $state('');
	let amountMin = $state('');
	let amountMax = $state('');
	let dateFrom = $state('');
	let dateTo = $state('');
	let minScore = $state('');
	let refDescription = $state('');

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');
	const subscriptionId = $derived($page.params.id ?? '');
	const accountOptions = $derived([{ value: '', label: '—' }, ...accountSelectOptions(accounts)]);
	const selectedIds = $derived(
		Object.entries(selected)
			.filter(([, v]) => v)
			.map(([id]) => id)
	);
	const allSelected = $derived(items.length > 0 && items.every((item) => selected[item.id]));

	onMount(() => {
		const initialMode = $page.url.searchParams.get('mode');
		if (initialMode === 'all' || initialMode === 'auto') mode = initialMode;
		if ($page.url.searchParams.get('next') === 'edit') nextAfterAttach = 'edit';
		if (!requireOnline()) {
			loading = false;
			loadError = $_('offline.onlineOnly');
			return;
		}
		void loadMeta();
	});

	async function loadMeta() {
		loading = true;
		try {
			const [subs, meta] = await Promise.all([listSubscriptions(), getUIMeta()]);
			subscription = subs.find((item) => item.id === subscriptionId) ?? null;
			accounts = accountsFromUIMeta(
				meta.accounts.filter((acc) => acc.status === 'active'),
				meta.banks
			) as Account[];
			if (subscription) {
				accountId = subscription.account_id;
				refDescription = subscription.description ?? subscription.name;
				q = subscription.name;
			}
			await search(mode);
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: items.length > 0 });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	function buildParams(nextMode: 'auto' | 'all'): Record<string, string> {
		const params: Record<string, string> = { mode: nextMode, limit: '100', offset: '0' };
		if (q.trim()) params.q = q.trim();
		if (accountId) params.account_id = accountId;
		if (amountMin.trim()) {
			try {
				params.amount_min = String(toCents(amountMin));
			} catch {
				/* ignore invalid */
			}
		}
		if (amountMax.trim()) {
			try {
				params.amount_max = String(toCents(amountMax));
			} catch {
				/* ignore invalid */
			}
		}
		if (dateFrom) params.date_from = dateFrom.slice(0, 10);
		if (dateTo) params.date_to = dateTo.slice(0, 10);
		if (minScore.trim()) params.min_score = minScore.trim();
		if (refDescription.trim()) params.ref_description = refDescription.trim();
		return params;
	}

	async function search(nextMode: 'auto' | 'all' = mode) {
		if (!subscriptionId) return;
		if (!requireOnline()) return;
		mode = nextMode;
		searching = true;
		try {
			const result = await listSubscriptionCandidateTransactions(
				subscriptionId,
				buildParams(nextMode)
			);
			items = result.items;
			total = result.total;
			selected = {};
		} catch (err) {
			toast.fromError(err);
		} finally {
			searching = false;
		}
	}

	function toggleAll() {
		if (allSelected) {
			selected = {};
			return;
		}
		const next: Record<string, boolean> = {};
		for (const item of items) next[item.id] = true;
		selected = next;
	}

	async function attachSelected() {
		if (!subscriptionId || selectedIds.length === 0) return;
		if (!requireOnline()) return;
		attaching = true;
		try {
			const res = await attachSubscriptionTransactions(subscriptionId, selectedIds);
			toast(
				$_('subscriptions.find.attached', {
					values: { count: res.attached_count }
				})
			);
			if (nextAfterAttach === 'edit') {
				await goto(resolve(subscriptionEditPath(subscriptionId, '/subscriptions')));
				return;
			}
			await search(mode);
		} catch (err) {
			toast.fromError(err);
		} finally {
			attaching = false;
		}
	}
</script>

<svelte:head>
	<title>
		{$_('subscriptions.find.title')} — {subscription?.name ?? $_('subscriptions.title')} — {$_(
			'app.title'
		)}
	</title>
</svelte:head>

<div class="space-y-5">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/subscriptions', label: $_('subscriptions.title') },
			{ href: '/subscriptions', label: $_('subscriptions.find.title') }
		]}
	/>

	<SectionHeader
		title={subscription
			? `${$_('subscriptions.find.title')}: ${subscription.name}`
			: $_('subscriptions.find.title')}
	/>

	<PageLoadGate {loading} error={loadError} onretry={() => void loadMeta()} inline>
		<div class="card space-y-4">
			<div class="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
				<div>
					<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="sub-find-q">
						{$_('subscriptions.find.q')}
					</label>
					<input id="sub-find-q" class="input w-full" bind:value={q} />
				</div>
				<Select
					label={$_('transactions.field.account')}
					bind:value={accountId}
					options={accountOptions}
					usePortal
				/>
				<div>
					<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="sub-find-ref">
						{$_('subscriptions.find.refDescription')}
					</label>
					<input id="sub-find-ref" class="input w-full" bind:value={refDescription} />
				</div>
				<div>
					<label
						class="mb-1 block text-sm"
						style:color="var(--text-muted)"
						for="sub-find-amount-min"
					>
						{$_('subscriptions.find.amountMin')}
					</label>
					<MoneyInput id="sub-find-amount-min" bind:value={amountMin} />
				</div>
				<div>
					<label
						class="mb-1 block text-sm"
						style:color="var(--text-muted)"
						for="sub-find-amount-max"
					>
						{$_('subscriptions.find.amountMax')}
					</label>
					<MoneyInput id="sub-find-amount-max" bind:value={amountMax} />
				</div>
				<div>
					<label
						class="mb-1 block text-sm"
						style:color="var(--text-muted)"
						for="sub-find-min-score"
					>
						{$_('subscriptions.find.minScore')}
					</label>
					<input
						id="sub-find-min-score"
						class="input w-full"
						type="number"
						min="0"
						bind:value={minScore}
					/>
				</div>
				<div>
					<DateTimePicker
						id="sub-find-date-from"
						label={$_('subscriptions.find.dateFrom')}
						bind:value={dateFrom}
						{...dateOnlyPicker}
						usePortal
					/>
				</div>
				<div>
					<DateTimePicker
						id="sub-find-date-to"
						label={$_('subscriptions.find.dateTo')}
						bind:value={dateTo}
						{...dateOnlyPicker}
						usePortal
					/>
				</div>
			</div>

			<div class="flex flex-wrap gap-2">
				<button
					type="button"
					class={mode === 'auto' ? 'btn-primary' : 'btn-ghost'}
					disabled={searching}
					onclick={() => void search('auto')}
				>
					{$_('subscriptions.find.auto')}
				</button>
				<button
					type="button"
					class={mode === 'all' ? 'btn-primary' : 'btn-ghost'}
					disabled={searching}
					onclick={() => void search('all')}
				>
					{$_('subscriptions.find.all')}
				</button>
			</div>
		</div>

		<div class="flex flex-wrap items-center justify-between gap-2">
			<label class="inline-flex items-center gap-2 text-sm">
				<input
					type="checkbox"
					checked={allSelected}
					onchange={toggleAll}
					disabled={items.length === 0}
				/>
				{$_('subscriptions.find.selectAll')}
				<span style:color="var(--text-muted)">({total})</span>
			</label>
			<button
				type="button"
				class="btn-primary"
				disabled={attaching || selectedIds.length === 0}
				onclick={() => void attachSelected()}
			>
				{attaching
					? $_('common.loading')
					: $_('subscriptions.find.attach', { values: { count: selectedIds.length } })}
			</button>
		</div>

		{#if searching && items.length === 0}
			<EmptyStateCard message={$_('common.loading')} ariaBusy />
		{:else if items.length === 0}
			<EmptyStateCard message={$_('subscriptions.find.empty')} />
		{:else}
			<div class="card md:overflow-x-auto" class:opacity-60={searching}>
				<div class="hidden md:block">
					<table class="w-full text-left text-sm">
						<thead>
							<tr style:color="var(--text-muted)">
								<th class="p-3 w-10"></th>
								<th class="p-3">{$_('transactions.col.date')}</th>
								<th class="p-3">{$_('transactions.col.account')}</th>
								<th class="p-3">{$_('transactions.col.amount')}</th>
								<th class="p-3">{$_('transactions.col.description')}</th>
								<th class="p-3">{$_('subscriptions.find.score')}</th>
							</tr>
						</thead>
						<tbody>
							{#each items as item (item.id)}
								<tr class="border-t" style:border-color="var(--border)">
									<td class="p-3">
										<input
											type="checkbox"
											checked={!!selected[item.id]}
											onchange={() => {
												selected = { ...selected, [item.id]: !selected[item.id] };
											}}
										/>
									</td>
									<td class="p-3 whitespace-nowrap">
										{formatAPIOperationDateTimeForDisplay(item.transaction_date, tz)}
									</td>
									<td class="p-3">{item.account_name}</td>
									<td class="p-3 tabular-nums">
										<MoneyDisplay value={item.amount_display} {currency} class="" />
									</td>
									<td class="p-3" style:color="var(--text-muted)">
										{item.description ?? ''}
										{#if item.match_reasons?.length}
											<div class="text-xs">{item.match_reasons.join(', ')}</div>
										{/if}
									</td>
									<td class="p-3 tabular-nums">{item.score}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<div class="space-y-3 p-3 md:hidden">
					{#each items as item (item.id)}
						<article class="rounded-xl border p-4" style:border-color="var(--border)">
							<label class="flex items-start gap-3">
								<input
									class="mt-1"
									type="checkbox"
									checked={!!selected[item.id]}
									onchange={() => {
										selected = { ...selected, [item.id]: !selected[item.id] };
									}}
								/>
								<div class="min-w-0 flex-1">
									<div class="flex items-start justify-between gap-2">
										<p class="text-sm" style:color="var(--text-muted)">
											{formatAPIOperationDateTimeForDisplay(item.transaction_date, tz)}
										</p>
										<p class="font-semibold tabular-nums">
											<MoneyDisplay value={item.amount_display} {currency} class="" />
										</p>
									</div>
									<p class="mt-1 text-sm font-medium">{item.account_name}</p>
									{#if item.description}
										<p class="mt-1 text-sm" style:color="var(--text-muted)">{item.description}</p>
									{/if}
									<p class="mt-1 text-xs" style:color="var(--text-muted)">
										{$_('subscriptions.find.score')}: {item.score}
										{#if item.match_reasons?.length}
											· {item.match_reasons.join(', ')}
										{/if}
									</p>
								</div>
							</label>
						</article>
					{/each}
				</div>
			</div>
		{/if}
	</PageLoadGate>
</div>
