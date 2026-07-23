<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { listBanks, listCredits, type Bank, type Credit } from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import CreditForm from '$lib/components/CreditForm.svelte';
	import CreditList from '$lib/components/CreditList.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { user } from '$lib/stores/auth';

	let tab = $state<'active' | 'closed'>('active');
	let credits = $state<Credit[]>([]);
	let banks = $state<Bank[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let filterLoading = $state(false);
	let formOpen = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	onMount(() => void load());

	async function load(opts: { tabChange?: boolean } = {}) {
		if (opts?.tabChange) filterLoading = true;
		else loading = true;
		try {
			const status = tab === 'active' ? 'active' : 'closed';
			const [creditsList, banksList] = await Promise.all([listCredits({ status }), listBanks()]);
			credits = [...creditsList].sort((a, b) => {
				const byCreated = b.created_at.localeCompare(a.created_at);
				if (byCreated !== 0) return byCreated;
				return b.issue_date.localeCompare(a.issue_date);
			});
			banks = banksList;
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: credits.length > 0 });
			if (msg) loadError = msg;
		} finally {
			loading = false;
			filterLoading = false;
		}
	}

	async function switchTab(next: 'active' | 'closed') {
		if (next === tab) return;
		tab = next;
		await load({ tabChange: true });
	}

	function creditName(c: Credit): string {
		return c.name?.trim() || $_('credits.title');
	}

	function bankIconFor(c: Credit): string | null {
		if (!c.bank_id) return null;
		return banks.find((item) => item.id === c.bank_id)?.icon_path ?? null;
	}
</script>

<svelte:head>
	<title>{$_('credits.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-4">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/credits', label: $_('credits.title') }
		]}
	/>

	<SectionHeader title={$_('credits.title')}>
		{#snippet actions()}
			<button type="button" class="btn-primary" onclick={() => (formOpen = true)}>
				{$_('credits.new')}
			</button>
		{/snippet}
	</SectionHeader>

	<PageTabs
		active={tab}
		tabs={[
			{ id: 'active', label: $_('credits.tab.active') },
			{ id: 'closed', label: $_('credits.tab.closed') }
		]}
		onchange={(next) => void switchTab(next as 'active' | 'closed')}
	/>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if credits.length === 0 && filterLoading}
			<EmptyStateCard message={$_('common.loading')} ariaBusy />
		{:else if credits.length === 0}
			<EmptyStateCard
				message={tab === 'closed' ? $_('credits.empty.closed') : $_('credits.empty')}
			/>
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
				<CreditList {credits} {tz} {currency} nameFor={creditName} {bankIconFor} />
			</div>
		{/if}
	</PageLoadGate>
</div>

<CreditForm bind:open={formOpen} onclose={() => (formOpen = false)} onsaved={() => void load()} />
