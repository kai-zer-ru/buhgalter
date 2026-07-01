<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		deleteDebt,
		getDebtsSummary,
		listDebts,
		type Debt,
		type DebtsSummary
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import DebtForm from '$lib/components/DebtForm.svelte';
	import DebtList from '$lib/components/DebtList.svelte';
	import DebtSummaryCard from '$lib/components/DebtSummaryCard.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import TransactionContextStats from '$lib/components/TransactionContextStats.svelte';
	import SettleDebtForm from '$lib/components/SettleDebtForm.svelte';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let tab = $state<'active' | 'settled'>('active');
	let debts = $state<Debt[]>([]);
	let summary = $state<DebtsSummary | null>(null);
	let loading = $state(true);
	let filterLoading = $state(false);
	let formOpen = $state(false);
	let formDirection = $state<'lent' | 'borrowed'>('lent');
	let settleOpen = $state(false);
	let settleDebtItem = $state<Debt | null>(null);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	onMount(() => void load());

	async function load(opts: { tabChange?: boolean } = {}) {
		if (opts?.tabChange) {
			filterLoading = true;
		} else {
			loading = true;
		}
		try {
			const settled = tab === 'active' ? 'false' : 'true';
			const [summaryData, list] = await Promise.all([getDebtsSummary(), listDebts({ settled })]);
			summary = summaryData;
			debts = list;
		} catch (err) {
			toast.fromError(err);
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
			<button type="button" class="btn-primary" onclick={() => openDebtForm('lent')}>
				{$_('debts.action.lend')}
			</button>
			<button type="button" class="btn-ghost" onclick={() => openDebtForm('borrowed')}>
				{$_('debts.action.borrow')}
			</button>
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

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if debts.length === 0 && filterLoading}
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
</div>

<DebtForm
	bind:open={formOpen}
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
