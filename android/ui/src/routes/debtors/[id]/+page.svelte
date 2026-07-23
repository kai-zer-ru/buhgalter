<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		deleteTransaction,
		getDebtor,
		type Debt,
		type DebtorDetail,
		type Transaction
	} from '$lib/api/client';
	import { deleteDebt } from '$lib/offline/debts-api';
	import { debtNewPath, debtSettlePath } from '$lib/android/form-routes';
	import BackLink from '$lib/components/BackLink.svelte';
	import DebtList from '$lib/components/DebtList.svelte';
	import DebtSummaryCard from '$lib/components/DebtSummaryCard.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import CollapsibleSection from '$lib/components/CollapsibleSection.svelte';
	import TransactionContextStats from '$lib/components/TransactionContextStats.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { confirm } from '$lib/confirm';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let detail = $state<DebtorDetail | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let tab = $state<'active' | 'settled'>('active');

	const debtorId = $derived($page.params.id ?? '');
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	const activeDebts = $derived(detail?.debts.filter((d) => !d.is_settled) ?? []);
	const hasActiveLent = $derived(activeDebts.some((d) => d.direction === 'lent'));
	const hasActiveBorrowed = $derived(activeDebts.some((d) => d.direction === 'borrowed'));
	const hasAnyActiveDebt = $derived(activeDebts.length > 0);

	const relatedTransactions = $derived((detail?.transactions ?? []) as Transaction[]);

	const visibleDebts = $derived(
		detail?.debts.filter((d) => (tab === 'active' ? !d.is_settled : d.is_settled)) ?? []
	);

	onMount(() => void load());

	async function load() {
		if (!debtorId) return;
		loading = true;
		try {
			detail = await getDebtor(debtorId);
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: !!detail });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	function openDebtForm(direction: 'lent' | 'borrowed') {
		if (!debtorId) return;
		void goto(resolve(debtNewPath({ direction, debtorId, from: `/debtors/${debtorId}` })));
	}

	function openSettle(d: Debt) {
		void goto(resolve(debtSettlePath(d.id, `/debtors/${debtorId}`)));
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
		await load();
	}

	async function removeTx(tx: Transaction) {
		const ok = await confirm({
			message: $_('transactions.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		try {
			await deleteTransaction(tx.id);
			toast($_('common.deleted'));
			await load();
		} catch (err) {
			toast.fromError(err);
		}
	}
</script>

<svelte:head>
	<title>{detail?.name ?? $_('debts.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-4">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/debts', label: $_('debts.title') },
			{ href: '/debts', label: detail?.name ?? $_('debtors.title') }
		]}
	/>

	<PageLoadGate loading={loading && !detail} error={loadError} onretry={() => void load()} inline>
		{#if detail}
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h1 class="text-2xl font-semibold">{detail.name}</h1>
				<div class="btn-pair-row sm:max-w-md">
					{#if hasAnyActiveDebt}
						{#if hasActiveLent}
							<button type="button" class="btn-primary" onclick={() => openDebtForm('lent')}>
								{$_('debts.action.lendMore')}
							</button>
						{/if}
						{#if hasActiveBorrowed}
							<button
								type="button"
								class={hasActiveLent ? 'btn-ghost' : 'btn-primary'}
								onclick={() => openDebtForm('borrowed')}
							>
								{$_('debts.action.borrowMore')}
							</button>
						{/if}
					{:else}
						<button type="button" class="btn-primary" onclick={() => openDebtForm('lent')}>
							{$_('debts.action.lend')}
						</button>
						<button type="button" class="btn-ghost" onclick={() => openDebtForm('borrowed')}>
							{$_('debts.action.borrow')}
						</button>
					{/if}
				</div>
			</div>

			<DebtSummaryCard iOwe={detail.i_owe} owedToMe={detail.owed_to_me} />
			<TransactionContextStats params={{ debtor_id: detail.id }} />

			<section>
				<h2 class="mb-3 text-lg font-medium">{$_('debts.title')}</h2>

				<PageTabs
					active={tab}
					tabs={[
						{ id: 'active', label: $_('debts.tab.active') },
						{ id: 'settled', label: $_('debts.tab.settled') }
					]}
					onchange={(next) => (tab = next as 'active' | 'settled')}
				/>

				<div class="mt-4">
					{#if visibleDebts.length === 0}
						<EmptyStateCard
							message={tab === 'settled' ? $_('debts.empty.settled') : $_('debts.empty')}
						/>
					{:else}
						<div class="card md:overflow-x-auto">
							<DebtList
								debts={visibleDebts}
								{tz}
								{currency}
								showDebtor={false}
								onsettle={openSettle}
								ondelete={(d) => void remove(d)}
							/>
						</div>
					{/if}
				</div>
			</section>

			{#if detail.transactions.length > 0}
				<CollapsibleSection
					label={$_('debts.recentTransactions')}
					count={detail.transactions.length}
				>
					<div class="card md:overflow-x-auto">
						<TransactionList
							transactions={relatedTransactions}
							siblings={relatedTransactions}
							{tz}
							emptyMessage={$_('transactions.empty')}
							showDescription
							showAmountSign
							showCategory={false}
							showDelete
							ondelete={(tx) => void removeTx(tx)}
						/>
					</div>
				</CollapsibleSection>
			{/if}
		{/if}
	</PageLoadGate>
</div>
