<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		attachSubscriptionTransactions,
		getSubscriptionsSummary,
		listSubscriptions,
		type Subscription,
		type SubscriptionPeriod,
		type SubscriptionSummary
	} from '$lib/api/client';
	import { deleteSubscription } from '$lib/offline/subscription-api';
	import { refCacheReady, refCacheUpdate } from '$lib/offline/ref-cache';
	import { refCachePathMatches } from '$lib/offline/ref-cache-watch';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { assignIfChanged } from '$lib/state-utils';
	import { requireOnline } from '$lib/offline/require-online';
	import { subscriptionEditPath, subscriptionNewPath } from '$lib/android/form-routes';
	import BackLink from '$lib/components/BackLink.svelte';
	import CategoryIcon from '$lib/components/CategoryIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import { confirm } from '$lib/confirm';
	import { formatAPIOperationDateTimeForDisplay } from '$lib/dates';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	const SUBSCRIPTIONS_PATH = '/api/v1/subscriptions';
	const LIST_FROM = '/subscriptions';

	let items = $state<Subscription[]>([]);
	let summary = $state<SubscriptionSummary | null>(null);
	let loading = $state(!refCacheReady(SUBSCRIPTIONS_PATH));
	let loadError = $state<string | null>(null);
	let attachTxId = $state<string | null>(null);
	let attaching = $state(false);
	let ready = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	onMount(() => {
		void loadAll();
	});

	$effect(() => {
		const tick = $dataRefreshTick;
		if (tick === 0 || !ready) return;
		void loadAll({ background: true });
	});

	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || !ready) return;
		if (refCachePathMatches(update.path, SUBSCRIPTIONS_PATH)) {
			void loadAll({ background: true });
		}
	});

	async function loadAll(opts: { background?: boolean } = {}) {
		if (!opts.background && !refCacheReady(SUBSCRIPTIONS_PATH)) loading = true;
		try {
			const subs = await listSubscriptions();
			items = opts.background ? assignIfChanged(items, subs) : subs;
			try {
				const sum = await getSubscriptionsSummary({ upcoming_days: 14 });
				summary = opts.background ? assignIfChanged(summary, sum) : sum;
			} catch {
				if (!opts.background && !summary) summary = null;
			}
			if (!opts.background) {
				await handleFromTxQuery();
				await handleAttachQuery();
				await handleEditQuery();
			}
			loadError = null;
			ready = true;
		} catch (err) {
			const msg = reportPageLoadFailure(err, {
				background: opts.background,
				hasData: items.length > 0
			});
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	async function handleFromTxQuery() {
		const txID = $page.url.searchParams.get('from_tx');
		if (!txID) return;
		await goto(resolve(subscriptionNewPath({ from: LIST_FROM, fromTxId: txID })), {
			replaceState: true,
			noScroll: true
		});
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
		await goto(resolve(subscriptionEditPath(id, LIST_FROM)), {
			replaceState: true,
			noScroll: true
		});
	}

	async function attachToSubscription(item: Subscription) {
		if (!attachTxId) return;
		if (!requireOnline()) return;
		attaching = true;
		try {
			await attachSubscriptionTransactions(item.id, [attachTxId]);
			toast($_('subscriptions.attached'));
			attachTxId = null;
			await goto(resolve(subscriptionEditPath(item.id, LIST_FROM)));
		} catch (err) {
			toast.fromError(err);
		} finally {
			attaching = false;
		}
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
		} catch (err) {
			toast.fromError(err);
		}
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
				onclick: () => {
					if (!requireOnline()) return;
					void goto(resolve(`/subscriptions/${encodeURIComponent(item.id)}/find-transactions`));
				}
			},
			{
				icon: 'edit',
				label: $_('common.edit'),
				onclick: () => void goto(resolve(subscriptionEditPath(item.id, LIST_FROM)))
			},
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void remove(item)
			}
		];
	}
</script>

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
			<button
				type="button"
				class="btn-primary shrink-0"
				onclick={() => void goto(resolve(subscriptionNewPath({ from: LIST_FROM })))}
			>
				{$_('subscriptions.add')}
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

	<PageLoadGate {loading} error={loadError} onretry={() => void loadAll()} inline>
		{#if items.length === 0}
			<EmptyStateCard message={$_('subscriptions.empty')} />
		{:else}
			<div class="card space-y-3 p-3">
				{#each items as item (item.id)}
					<article class="rounded-xl border p-4" style:border-color="var(--border)">
						<div class="flex items-start justify-between gap-3">
							<div class="flex min-w-0 items-start gap-2">
								<CategoryIcon icon={item.icon || 'default'} size={28} />
								<div class="min-w-0">
									<p class="font-medium">
										{item.name}
										{#if !item.active}
											<span class="text-xs font-normal" style:color="var(--text-muted)">
												({$_('subscriptions.inactive')})
											</span>
										{/if}
									</p>
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
				{/each}
			</div>
		{/if}
	</PageLoadGate>
</div>
