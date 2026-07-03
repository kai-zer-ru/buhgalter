<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		createAccount,
		listAccounts,
		listBanks,
		type Account,
		type AccountType,
		type Bank
	} from '$lib/api/client';
	import BackLink from '$lib/components/BackLink.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import { debitAccounts } from '$lib/credit-card';
	import { accountSelectOptions } from '$lib/select-options';
	import { toast } from '$lib/toast';
	import { bankIconUrl } from '$lib/finance';
	import { toAPIAmount } from '$lib/money';

	let name = $state('');
	let type = $state<AccountType>('cash');
	let bankId = $state('');
	let bankSearch = $state('');
	let creditLimit = $state('');
	let initialBalance = $state('');
	let paymentAccountId = $state('');
	let banks = $state<Bank[]>([]);
	let debitAccountList = $state<Account[]>([]);
	let loading = $state(false);

	const filteredBanks = $derived(
		banks.filter((b) => b.name.toLowerCase().includes(bankSearch.toLowerCase()))
	);
	const needsBank = $derived(type === 'bank' || type === 'credit_card');
	const paymentOptions = $derived(accountSelectOptions(debitAccounts(debitAccountList)));

	onMount(async () => {
		try {
			const [bankList, accountList] = await Promise.all([listBanks(), listAccounts()]);
			banks = bankList;
			debitAccountList = accountList;
		} catch (err) {
			toast.fromError(err);
		}
	});

	function applyLimitToBalance() {
		if (creditLimit.trim()) initialBalance = creditLimit;
	}

	async function submit(e: Event) {
		e.preventDefault();
		loading = true;
		try {
			const acc = await createAccount({
				name,
				type,
				bank_id: needsBank ? bankId : undefined,
				initial_balance: toAPIAmount(initialBalance || '0'),
				credit_limit: type === 'credit_card' ? toAPIAmount(creditLimit) : undefined,
				payment_account_id:
					type === 'credit_card' && paymentAccountId ? paymentAccountId : undefined
			});
			await goto(resolve(`/accounts/${acc.id}`));
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
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
			<div class="flex flex-wrap gap-2">
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
				<button
					type="button"
					class={type === 'credit_card' ? 'tab tab-active' : 'tab'}
					onclick={() => (type = 'credit_card')}
				>
					{$_('accounts.type.credit_card')}
				</button>
			</div>
		</div>

		{#if needsBank}
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

		{#if type === 'credit_card'}
			<div>
				<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="credit-limit">
					{$_('accounts.field.creditLimit')}
				</label>
				<MoneyInput id="credit-limit" bind:value={creditLimit} required />
			</div>
			<Select
				label={$_('accounts.field.paymentAccount')}
				bind:value={paymentAccountId}
				options={[
					{ value: '', label: $_('accounts.creditCard.paymentAccountDefault') },
					...paymentOptions
				]}
				usePortal
			/>
		{/if}

		<div>
			<label class="mb-1 block text-sm" style:color="var(--text-muted)" for="balance">
				{$_('accounts.field.balance')}
			</label>
			<MoneyInput id="balance" bind:value={initialBalance} />
			{#if type === 'credit_card'}
				<button type="button" class="btn-ghost mt-1 text-sm" onclick={applyLimitToBalance}>
					{$_('accounts.creditCard.limitButton')}
				</button>
			{/if}
		</div>

		<div class="flex gap-2 pt-2">
			<button
				type="submit"
				class="btn-primary"
				disabled={loading ||
					(needsBank && !bankId) ||
					(type === 'credit_card' && !creditLimit.trim())}
			>
				{loading ? $_('common.loading') : $_('common.create')}
			</button>
			<a href={resolve('/accounts')} class="btn-ghost">{$_('common.cancel')}</a>
		</div>
	</form>
</div>
