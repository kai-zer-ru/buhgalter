<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		archiveAccount,
		deleteAccount,
		listAccounts,
		listBanks,
		setPrimaryAccount,
		unarchiveAccount,
		updateAccount,
		type Account,
		type Bank
	} from '$lib/api/client';
	import CreditCardFeeForm from '$lib/components/CreditCardFeeForm.svelte';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import { isCreditCard } from '$lib/credit-card';
	import BackLink from '$lib/components/BackLink.svelte';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import Select from '$lib/components/Select.svelte';
	import { confirm } from '$lib/confirm';
	import { formatBalance } from '$lib/finance';
	import { formatMoneyForInput, toAPIAmount } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let accounts = $state<Account[]>([]);
	let activeAccounts = $state<Account[]>([]);
	let banks = $state<Bank[]>([]);
	let filter = $state<'active' | 'archived'>('active');
	let loading = $state(true);
	let filterLoading = $state(false);
	let ready = $state(false);
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
	let actionCard = $state<Account | null>(null);
	let savingEditId = $state('');

	const bankOptions = $derived(banks.map((bank) => ({ value: bank.id, label: bank.name })));

	onMount(() => {
		void load();
	});

	async function load(opts: { filterChange?: boolean } = {}) {
		if (opts.filterChange) {
			filterLoading = true;
		} else {
			loading = true;
		}
		try {
			const [accountList, bankList, activeList] = await Promise.all([
				listAccounts(filter),
				listBanks(),
				listAccounts('active')
			]);
			accounts = accountList;
			banks = bankList;
			activeAccounts = activeList;
			ready = true;
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
			filterLoading = false;
		}
	}

	function setFilter(next: 'active' | 'archived') {
		if (next === filter) return;
		filter = next;
		void load({ filterChange: true });
	}

	const paymentOptions = $derived(
		activeAccounts
			.filter((a) => a.type !== 'credit_card')
			.map((a) => ({ value: a.id, label: a.name }))
	);

	function startEdit(acc: Account) {
		editingId = acc.id;
		editName = acc.name;
		editBankId = acc.bank_id ?? '';
		editCreditLimit = acc.credit_limit_display ?? '';
		editPaymentAccountId = acc.payment_account_id ?? '';
		editInitialBalance = formatMoneyForInput(acc.balance_display);
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
		} catch (err) {
			toast.fromError(err);
		} finally {
			primarySavingId = '';
		}
	}

	async function archive(id: string) {
		if (!id || actionSavingId) return;
		actionSavingId = id;
		try {
			await archiveAccount(id);
			accounts = accounts.filter((acc) => acc.id !== id);
			activeAccounts = activeAccounts.filter((acc) => acc.id !== id);
			if (editingId === id) editingId = null;
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

	async function remove(id: string) {
		if (!id || actionSavingId) return;
		const ok = await confirm({
			message: $_('accounts.confirm.delete'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		actionSavingId = id;
		try {
			await deleteAccount(id);
			accounts = accounts.filter((acc) => acc.id !== id);
			if (editingId === id) editingId = null;
			toast($_('common.deleted'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			actionSavingId = '';
		}
	}

	function accountActions(acc: Account): RowAction[] {
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
		if (filter === 'active' && !acc.is_primary) {
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
		} else {
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
			onclick: () => void remove(acc.id)
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
		<a href={resolve('/accounts/new')} class="btn-primary">
			{$_('accounts.new')}
		</a>
	</div>

	<PageTabs
		active={filter}
		tabs={[
			{ id: 'active', label: $_('accounts.filter.active') },
			{ id: 'archived', label: $_('accounts.filter.archived') }
		]}
		onchange={(next) => setFilter(next as 'active' | 'archived')}
	/>

	{#if loading}
		<p style:color="var(--text-muted)">{$_('common.loading')}</p>
	{:else if !ready}
		<!-- load failed; toast shown -->
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
											<button
												type="button"
												class="btn-ghost mt-1 text-sm"
												onclick={applyLimitToBalance}
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
										{formatBalance(acc.balance_display, $user?.currency ?? 'RUB')}
									</p>
									{#if acc.credit_limit_display}
										<p class="mt-0.5 text-sm tabular-nums" style:color="var(--text-muted)">
											{$_('accounts.field.creditLimit')}:
											{formatBalance(acc.credit_limit_display, $user?.currency ?? 'RUB')}
										</p>
									{/if}
								</div>
							</a>
							<RowActionsMenu actions={accountActions(acc)} />
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

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
{/if}
