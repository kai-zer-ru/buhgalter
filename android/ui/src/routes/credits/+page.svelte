<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { listBanks, listCredits, type Bank, type Credit } from '$lib/api/client';
	import { addCreditPayment } from '$lib/offline/credits-api';
	import { creditNewPath } from '$lib/android/form-routes';
	import BackLink from '$lib/components/BackLink.svelte';
	import CreditList from '$lib/components/CreditList.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import { user } from '$lib/stores/auth';
	import { refCacheReady, refCacheUpdate } from '$lib/offline/ref-cache';
	import { refCachePathMatches } from '$lib/offline/ref-cache-watch';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { assignIfChanged } from '$lib/state-utils';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { requireOnline } from '$lib/offline/require-online';
	import { toast } from '$lib/toast';

	const creditsPath = (status: 'active' | 'closed') => `/api/v1/credits?status=${status}`;

	let tab = $state<'active' | 'closed'>('active');
	let credits = $state<Credit[]>([]);
	let banks = $state<Bank[]>([]);
	let loading = $state(!refCacheReady(creditsPath('active')));
	let filterLoading = $state(false);
	let loadError = $state<string | null>(null);
	let ready = $state(false);
	let attachTxId = $state<string | null>(null);
	let attaching = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	onMount(() => void load());

	$effect(() => {
		const refresh = $dataRefreshTick;
		if (refresh === 0 || !ready) return;
		void load({ background: true });
	});

	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || !ready) return;
		const listPath = creditsPath(tab);
		if (refCachePathMatches(update.path, [listPath, '/api/v1/banks'])) {
			void load({ background: true });
		}
	});

	$effect(() => {
		const txID = $page.url.searchParams.get('attach_tx');
		if (!txID) {
			attachTxId = null;
			return;
		}
		attachTxId = txID;
		if (tab !== 'active') {
			tab = 'active';
			void load({ tabChange: true });
		}
	});

	async function load(opts: { tabChange?: boolean; background?: boolean } = {}) {
		const status = tab === 'active' ? 'active' : 'closed';
		const listPath = creditsPath(status as 'active' | 'closed');
		if (opts?.tabChange) {
			if (!opts.background && !refCacheReady(listPath)) filterLoading = true;
		} else if (!opts?.background && !refCacheReady(listPath)) {
			loading = true;
		}
		try {
			const [creditsList, banksList] = await Promise.all([listCredits({ status }), listBanks()]);
			const sorted = [...creditsList].sort((a, b) => {
				const byCreated = b.created_at.localeCompare(a.created_at);
				if (byCreated !== 0) return byCreated;
				return b.issue_date.localeCompare(a.issue_date);
			});
			credits = opts.background ? assignIfChanged(credits, sorted) : sorted;
			banks = opts.background ? assignIfChanged(banks, banksList) : banksList;
			loadError = null;
			ready = true;
		} catch (err) {
			const msg = reportPageLoadFailure(err, {
				background: opts.background,
				hasData: credits.length > 0
			});
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
		return c.name?.trim() || $_('credits.unnamed');
	}

	function bankIconFor(c: Credit): string | null {
		if (!c.bank_id) return null;
		return banks.find((item) => item.id === c.bank_id)?.icon_path ?? null;
	}

	async function attachToCredit(item: Credit) {
		if (!attachTxId) return;
		if (!requireOnline()) return;
		attaching = true;
		try {
			await addCreditPayment(item.id, { transaction_id: attachTxId });
			toast($_('credits.attached'));
			attachTxId = null;
			await goto(resolve('/credits'), { replaceState: true });
			await load();
		} catch (err) {
			toast.fromError(err);
		} finally {
			attaching = false;
		}
	}

	function clearAttach() {
		attachTxId = null;
		void goto(resolve('/credits'), { replaceState: true });
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
			<button
				type="button"
				class="btn-primary"
				onclick={() => {
					if (!requireOnline('offline.onlineOnly.creditCreate')) return;
					void goto(resolve(creditNewPath('/credits')));
				}}
			>
				{$_('credits.new')}
			</button>
		{/snippet}
	</SectionHeader>

	{#if attachTxId}
		<div class="card space-y-3">
			<h2 class="text-base font-semibold">{$_('credits.selectToAttach')}</h2>
			{#if credits.length === 0}
				<p class="text-sm" style:color="var(--text-muted)">{$_('credits.empty')}</p>
			{:else}
				<ul class="space-y-2">
					{#each credits as item (item.id)}
						<li>
							<button
								type="button"
								class="btn-ghost w-full justify-start text-left"
								disabled={attaching}
								onclick={() => void attachToCredit(item)}
							>
								<span class="inline-flex items-center gap-2">
									<span class="font-medium">{creditName(item)}</span>
									{#if item.monthly_payment_display}
										<span class="text-sm" style:color="var(--text-muted)">
											<MoneyDisplay value={item.monthly_payment_display} {currency} class="" />
										</span>
									{/if}
								</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
			<button type="button" class="btn-ghost" onclick={clearAttach}>
				{$_('common.cancel')}
			</button>
		</div>
	{/if}

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
