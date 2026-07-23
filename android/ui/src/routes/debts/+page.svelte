<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { getDebtsSummary, listDebts, type Debt, type DebtsSummary } from '$lib/api/client';
	import { deleteDebt } from '$lib/offline/debts-api';
	import { debtNewPath, debtSettlePath } from '$lib/android/form-routes';
	import BackLink from '$lib/components/BackLink.svelte';
	import DebtList from '$lib/components/DebtList.svelte';
	import DebtSummaryCard from '$lib/components/DebtSummaryCard.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import TransactionContextStats from '$lib/components/TransactionContextStats.svelte';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import {
		refCacheReady,
		refCacheReadyAny,
		refCacheTick,
		refCacheUpdate
	} from '$lib/offline/ref-cache';
	import { refCachePathMatches } from '$lib/offline/ref-cache-watch';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { assignIfChanged } from '$lib/state-utils';
	import { reportPageLoadFailure } from '$lib/page-load';

	const debtsListPath = (settled: 'active' | 'settled') =>
		`/api/v1/debts?settled=${settled === 'active' ? 'false' : 'true'}`;

	let tab = $state<'active' | 'settled'>('active');
	let debts = $state<Debt[]>([]);
	let summary = $state<DebtsSummary | null>(null);
	let loading = $state(
		!refCacheReady('/api/v1/debts/summary') && !refCacheReady(debtsListPath('active'))
	);
	let filterLoading = $state(false);
	let loadError = $state<string | null>(null);
	let ready = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	onMount(() => void load());

	$effect(() => {
		const tick = $refCacheTick;
		const refresh = $dataRefreshTick;
		if ((tick === 0 && refresh === 0) || !ready) return;
		void load({ background: true });
	});

	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || !ready) return;
		const listPath = debtsListPath(tab);
		if (refCachePathMatches(update.path, [listPath, '/api/v1/debts/summary'])) {
			void load({ background: true });
		}
	});

	async function load(opts: { tabChange?: boolean; background?: boolean } = {}) {
		const listPath = debtsListPath(tab);
		if (opts?.tabChange) {
			if (!opts.background && !refCacheReady(listPath)) filterLoading = true;
		} else if (!opts?.background && !refCacheReadyAny(['/api/v1/debts/summary', listPath])) {
			loading = true;
		}
		try {
			const settled = tab === 'active' ? 'false' : 'true';
			const [summaryData, list] = await Promise.all([getDebtsSummary(), listDebts({ settled })]);
			summary = opts.background ? assignIfChanged(summary, summaryData) : summaryData;
			debts = opts.background ? assignIfChanged(debts, list) : list;
			loadError = null;
			ready = true;
		} catch (err) {
			const msg = reportPageLoadFailure(err, {
				background: opts.background,
				hasData: debts.length > 0
			});
			if (msg) loadError = msg;
		} finally {
			loading = false;
			filterLoading = false;
		}
	}

	async function switchTab(next: 'active' | 'settled') {
		if (next === tab) return;
		tab = next;
		await load({ tabChange: true });
	}

	function openDebtForm(direction: 'lent' | 'borrowed') {
		void goto(resolve(debtNewPath({ direction, from: '/debts' })));
	}

	function openSettle(d: Debt) {
		void goto(resolve(debtSettlePath(d.id, '/debts')));
	}

	async function remove(d: Debt) {
		const ok = await confirm({
			message: $_('debts.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		await deleteDebt(d.id);
		toast($_('common.deleted'));
		debts = debts.filter((row) => row.id !== d.id);
	}
</script>

<svelte:head>
	<title>{$_('debts.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-4">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/debts', label: $_('debts.title') }
		]}
	/>

	<SectionHeader title={$_('debts.title')}>
		{#snippet actions()}
			<div class="btn-pair-row">
				<button type="button" class="btn-primary" onclick={() => openDebtForm('lent')}>
					{$_('debts.action.lend')}
				</button>
				<button type="button" class="btn-ghost" onclick={() => openDebtForm('borrowed')}>
					{$_('debts.action.borrow')}
				</button>
			</div>
		{/snippet}
	</SectionHeader>

	<DebtSummaryCard iOwe={summary?.i_owe ?? 0} owedToMe={summary?.owed_to_me ?? 0} />
	<TransactionContextStats params={{ debts: '1' }} />

	<PageTabs
		active={tab}
		tabs={[
			{ id: 'active', label: $_('debts.tab.active') },
			{ id: 'settled', label: $_('debts.tab.settled') }
		]}
		onchange={(next) => void switchTab(next as 'active' | 'settled')}
	/>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if debts.length === 0 && filterLoading}
			<EmptyStateCard message={$_('common.loading')} ariaBusy />
		{:else if debts.length === 0}
			<EmptyStateCard message={tab === 'settled' ? $_('debts.empty.settled') : $_('debts.empty')} />
		{:else}
			<div class="relative card md:overflow-x-auto" class:opacity-60={filterLoading}>
				{#if filterLoading}
					<p
						class="pointer-events-none absolute inset-x-0 top-0 z-10 py-2 text-center text-sm"
						style:color="var(--text-muted)"
					>
						{$_('common.loading')}
					</p>
				{/if}
				<DebtList {debts} {tz} {currency} onsettle={openSettle} ondelete={(d) => void remove(d)} />
			</div>
		{/if}
	</PageLoadGate>
</div>
