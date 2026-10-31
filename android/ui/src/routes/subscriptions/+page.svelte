<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _, locale } from 'svelte-i18n';
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
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import { confirm } from '$lib/confirm';
	import {
		formatAPIDayMonthTimeForDisplay,
		formatAPIDayMonthTimeShortForDisplay
	} from '$lib/dates';
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

<div class="space-y-6">
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

	<div class="space-y-8">
		{#if attachTxId}
			<section class="space-y-3">
				<GroupHeading
					title={$_('subscriptions.selectToAttach')}
					hint={$_('subscriptions.selectToAttach_hint')}
				/>
				{#if items.length === 0}
					<p class="card text-sm" style:color="var(--text-muted)">{$_('subscriptions.empty')}</p>
				{:else}
					<ul class="card list-divided p-0 sm:p-0">
						{#each items as item (item.id)}
							<li>
								<button
									type="button"
									class="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-[color-mix(in_srgb,var(--border)_30%,transparent)]"
									disabled={attaching}
									onclick={() => void attachToSubscription(item)}
								>
									<CategoryIcon icon={item.icon || 'default'} size={32} />
									<span class="min-w-0 flex-1 truncate font-medium">{item.name}</span>
									<span class="shrink-0 text-sm tabular-nums" style:color="var(--text-muted)">
										<MoneyDisplay value={item.amount_display} {currency} class="" />
									</span>
								</button>
							</li>
						{/each}
					</ul>
				{/if}
				<div class="flex justify-end">
					<button type="button" class="btn-ghost" onclick={() => (attachTxId = null)}>
						{$_('common.cancel')}
					</button>
				</div>
			</section>
		{/if}

		{#if summary}
			<div class="card overflow-hidden p-0 sm:p-0">
				<div class="stat-grid grid-cols-2">
					<div>
						<p class="stat-label">{$_('subscriptions.summary.monthly')}</p>
						<p class="stat-value sm:text-2xl">
							<MoneyDisplay value={summary.monthly_total_display} {currency} class="" />
						</p>
					</div>
					<div>
						<p class="stat-label">{$_('subscriptions.summary.yearly')}</p>
						<p class="stat-value sm:text-2xl">
							<MoneyDisplay value={summary.yearly_total_display} {currency} class="" />
						</p>
					</div>
				</div>
			</div>
			{#if summary.upcoming.length > 0}
				<section class="space-y-3">
					<GroupHeading
						title={$_('subscriptions.upcoming')}
						hint={$_('subscriptions.upcoming_hint')}
					/>
					<ul class="card list-divided p-0 sm:p-0">
						{#each summary.upcoming as item (item.id)}
							<li class="flex items-center gap-3 px-4 py-2.5 text-sm">
								<CategoryIcon icon={item.icon || 'default'} size={32} />
								<span class="min-w-0 flex-1">
									<span class="block truncate font-medium">{item.name}</span>
									<span class="block text-xs" style:color="var(--text-muted)">
										{formatAPIDayMonthTimeForDisplay(item.next_run_at, tz, $locale ?? 'ru')}
									</span>
								</span>
								<span class="shrink-0 font-medium tabular-nums">
									<MoneyDisplay value={item.amount_display} {currency} class="" />
								</span>
							</li>
						{/each}
					</ul>
				</section>
			{/if}
		{/if}

		<PageLoadGate {loading} error={loadError} onretry={() => void loadAll()} inline>
			{#if items.length === 0}
				<EmptyStateCard message={$_('subscriptions.empty')} />
			{:else}
				<section class="space-y-3">
					<GroupHeading
						title={$_('subscriptions.group.all', { values: { count: items.length } })}
						hint={$_('subscriptions.group.all_hint')}
					/>
					<div class="card list-divided p-0 sm:p-0">
						{#each items as item (item.id)}
							<article class="flex items-center gap-3 px-4 py-3">
								<CategoryIcon icon={item.icon || 'default'} size={36} />
								<div class="min-w-0 flex-1">
									<p class="truncate text-sm font-medium">
										{item.name}
										{#if !item.active}
											<span class="text-xs font-normal" style:color="var(--text-muted)">
												({$_('subscriptions.inactive')})
											</span>
										{/if}
									</p>
									<p class="truncate text-xs" style:color="var(--text-muted)">
										{periodLabel(item.period)} · {item.account_name}
									</p>
								</div>
								<div class="shrink-0 text-right">
									<p class="text-sm font-semibold tabular-nums">
										<MoneyDisplay value={item.amount_display} {currency} class="" />
									</p>
									<p class="text-xs" style:color="var(--text-muted)">
										{formatAPIDayMonthTimeShortForDisplay(item.next_run_at, tz, $locale ?? 'ru')}
									</p>
								</div>
								<RowActionsMenu actions={rowActions(item)} />
							</article>
						{/each}
					</div>
				</section>
			{/if}
		</PageLoadGate>
	</div>
</div>
