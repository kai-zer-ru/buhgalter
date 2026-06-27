<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		archiveAccount,
		deleteAccount,
		listAccounts,
		listBanks,
		setPrimaryAccount,
		updateAccount,
		type Account,
		type Bank
	} from '$lib/api/client';
	import AccountIcon from '$lib/components/AccountIcon.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import Select from '$lib/components/Select.svelte';
	import { confirm } from '$lib/confirm';
	import { formatBalance } from '$lib/finance';
	import { formatMoneyDisplay, toAPIAmount } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let accounts = $state<Account[]>([]);
	let banks = $state<Bank[]>([]);
	let loading = $state(true);
	let error = $state('');
	let primarySavingId = $state('');
	let actionSavingId = $state('');
	let editingId = $state<string | null>(null);
	let editName = $state('');
	let editBankId = $state('');
	let editInitialBalance = $state('');
	let savingEditId = $state('');

	const bankOptions = $derived(banks.map((bank) => ({ value: bank.id, label: bank.name })));

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			const [accountList, bankList] = await Promise.all([listAccounts(), listBanks()]);
			accounts = accountList;
			banks = bankList;
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}

	function startEdit(acc: Account) {
		editingId = acc.id;
		editName = acc.name;
		editBankId = acc.bank_id ?? '';
		editInitialBalance = formatMoneyDisplay(acc.balance_display);
	}

	function cancelEdit() {
		editingId = null;
	}

	async function saveEdit(e: Event, acc: Account) {
		e.preventDefault();
		if (savingEditId) return;
		savingEditId = acc.id;
		error = '';
		try {
			const updated = await updateAccount(acc.id, {
				name: editName,
				bank_id: acc.type === 'bank' ? editBankId : undefined,
				initial_balance: toAPIAmount(editInitialBalance)
			});
			accounts = accounts.map((item) => (item.id === updated.id ? updated : item));
			editingId = null;
			toast($_('common.saved'));
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			savingEditId = '';
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

	async function archive(id: string) {
		if (!id || actionSavingId) return;
		actionSavingId = id;
		error = '';
		try {
			await archiveAccount(id);
			accounts = accounts.filter((acc) => acc.id !== id);
			if (editingId === id) editingId = null;
			toast($_('common.saved'));
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
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
		error = '';
		try {
			await deleteAccount(id);
			accounts = accounts.filter((acc) => acc.id !== id);
			if (editingId === id) editingId = null;
			toast($_('common.deleted'));
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			actionSavingId = '';
		}
	}

	function accountActions(acc: Account): RowAction[] {
		const busy = actionSavingId === acc.id || primarySavingId === acc.id || savingEditId === acc.id;
		const actions: RowAction[] = [
			{
				icon: 'edit',
				label: $_('accounts.action.edit'),
				disabled: busy || (editingId !== null && editingId !== acc.id),
				onclick: () => startEdit(acc)
			}
		];
		if (!acc.is_primary) {
			actions.push({
				icon: 'save',
				label: $_('accounts.primary.set'),
				disabled: busy || editingId !== null,
				onclick: () => void makePrimary(acc.id)
			});
		}
		actions.push(
			{
				icon: 'archive',
				label: $_('accounts.action.archive'),
				disabled: busy || editingId !== null,
				onclick: () => void archive(acc.id)
			},
			{
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				disabled: busy || editingId !== null,
				onclick: () => void remove(acc.id)
			}
		);
		return actions;
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
									{#if acc.type === 'bank'}
										<Select
											label={$_('accounts.field.bank')}
											bind:value={editBankId}
											options={bankOptions}
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
									</div>
									<div class="flex flex-wrap gap-2">
										<button
											type="submit"
											class="btn-primary"
											disabled={savingEditId === acc.id || (acc.type === 'bank' && !editBankId)}
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
