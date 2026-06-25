<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { ApiError, listAccounts, type Account } from '$lib/api/client';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import { formatBalance } from '$lib/finance';
	import { user } from '$lib/stores/auth';

	let accounts = $state<Account[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			accounts = await listAccounts();
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{$_('accounts.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-semibold">{$_('accounts.title')}</h1>
		<a href={resolve('/accounts/new')} class="btn-primary">
			{$_('accounts.new')}
		</a>
	</div>

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if error}
		<p style:color="var(--danger)">{error}</p>
	{:else if accounts.length === 0}
		<EmptyStateCard message={$_('accounts.empty')} />
	{:else}
		<div class="grid gap-4 sm:grid-cols-2">
			{#each accounts as acc (acc.id)}
				<a
					href={resolve(`/accounts/${acc.id}`)}
					class="card flex items-center gap-4 transition hover:opacity-90"
				>
					<AccountIcon type={acc.type} bankIcon={acc.bank_icon} size={48} />
					<div class="min-w-0 flex-1">
						<p class="truncate font-medium">{acc.name}</p>
						<p class="mt-1 text-xl font-semibold tabular-nums">
							{formatBalance(acc.balance_display, $user?.currency ?? 'RUB')}
						</p>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
