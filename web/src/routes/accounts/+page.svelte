<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		listAccounts,
		listBanks,
		setPrimaryAccount,
		unarchiveAccount,
		updateAccount,
		type Account,
		type Bank
	} from '$lib/api/client';
	import {
		promptArchiveAccount,
		executeArchiveAccount,
		promptDeleteAccount,
		executeDeleteAccount
	} from '$lib/accounts/account-inactive-prompt';
	import CreditCardFeeForm from '$lib/components/CreditCardFeeForm.svelte';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import { isAutoTopupEligible } from '$lib/accounts/auto-topup';
	import AccountAutoTopupDialog from '$lib/components/AccountAutoTopupDialog.svelte';
	import {
		groupAccountsByType,
		accountGroupKind,
		accountGroupLabelKey
	} from '$lib/accounts/group-by-type';
	import { isCreditCard } from '$lib/credit-card';
	import BackLink from '$lib/components/BackLink.svelte';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import Select from '$lib/components/Select.svelte';
	import { accountSelectOptions } from '$lib/select-options';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { canSetAsPrimary, formatAccountInitialBalanceForEdit } from '$lib/accounts';
	import { toAPIAmount } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import { refCacheReady, refCacheReadyAny, refCacheUpdate } from '$lib/ref-cache';
	import { refCachePathMatches } from '$lib/ref-cache-watch';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { assignIfChanged } from '$lib/state-utils';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';

	function accountsPath(status: 'active' | 'archived' | 'deleted' | '' = '') {
		return status ? `/api/v1/accounts?status=${status}` : '/api/v1/accounts';
	}

	let accounts = $state<Account[]>([]);
	let activeAccounts = $state<Account[]>([]);
	let banks = $state<Bank[]>([]);
	let filter = $state<'active' | 'archived' | 'deleted'>('active');
	let loading = $state(
		!refCacheReady(accountsPath('active')) &&
			!refCacheReady(accountsPath()) &&
			!refCacheReady('/api/v1/banks')
	);
	let filterLoading = $state(false);
	let ready = $state(false);
	let loadError = $state<string | null>(null);
	let primarySavingId = $state('');
	let actionSavingId = $state('');
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editBankId = $state('');
	let editCreditLimit = $state('');
	let editPaymentAccountId = $state('');
	let editInitialBalance = $state('');
	let payOpen = $state(false);
	let feeOpen = $state(false);
	let autoTopupOpen = $state(false);
	let actionCard = $state<Account | null>(null);
	let savingEditId = $state('');

	const bankOptions = $derived(banks.map((bank) => ({ value: bank.id, label: bank.name })));
	const accountGroups = $derived(groupAccountsByType(accounts));

	onMount(() => {
		const status = $page.url.searchParams.get('status');
		if (status === 'archived' || status === 'deleted') {
			filter = status;
		}
		void load();
	});

	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || !ready) return;
		const listPath = accountsPath(filter);
		if (refCachePathMatches(update.path, [listPath, accountsPath('active'), '/api/v1/banks'])) {
			void load({ silent: true });
		}
	});

	async function load(opts: { filterChange?: boolean; silent?: boolean } = {}) {
		const listPath = accountsPath(filter);
		if (opts.filterChange) {
			if (!opts.silent && !refCacheReady(listPath)) filterLoading = true;
		} else if (
			!opts.silent &&
			!refCacheReadyAny([listPath, accountsPath('active'), '/api/v1/banks'])
		) {
			loading = true;
		}
		try {
			const [accountList, bankList, activeList] = await Promise.all([
				listAccounts(filter),
				listBanks(),
				listAccounts('active')
			]);
			accounts = opts.silent ? assignIfChanged(accounts, accountList) : accountList;
			banks = opts.silent ? assignIfChanged(banks, bankList) : bankList;
			activeAccounts = opts.silent ? assignIfChanged(activeAccounts, activeList) : activeList;
			ready = true;
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { silent: opts.silent, hasData: ready });
			if (msg) loadError = msg;
		} finally {
			loading = false;
			filterLoading = false;
		}
	}

	function setFilter(next: 'active' | 'archived' | 'deleted') {
		if (next === filter) return;
		filter = next;
		void load({ filterChange: true });
	}

	const paymentOptions = $derived(
		accountSelectOptions(activeAccounts.filter((a) => a.type !== 'credit_card'))
	);

	function startEdit(acc: Account) {
		editingId = acc.id;
		editName = acc.name;
		editBankId = acc.bank_id ?? '';
		editCreditLimit = acc.credit_limit_display ?? '';
		editPaymentAccountId = acc.payment_account_id ?? '';
		editInitialBalance = formatAccountInitialBalanceForEdit(acc.initial_balance);
	}

	function cancelEdit() {
		editingId = null;
	}

	function applyLimitToBalance() {
		if (editCreditLimit.trim()) editInitialBalance = editCreditLimit;
	}

	async function saveEdit(e: Event, acc: Account) {
		e.preventDefault();
		if (savingEditId) return;
		savingEditId = acc.id;
		try {
			const updated = await updateAccount(acc.id, {
				name: editName,
				bank_id: acc.type === 'bank' || acc.type === 'credit_card' ? editBankId : undefined,
				initial_balance: toAPIAmount(editInitialBalance),
				credit_limit: acc.type === 'credit_card' ? toAPIAmount(editCreditLimit) : undefined,
				payment_account_id: acc.type === 'credit_card' ? editPaymentAccountId || null : undefined
			});
			accounts = accounts.map((item) => (item.id === updated.id ? updated : item));
			editingId = null;
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			savingEditId = '';
		}
	}

	async function makePrimary(id: string) {
		if (!id || primarySavingId) return;
		primarySavingId = id;
		try {
			const updated = await setPrimaryAccount(id);
			accounts = accounts.map((acc) => ({
				...acc,
				is_primary: acc.id === updated.id
			}));
			void load({ silent: true });
		} catch (err) {
			toast.fromError(err);
		} finally {
			primarySavingId = '';
		}
	}

	async function archive(id: string) {
		if (!id || actionSavingId) return;
		const acc = accounts.find((a) => a.id === id);
		if (!acc) return;
		const confirmed = await promptArchiveAccount({ acc, activeAccounts });
		if (!confirmed.ok) return;
		actionSavingId = id;
		try {
			await executeArchiveAccount(acc, confirmed.transferToAccountId);
			if (editingId === id) editingId = null;
			await load({ filterChange: true });
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			actionSavingId = '';
		}
	}

	async function unarchive(id: string) {
		if (!id || actionSavingId) return;
		actionSavingId = id;
		try {
			const restored = await unarchiveAccount(id);
			accounts = accounts.filter((acc) => acc.id !== id);
			activeAccounts = [...activeAccounts, restored];
			if (editingId === id) editingId = null;
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			actionSavingId = '';
		}
	}

	async function remove(acc: Account) {
		if (!acc.id || actionSavingId) return;
		const confirmed = await promptDeleteAccount({ acc, activeAccounts });
		if (!confirmed.ok) return;
		actionSavingId = acc.id;
		try {
			await executeDeleteAccount(acc, confirmed.transferToAccountId);
			filter = 'deleted';
			await load({ filterChange: true });
			if (editingId === acc.id) editingId = null;
			toast($_('common.deleted'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			actionSavingId = '';
		}
	}

	function accountActions(acc: Account): RowAction[] {
		if (filter === 'deleted') return [];
		const busy = actionSavingId === acc.id || primarySavingId === acc.id || savingEditId === acc.id;
		const actions: RowAction[] = [];
		if (filter === 'active' && isCreditCard(acc)) {
			actions.push(
				{
					icon: 'transfer',
					label: $_('accounts.creditCard.pay'),
					disabled: busy || editingId !== null,
					onclick: () => {
						actionCard = acc;
						payOpen = true;
					}
				},
				{
					icon: 'expense',
					label: $_('accounts.creditCard.chargeFee'),
					disabled: busy || editingId !== null,
					onclick: () => {
						actionCard = acc;
						feeOpen = true;
					}
				}
			);
		}
		actions.push({
			icon: 'edit',
			label: $_('accounts.action.edit'),
			disabled: busy || (editingId !== null && editingId !== acc.id),
			onclick: () => startEdit(acc)
		});
		if (filter === 'active' && isAutoTopupEligible(acc)) {
			actions.push({
				icon: 'transfer',
				label: $_('accounts.action.autoTopup'),
				disabled: busy || editingId !== null,
				onclick: () => {
					actionCard = acc;
					autoTopupOpen = true;
				}
			});
		}
		if (filter === 'active' && canSetAsPrimary(acc)) {
			actions.push({
				icon: 'save',
				label: $_('accounts.primary.set'),
				disabled: busy || editingId !== null,
				onclick: () => void makePrimary(acc.id)
			});
		}
		if (filter === 'active') {
			actions.push({
				icon: 'archive',
				label: $_('accounts.action.archive'),
				disabled: busy || editingId !== null,
				onclick: () => void archive(acc.id)
			});
		} else if (filter === 'archived') {
			actions.push({
				icon: 'archive',
				label: $_('accounts.action.unarchive'),
				disabled: busy || editingId !== null,
				onclick: () => void unarchive(acc.id)
			});
		}
		actions.push({
			icon: 'delete',
			label: $_('common.delete'),
			variant: 'danger',
			disabled: busy || editingId !== null,
			onclick: () => void remove(acc)
		});
		return actions;
	}
</script>

<svelte:head>
	<title>{$_('accounts.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/accounts', label: $_('accounts.title') }
		]}
	/>

	<div class="flex flex-wrap items-center justify-between gap-3">
		<h1 class="text-2xl font-semibold">{$_('accounts.title')}</h1>
		<!-- eslint-disable svelte/no-navigation-without-resolve -- query after resolved path -->
		<a
			href={`${resolve('/accounts/new')}?from=${encodeURIComponent('/accounts')}`}
			class="btn-primary"
		>
			{$_('accounts.new')}
		</a>
		<!-- eslint-enable svelte/no-navigation-without-resolve -->
	</div>

	<PageTabs
		active={filter}
		tabs={[
			{ id: 'active', label: $_('accounts.filter.active') },
			{ id: 'archived', label: $_('accounts.filter.archived') },
			{ id: 'deleted', label: $_('accounts.filter.deleted') }
		]}
		onchange={(next) => setFilter(next as 'active' | 'archived' | 'deleted')}
	/>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
		{#if accounts.length === 0 && filterLoading}
			<EmptyStateCard message={$_('common.loading')} ariaBusy />
		{:else if accounts.length === 0}
			<EmptyStateCard message={$_('accounts.empty')} />
		{:else}
			<div class="relative space-y-6" class:opacity-60={filterLoading}>
				{#if filterLoading}
					<p
						class="pointer-events-none absolute inset-x-0 top-0 py-2 text-center text-sm"
						style:color="var(--text-muted)"
					>
						{$_('common.loading')}
					</p>
				{/if}
				{#each accountGroups as group (accountGroupKind(group))}
					{@const kind = accountGroupKind(group)}
					<section>
						<h2 class="mb-3 text-lg font-medium">
							{$_(accountGroupLabelKey(kind))}
							<span class="font-normal tabular-nums" style:color="var(--text-muted)">
								({group.length})
							</span>
						</h2>
						<div class="grid gap-4 sm:grid-cols-2">
							{#each group as acc (acc.id)}
								{@render accountCard(acc)}
							{/each}
						</div>
					</section>
				{/each}
			</div>
		{/if}
	</PageLoadGate>
</div>

{#snippet accountCard(acc: Account)}
	<div class="card">
		{#if editingId === acc.id}
			<form class="space-y-3" onsubmit={(e) => saveEdit(e, acc)}>
				<div class="flex items-start gap-3">
					<AccountIcon type={acc.type} bankIcon={acc.bank_icon} size={48} />
					<div class="min-w-0 flex-1 space-y-3">
						<div>
							<label
								class="mb-1 block text-sm"
								style:color="var(--text-muted)"
								for="edit-name-{acc.id}"
							>
								{$_('accounts.field.name')}
							</label>
							<input
								id="edit-name-{acc.id}"
								class="input w-full"
								bind:value={editName}
								required
								maxlength="64"
								autofocus
							/>
						</div>
						{#if acc.type === 'bank' || acc.type === 'credit_card'}
							<Select
								label={$_('accounts.field.bank')}
								bind:value={editBankId}
								options={bankOptions}
								usePortal
							/>
						{/if}
						{#if acc.type === 'credit_card'}
							<div>
								<label
									class="mb-1 block text-sm"
									style:color="var(--text-muted)"
									for="edit-limit-{acc.id}"
								>
									{$_('accounts.field.creditLimit')}
								</label>
								<MoneyInput id="edit-limit-{acc.id}" bind:value={editCreditLimit} />
							</div>
							<Select
								label={$_('accounts.field.paymentAccount')}
								bind:value={editPaymentAccountId}
								options={[
									{ value: '', label: $_('accounts.creditCard.paymentAccountDefault') },
									...paymentOptions.filter((o) => o.value !== acc.id)
								]}
								usePortal
							/>
						{/if}
						<div>
							<label
								class="mb-1 block text-sm"
								style:color="var(--text-muted)"
								for="edit-balance-{acc.id}"
							>
								{$_('accounts.field.balance')}
							</label>
							<MoneyInput id="edit-balance-{acc.id}" bind:value={editInitialBalance} />
							{#if acc.type === 'credit_card'}
								<button type="button" class="btn-ghost mt-1 text-sm" onclick={applyLimitToBalance}>
									{$_('accounts.creditCard.limitButton')}
								</button>
							{/if}
						</div>
						<div class="flex flex-wrap gap-2">
							<button
								type="submit"
								class="btn-primary"
								disabled={savingEditId === acc.id ||
									((acc.type === 'bank' || acc.type === 'credit_card') && !editBankId) ||
									(acc.type === 'credit_card' && !editCreditLimit.trim())}
							>
								{savingEditId === acc.id ? $_('common.loading') : $_('common.save')}
							</button>
							<button type="button" class="btn-ghost" onclick={cancelEdit}>
								{$_('common.cancel')}
							</button>
						</div>
					</div>
				</div>
			</form>
		{:else}
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
							<MoneyDisplay
								value={acc.balance_display}
								currency={$user?.currency ?? 'RUB'}
								class=""
							/>
						</p>
						{#if acc.credit_limit_display}
							<p class="mt-0.5 text-sm tabular-nums" style:color="var(--text-muted)">
								{$_('accounts.field.creditLimit')}:
								<MoneyDisplay
									value={acc.credit_limit_display}
									currency={$user?.currency ?? 'RUB'}
									class=""
								/>
							</p>
						{/if}
					</div>
				</a>
				{#if filter !== 'deleted'}
					<RowActionsMenu actions={accountActions(acc)} />
				{/if}
			</div>
		{/if}
	</div>
{/snippet}

{#if actionCard}
	<TransferForm
		bind:open={payOpen}
		accountId={actionCard.id}
		creditCardPay={actionCard}
		onclose={() => {
			payOpen = false;
			actionCard = null;
		}}
		onsaved={load}
	/>
	<CreditCardFeeForm
		bind:open={feeOpen}
		account={actionCard}
		onclose={() => {
			feeOpen = false;
			actionCard = null;
		}}
		onsaved={load}
	/>
	<AccountAutoTopupDialog
		bind:open={autoTopupOpen}
		account={actionCard}
		onclose={() => {
			autoTopupOpen = false;
			actionCard = null;
		}}
		onsaved={load}
	/>
{/if}
