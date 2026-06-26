<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { ApiError, listAccounts, setPrimaryAccount, type Account } from '$lib/api/client';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import { formatBalance } from '$lib/finance';
	import { user } from '$lib/stores/auth';

	let accounts = $state<Account[]>([]);
	let loading = $state(true);
	let error = $state('');
	let primarySavingId = $state('');

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

	async function makePrimary(id: string) {
		if (!id || primarySavingId) return;
		primarySavingId = id;
		error = '';
		try {
			const updated = await setPrimaryAccount(id);
			accounts = accounts.map((acc) => ({
				...acc,
				is_primary: acc.id === updated.id
			}));
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			primarySavingId = '';
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
				<div class="card">
					<div class="flex items-center gap-3">
						<a
							href={resolve(`/accounts/${acc.id}`)}
							class="flex min-w-0 flex-1 items-center gap-4 transition hover:opacity-90"
						>
							<AccountIcon type={acc.type} bankIcon={acc.bank_icon} size={48} />
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-1">
									<p class="truncate font-medium">{acc.name}</p>
									{#if acc.is_primary}
										<span
											class="shrink-0"
											style:color="var(--primary)"
											title={$_('accounts.primary.badge')}
											aria-label={$_('accounts.primary.badge')}
										>
											<svg
												aria-hidden="true"
												class="h-4 w-4"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
												stroke-width="2"
											>
												<path d="M20 6 9 17l-5-5" />
											</svg>
										</span>
									{/if}
								</div>
								<p class="mt-1 text-xl font-semibold tabular-nums">
									{formatBalance(acc.balance_display, $user?.currency ?? 'RUB')}
								</p>
							</div>
						</a>
						{#if !acc.is_primary}
							<IconButton
								icon="save"
								label={$_('accounts.primary.set')}
								disabled={primarySavingId === acc.id}
								onclick={() => void makePrimary(acc.id)}
							/>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
