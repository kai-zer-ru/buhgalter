<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		addCreditPayment,
		listBanks,
		listCredits,
		type Bank,
		type Credit
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import CreditForm from '$lib/components/CreditForm.svelte';
	import CreditList from '$lib/components/CreditList.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import PullToRefresh from '$lib/components/PullToRefresh.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let tab = $state<'active' | 'closed'>('active');
	let credits = $state<Credit[]>([]);
	let banks = $state<Bank[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let filterLoading = $state(false);
	let formOpen = $state(false);
	let attachTxId = $state<string | null>(null);
	let attaching = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	onMount(() => void load());

	$effect(() => {
		const txID = $page.url.searchParams.get('attach_tx');
		attachTxId = txID || null;
		if (txID && tab !== 'active') {
			tab = 'active';
			void load({ tabChange: true });
		}
	});

	async function load(opts: { tabChange?: boolean; silent?: boolean } = {}) {
		if (!opts.silent) {
			if (opts.tabChange) filterLoading = true;
			else loading = true;
		}
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
			const msg = reportPageLoadFailure(err, {
				silent: opts.silent,
				hasData: credits.length > 0
			});
			if (msg) loadError = msg;
		} finally {
			if (!opts.silent) {
				loading = false;
				filterLoading = false;
			}
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

<PullToRefresh onRefresh={() => load({ silent: true })} disabled={formOpen}>
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

		{#if attachTxId}
			<section class="space-y-3">
				<GroupHeading
					title={$_('credits.selectToAttach')}
					hint={$_('credits.selectToAttach_hint')}
				/>
				{#if credits.length === 0}
					<p class="card text-sm" style:color="var(--text-muted)">{$_('credits.empty')}</p>
				{:else}
					<ul class="card list-divided p-0 sm:p-0">
						{#each credits as item (item.id)}
							<li>
								<button
									type="button"
									class="flex w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors hover:bg-[color-mix(in_srgb,var(--border)_30%,transparent)]"
									disabled={attaching}
									onclick={() => void attachToCredit(item)}
								>
									<span class="min-w-0 truncate font-medium">{creditName(item)}</span>
									{#if item.monthly_payment_display}
										<span class="shrink-0 text-sm tabular-nums" style:color="var(--text-muted)">
											<MoneyDisplay value={item.monthly_payment_display} {currency} class="" />
										</span>
									{/if}
								</button>
							</li>
						{/each}
					</ul>
				{/if}
				<div class="flex justify-end">
					<button type="button" class="btn-ghost" onclick={clearAttach}>
						{$_('common.cancel')}
					</button>
				</div>
			</section>
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
				<div class="relative card overflow-x-auto p-0 sm:p-0" class:opacity-60={filterLoading}>
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
</PullToRefresh>

<CreditForm bind:open={formOpen} onclose={() => (formOpen = false)} onsaved={() => void load()} />
