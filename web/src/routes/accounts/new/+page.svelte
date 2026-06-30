<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { ApiError, createAccount, listBanks, type Bank } from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import { toast } from '$lib/toast';
	import { bankIconUrl } from '$lib/finance';
	import { toAPIAmount } from '$lib/money';

	let name = $state('');
	let type = $state<'cash' | 'bank'>('cash');
	let bankId = $state('');
	let bankSearch = $state('');
	let initialBalance = $state('');
	let banks = $state<Bank[]>([]);
	let loading = $state(false);
	let error = $state('');

	const filteredBanks = $derived(
		banks.filter((b) => b.name.toLowerCase().includes(bankSearch.toLowerCase()))
	);

	onMount(async () => {
		try {
			banks = await listBanks();
		} catch {
			error = $_('common.error');
		}
	});

	async function submit(e: Event) {
		e.preventDefault();
		loading = true;
		error = '';
		try {
			const acc = await createAccount({
				name,
				type,
				bank_id: type === 'bank' ? bankId : undefined,
				initial_balance: toAPIAmount(initialBalance || '0')
			});
			await goto(resolve(`/accounts/${acc.id}`));
			toast($_('common.saved'));
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}
</script>

<div class="mx-auto max-w-lg space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/accounts', label: $_('accounts.title') },
			{ href: '/accounts/new', label: $_('accounts.new') }
		]}
	/>
	<h1 class="text-2xl font-semibold tracking-tight">{$_('accounts.new')}</h1>

	<form class="card space-y-4" onsubmit={submit}>
		<div>
			<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="name">
				{$_('accounts.field.name')}
			</label>
			<input id="name" class="input w-full" bind:value={name} required maxlength="64" />
		</div>

		<div>
			<span class="mb-2 block text-sm" style:color="var(--text-muted)"
				>{$_('accounts.field.type')}</span
			>
			<div class="flex gap-2">
				<button
					type="button"
					class={type === 'cash' ? 'tab tab-active' : 'tab'}
					onclick={() => (type = 'cash')}
				>
					{$_('accounts.type.cash')}
				</button>
				<button
					type="button"
					class={type === 'bank' ? 'tab tab-active' : 'tab'}
					onclick={() => (type = 'bank')}
				>
					{$_('accounts.type.bank')}
				</button>
			</div>
		</div>

		{#if type === 'bank'}
			<div>
				<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="bank-search">
					{$_('accounts.field.bank')}
				</label>
				<input
					id="bank-search"
					class="input mb-2 w-full"
					placeholder={$_('accounts.bank.search')}
					bind:value={bankSearch}
				/>
				<div
					class="max-h-48 space-y-1 overflow-y-auto rounded-lg border p-2"
					style:border-color="var(--border)"
				>
					{#each filteredBanks as bank (bank.id)}
						<button
							type="button"
							class="flex w-full items-center gap-3 rounded-lg px-2 py-2 text-left transition hover:opacity-80"
							style:background-color={bankId === bank.id
								? 'color-mix(in srgb, var(--primary) 12%, transparent)'
								: 'transparent'}
							onclick={() => (bankId = bank.id)}
						>
							<img
								src={bankIconUrl(bank.icon_path)}
								alt=""
								class="h-8 w-8 rounded-lg"
								width="32"
								height="32"
							/>
							<span>{bank.name}</span>
						</button>
					{/each}
				</div>
			</div>
		{/if}

		<div>
			<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="balance">
				{$_('accounts.field.balance')}
			</label>
			<MoneyInput id="balance" bind:value={initialBalance} />
		</div>

		<div class="flex gap-2 pt-2">
			<button type="submit" class="btn-primary" disabled={loading || (type === 'bank' && !bankId)}>
				{loading ? $_('common.loading') : $_('common.create')}
			</button>
			<a href={resolve('/accounts')} class="btn-ghost">{$_('common.cancel')}</a>
		</div>
		<FormFeedback {error} />
	</form>
</div>
