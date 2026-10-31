<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		listAccounts,
		listBanks,
		listCurrencies,
		setPrimaryAccount,
		unarchiveAccount,
		updateAccount,
		type Account,
		type Bank,
		type Currency
	} from '$lib/api/client';
	import {
		promptArchiveAccount,
		executeArchiveAccount,
		promptDeleteAccount,
		executeDeleteAccount
	} from '$lib/accounts/account-inactive-prompt';
	import CreditCardChangeLimitForm from '$lib/components/CreditCardChangeLimitForm.svelte';
	import CreditCardFeeForm from '$lib/components/CreditCardFeeForm.svelte';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import { isAutoTopupEligible } from '$lib/accounts/auto-topup';
	import AccountAutoTopupDialog from '$lib/components/AccountAutoTopupDialog.svelte';
	import { featureFlags, isFeatureEnabled } from '$lib/features';
	import {
		groupAccountsForDisplay,
		groupAccountsByOwnership,
		accountGroupLabelKey,
		accountOwnershipLabelKey
	} from '$lib/accounts/group-by-type';
	import FamilyBadge from '$lib/components/FamilyBadge.svelte';
	import { householdMe } from '$lib/household/membership';
	import {
		canShareAccount,
		canUnshareAccount,
		promptShareAccount,
		promptUnshareAccount
	} from '$lib/household/share-account';
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
	import { currencySelectOptions } from '$lib/currency';
	import { currencySymbol } from '$lib/finance';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import { refCacheReady, refCacheReadyAny, refCacheUpdate } from '$lib/ref-cache';
	import { refCacheUpdateMatches } from '$lib/ref-cache-watch';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { assignIfChanged } from '$lib/state-utils';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import PullToRefresh from '$lib/components/PullToRefresh.svelte';

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
	let editPaymentAccountId = $state('');
	let editInitialBalance = $state('');
	let editCurrency = $state('');
	let currencyLocked = $state(false);
	let currencies = $state<Currency[]>([]);
	let payOpen = $state(false);
	let feeOpen = $state(false);
	let changeLimitOpen = $state(false);
	let autoTopupOpen = $state(false);
	let actionCard = $state<Account | null>(null);
	let savingEditId = $state('');

	const bankOptions = $derived(banks.map((bank) => ({ value: bank.id, label: bank.name })));
	const accountGroups = $derived(groupAccountsForDisplay(accounts));
	const accountSections = $derived(groupAccountsByOwnership(accountGroups));
	const showOwnershipHeadings = $derived(
		accountGroups.some((group) => group.ownership === 'household')
	);

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
		if (refCacheUpdateMatches(update, [listPath, accountsPath('active'), '/api/v1/banks'])) {
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
			const [accountList, bankList, activeList, currencyList] = await Promise.all([
				listAccounts(filter),
				listBanks(),
				listAccounts('active'),
				listCurrencies()
			]);
			accounts = opts.silent ? assignIfChanged(accounts, accountList) : accountList;
			banks = opts.silent ? assignIfChanged(banks, bankList) : bankList;
			activeAccounts = opts.silent ? assignIfChanged(activeAccounts, activeList) : activeList;
			currencies = opts.silent ? assignIfChanged(currencies, currencyList) : currencyList;
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

	const currencyOptions = $derived(currencySelectOptions(currencies));

	function startEdit(acc: Account) {
		editingId = acc.id;
		editName = acc.name;
		editBankId = acc.bank_id ?? '';
		editPaymentAccountId = acc.payment_account_id ?? '';
		editInitialBalance = formatAccountInitialBalanceForEdit(acc.initial_balance);
		editCurrency = acc.currency ?? $user?.currency ?? '';
		currencyLocked = false;
	}

	function cancelEdit() {
		editingId = null;
	}

	function applyLimitToBalance(acc: Account) {
		if (acc.credit_limit_display) editInitialBalance = acc.credit_limit_display;
	}

	async function saveEdit(e: Event, acc: Account) {
		e.preventDefault();
		if (savingEditId) return;
		savingEditId = acc.id;
		try {
			const updated = await updateAccount(acc.id, {
				name: editName,
				currency: editCurrency || undefined,
				bank_id: acc.type === 'bank' || acc.type === 'credit_card' ? editBankId : undefined,
				initial_balance: toAPIAmount(editInitialBalance),
				payment_account_id: acc.type === 'credit_card' ? editPaymentAccountId || null : undefined
			});
			accounts = accounts.map((item) => (item.id === updated.id ? updated : item));
			editingId = null;
			toast($_('common.saved'));
		} catch (err) {
			if (err instanceof ApiError && err.code === 'ERR_ACCOUNT_CURRENCY_LOCKED') {
				currencyLocked = true;
			}
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

	async function onShare(acc: Account) {
		const result = await promptShareAccount(acc, accounts);
		if (result) await load();
	}

	async function onUnshare(acc: Account) {
		const result = await promptUnshareAccount(acc);
		if (result) await load();
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
				},
				{
					icon: 'bank',
					label: $_('accounts.creditCard.changeLimit'),
					disabled: busy || editingId !== null,
					onclick: () => {
						actionCard = acc;
						changeLimitOpen = true;
					}
				}
			);
		}
		if (filter === 'active' && canShareAccount(acc, $householdMe)) {
			actions.push({
				icon: 'share',
				label: $_('accounts.share'),
				disabled: busy || editingId !== null,
				onclick: () => void onShare(acc)
			});
		}
		if (filter === 'active' && canUnshareAccount(acc, $householdMe)) {
			actions.push({
				icon: 'transfer',
				label: $_('accounts.unshare'),
				disabled: busy || editingId !== null,
				onclick: () => void onUnshare(acc)
			});
		}
		actions.push({
			icon: 'edit',
			label: $_('accounts.action.edit'),
			disabled: busy || (editingId !== null && editingId !== acc.id),
			onclick: () => startEdit(acc)
		});
		if (
			filter === 'active' &&
			isFeatureEnabled('balance_maintenance', $featureFlags) &&
			isAutoTopupEligible(acc)
		) {
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

<PullToRefresh onRefresh={() => load({ silent: true })}>
	<div class="space-y-6">
		<BackLink
			items={[
				{ href: '/', label: $_('nav.home') },
				{ href: '/accounts', label: $_('accounts.title') }
			]}
		/>

		<SectionHeader title={$_('accounts.title')}>
			{#snippet actions()}
				<!-- eslint-disable svelte/no-navigation-without-resolve -- query after resolved path -->
				<a
					href={`${resolve('/accounts/new')}?from=${encodeURIComponent('/accounts')}`}
					class="btn-primary"
				>
					{$_('accounts.new')}
				</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
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
					{#each accountSections as section (section.ownership)}
						<section class="space-y-6">
							{#if showOwnershipHeadings}
								<h2 class="text-lg font-semibold">
									{$_(accountOwnershipLabelKey(section.ownership, $householdMe.household?.kind))}
								</h2>
							{/if}
							{#each section.groups as group (`${group.kind}`)}
								<div>
									<h3 class="mb-3 px-1 text-base font-semibold">
										{$_(accountGroupLabelKey(group.kind))}
										<span class="font-normal tabular-nums" style:color="var(--text-muted)">
											· {group.accounts.length}
										</span>
									</h3>
									<div class="grid gap-4 sm:grid-cols-2">
										{#each group.accounts as acc (acc.id)}
											{@render accountCard(acc)}
										{/each}
									</div>
								</div>
							{/each}
						</section>
					{/each}
				</div>
			{/if}
		</PageLoadGate>
	</div>
</PullToRefresh>

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
							<!-- svelte-ignore a11y_autofocus -->
							<input
								id="edit-name-{acc.id}"
								class="input w-full"
								bind:value={editName}
								required
								maxlength="64"
								autofocus
							/>
						</div>
						<Select
							id="account-currency"
							label={$_('accounts.field.currency')}
							hint={currencyLocked ? $_('accounts.currency.locked') : $_('accounts.currency.hint')}
							bind:value={editCurrency}
							options={currencyOptions}
							disabled={currencyLocked}
							usePortal
						/>
						{#if acc.type === 'bank' || acc.type === 'credit_card'}
							<Select
								label={$_('accounts.field.bank')}
								bind:value={editBankId}
								options={bankOptions}
								usePortal
							/>
						{/if}
						{#if acc.type === 'credit_card'}
							{#if acc.credit_limit_display}
								<div>
									<p class="mb-1 text-sm" style:color="var(--text-muted)">
										{$_('accounts.field.creditLimit')}
									</p>
									<p class="tabular-nums">
										<MoneyDisplay
											value={acc.credit_limit_display}
											currency={acc.currency}
											class=""
										/>
									</p>
								</div>
							{/if}
							<Select
								label={$_('accounts.field.paymentAccount')}
								bind:value={editPaymentAccountId}
								options={[
									{ value: '', label: $_('accounts.creditCard.paymentAccountDefault') },
									...accountSelectOptions(
										activeAccounts.filter(
											(a) =>
												a.type !== 'credit_card' &&
												a.id !== acc.id &&
												a.currency === (editCurrency || acc.currency)
										)
									)
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
								<button
									type="button"
									class="btn-ghost mt-1 text-sm"
									onclick={() => applyLimitToBalance(acc)}
								>
									{$_('accounts.creditCard.limitButton')}
								</button>
							{/if}
						</div>
						<div class="flex flex-wrap gap-2">
							<button
								type="submit"
								class="btn-primary"
								disabled={savingEditId === acc.id ||
									((acc.type === 'bank' || acc.type === 'credit_card') && !editBankId)}
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
						<div class="flex items-center gap-1.5">
							<p class="truncate font-medium">{acc.name}</p>
							{#if acc.scope === 'household'}
								<FamilyBadge compact />
							{/if}
							{#if acc.currency}
								<span class="shrink-0 text-sm font-normal" style:color="var(--text-muted)">
									{currencySymbol(acc.currency)}
								</span>
							{/if}
							{#if acc.is_primary}
								<span
									class="badge badge-success shrink-0"
									title={$_('accounts.primary.badge')}
									aria-label={$_('accounts.primary.badge')}>{$_('accounts.primary.short')}</span
								>
							{/if}
						</div>
						<p class="mt-1 text-xl font-semibold tabular-nums">
							<MoneyDisplay value={acc.balance_display} currency={acc.currency} class="" />
						</p>
						{#if acc.credit_limit_display}
							<p class="mt-0.5 text-sm tabular-nums" style:color="var(--text-muted)">
								{$_('accounts.field.creditLimit')}:
								<MoneyDisplay value={acc.credit_limit_display} currency={acc.currency} class="" />
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
	<CreditCardChangeLimitForm
		bind:open={changeLimitOpen}
		account={actionCard}
		onclose={() => {
			changeLimitOpen = false;
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
