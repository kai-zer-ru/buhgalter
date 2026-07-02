<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { listAccounts, setPrimaryAccount, type Account } from '$lib/api/client';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let accounts = $state<Account[]>([]);
	let filter = $state<'active' | 'archived' | 'deleted'>('active');
	let loading = $state(true);
	let filterLoading = $state(false);

	async function load(opts: { filterChange?: boolean } = {}) {
		if (opts?.filterChange) {
			filterLoading = true;
		} else {
			loading = true;
		}
		try {
			accounts = await listAccounts(filter);
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
			filterLoading = false;
		}
	}

	onMount(() => {
		void load();
	});

	function setFilter(next: 'active' | 'archived' | 'deleted') {
		if (next === filter) return;
		filter = next;
		void load({ filterChange: true });
	}

	async function makePrimary(id: string) {
		if (accounts.find((a) => a.id === id)?.is_primary) return;
		try {
			await setPrimaryAccount(id);
			accounts = accounts.map((a) => ({ ...a, is_primary: a.id === id }));
		} catch (err) {
			toast.fromError(err);
		}
	}
</script>

<div class="space-y-6">
	<SectionHeader title={$_('accounts.title')}>
		{#snippet actions()}
			<a href={resolve('/accounts/new')} class="btn-primary w-full sm:w-auto">
				{$_('accounts.new')}
			</a>
		{/snippet}
	</SectionHeader>

	<PageTabs
		active={filter}
		tabs={[
			{ id: 'active', label: $_('accounts.filter.active') },
			{ id: 'archived', label: $_('accounts.filter.archived') },
			{ id: 'deleted', label: $_('accounts.filter.deleted') }
		]}
		onchange={(next) => setFilter(next as 'active' | 'archived' | 'deleted')}
	/>

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if accounts.length === 0 && filterLoading}
		<EmptyStateCard message={$_('common.loading')} ariaBusy />
	{:else if accounts.length === 0}
		<EmptyStateCard message={$_('accounts.empty')} />
	{:else}
		<div class="relative grid gap-4 sm:grid-cols-2" class:opacity-60={filterLoading}>
			{#if filterLoading}
				<p
					class="pointer-events-none absolute inset-x-0 top-0 py-2 text-center text-sm"
					style:color="var(--text-muted)"
				>
					{$_('common.loading')}
				</p>
			{/if}
			{#each accounts as acc (acc.id)}
				<div class="card flex items-center gap-3 transition hover:opacity-90 sm:gap-4">
					<a href={resolve(`/accounts/${acc.id}`)} class="flex min-w-0 flex-1 items-center gap-4">
						<AccountIcon type={acc.type} bankIcon={acc.bank_icon} size={48} />
						<div class="min-w-0 flex-1">
							<div class="truncate font-medium">{acc.name}</div>
							<div class="text-sm" style:color="var(--text-muted)">
								{#if acc.type === 'bank' && acc.bank_name}
									{acc.bank_name}
								{:else}
									{$_('accounts.type.cash')}
								{/if}
							</div>
						</div>
					</a>
					{#if filter === 'active'}
						<button
							type="button"
							class="btn-icon btn-ghost shrink-0"
							title={acc.is_primary ? $_('accounts.primary.badge') : $_('accounts.primary.set')}
							aria-pressed={acc.is_primary}
							aria-label={acc.is_primary
								? $_('accounts.primary.badge')
								: $_('accounts.primary.set')}
							style:color={acc.is_primary ? 'var(--primary)' : 'var(--text-muted)'}
							onclick={() => void makePrimary(acc.id)}
						>
							{acc.is_primary ? '★' : '☆'}
						</button>
					{/if}
					<div class="shrink-0 text-right font-semibold tabular-nums">
						<MoneyDisplay
							value={acc.balance_display}
							currency={$user?.currency ?? 'RUB'}
							class=""
						/>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
