<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { ApiError, listCredits, type Credit } from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import CreditForm from '$lib/components/CreditForm.svelte';
	import CreditList from '$lib/components/CreditList.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import { user } from '$lib/stores/auth';

	let tab = $state<'active' | 'closed'>('active');
	let credits = $state<Credit[]>([]);
	let loading = $state(true);
	let filterLoading = $state(false);
	let error = $state('');
	let formOpen = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	onMount(() => void load());

	async function load(opts: { tabChange?: boolean } = {}) {
		if (opts?.tabChange) filterLoading = true;
		else loading = true;
		error = '';
		try {
			const status = tab === 'active' ? 'active' : 'closed';
			credits = await listCredits({ status });
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
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
</script>

<svelte:head>
	<title>{$_('credits.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-4">
	<BackLink href="/" label={$_('dashboard.title')} />

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

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if credits.length === 0 && filterLoading}
		<EmptyStateCard message={$_('common.loading')} ariaBusy />
	{:else if credits.length === 0}
		<FormFeedback {error} />
		<EmptyStateCard message={tab === 'closed' ? $_('credits.empty.closed') : $_('credits.empty')} />
	{:else}
		<FormFeedback {error} />
		<div class="relative card md:overflow-x-auto" class:opacity-60={filterLoading}>
			{#if filterLoading}
				<p
					class="pointer-events-none absolute inset-x-0 top-0 z-10 py-2 text-center text-sm"
					style:color="var(--text-muted)"
				>
					{$_('common.loading')}
				</p>
			{/if}
			<CreditList {credits} {tz} {currency} nameFor={creditName} />
		</div>
	{/if}
</div>

<CreditForm bind:open={formOpen} onclose={() => (formOpen = false)} onsaved={() => void load()} />
