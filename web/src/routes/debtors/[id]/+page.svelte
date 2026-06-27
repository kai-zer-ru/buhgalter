<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		deleteDebt,
		getDebtor,
		type Debt,
		type DebtorDetail,
		type Transaction
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import DebtForm from '$lib/components/DebtForm.svelte';
	import DebtList from '$lib/components/DebtList.svelte';
	import DebtSummaryCard from '$lib/components/DebtSummaryCard.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import SettleDebtForm from '$lib/components/SettleDebtForm.svelte';
	import TransactionContextStats from '$lib/components/TransactionContextStats.svelte';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let detail = $state<DebtorDetail | null>(null);
	let loading = $state(true);
	let error = $state('');
	let formOpen = $state(false);
	let formDirection = $state<'lent' | 'borrowed'>('lent');
	let tab = $state<'active' | 'settled'>('active');
	let settleOpen = $state(false);
	let settleDebtItem = $state<Debt | null>(null);

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
		error = '';
		try {
			detail = await getDebtor(debtorId);
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}

	function openDebtForm(direction: 'lent' | 'borrowed') {
		formDirection = direction;
		formOpen = true;
	}

	function openSettle(d: Debt) {
		settleDebtItem = d;
		settleOpen = true;
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

	{#if loading && !detail}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if error && !detail}
		<p style:color="var(--danger)">{error}</p>
	{:else if detail}
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-2xl font-semibold">{detail.name}</h1>
			<div class="flex flex-wrap gap-2">
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
			<section>
				<h2 class="mb-3 text-lg font-medium">{$_('debts.relatedTransactions')}</h2>
				<div class="card md:overflow-x-auto">
					<TransactionList
						transactions={relatedTransactions}
						siblings={relatedTransactions}
						{tz}
						emptyMessage={$_('transactions.empty')}
						showDescription
					/>
				</div>
			</section>
		{/if}
	{/if}
</div>

{#if detail}
	<DebtForm
		bind:open={formOpen}
		debtorId={detail.id}
		debtorName={detail.name}
		defaultDirection={formDirection}
		onclose={() => (formOpen = false)}
		onsaved={load}
	/>
	<SettleDebtForm
		bind:open={settleOpen}
		bind:debt={settleDebtItem}
		onclose={() => {
			settleOpen = false;
			settleDebtItem = null;
		}}
		onsaved={load}
	/>
{/if}
