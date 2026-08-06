<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { listAccounts, listBanks, type Account, type Bank } from '$lib/api/client';
	import Select from '$lib/components/Select.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { accountSelectOptions } from '$lib/select-options';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import {
		KNOWN_BANK_APPS,
		getCurrentInterceptSettings,
		getNotificationListenerState,
		isNotificationAccessEnabled,
		openBackgroundRestrictionsSettings,
		openNotificationAccessSettings,
		packageForBankId,
		reconnectNotificationListener,
		saveInterceptSettings,
		syncInterceptNativeFromSettings,
		type CardBinding,
		type InterceptSettings,
		normalizeLast4
	} from '$lib/android/notification-intercept';
	import { isNativeApp } from '$lib/platform/native';
	import { sortAccountsForSelect } from '$lib/accounts';

	let settings = $state<InterceptSettings>(getCurrentInterceptSettings());
	let accounts = $state<Account[]>([]);
	let banks = $state<Bank[]>([]);
	let accessEnabled = $state(false);
	let listenerConnected = $state(false);
	let reconnecting = $state(false);
	let loading = $state(true);
	let newCardBankId = $state('tinkoff');
	let newCardLast4 = $state('');
	let newCardAccountId = $state('');

	/** Accounts that can receive bank push (not cash). */
	const bindableAccounts = $derived(
		sortAccountsForSelect(accounts.filter((a) => a.type === 'bank' || a.type === 'credit_card'))
	);

	const bankOptions = $derived([
		{ value: '', label: $_('bankNotifications.bank.none') },
		...KNOWN_BANK_APPS.map((b) => ({
			value: b.bankId,
			label: banks.find((x) => x.id === b.bankId)?.name ?? $_(b.labelKey)
		}))
	]);

	const bankName = (bankId: string) =>
		banks.find((b) => b.id === bankId)?.name ?? $_(`bankNotifications.bank.${bankId}`);

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		settings = getCurrentInterceptSettings();
		try {
			accounts = await listAccounts('active');
			banks = await listBanks();
		} catch {
			accounts = [];
			banks = [];
		}
		accessEnabled = await isNotificationAccessEnabled();
		const listener = await getNotificationListenerState();
		listenerConnected = Boolean(listener?.listenerConnected);
		if (!newCardAccountId && bindableAccounts.length) {
			newCardAccountId = bindableAccounts[0].id;
			const bound = bankIdForAccount(bindableAccounts[0].id);
			if (bound) newCardBankId = bound;
			else if (bindableAccounts[0].bank_id) newCardBankId = bindableAccounts[0].bank_id;
		}
		loading = false;
	}

	async function reconnectListener() {
		reconnecting = true;
		try {
			const r = await reconnectNotificationListener();
			accessEnabled = r.notificationAccess;
			listenerConnected = r.listenerConnected;
			if (r.listenerConnected) {
				toast($_('bankNotifications.listener.reconnected'));
			} else {
				toast.error($_('bankNotifications.listener.reconnectFailed'));
			}
		} finally {
			reconnecting = false;
		}
	}

	function persist(next: InterceptSettings) {
		const uid = $user?.id;
		if (!uid) return;
		saveInterceptSettings(uid, next);
		settings = next;
		void syncInterceptNativeFromSettings(uid);
	}

	async function toggleEnabled() {
		if (!isNativeApp()) {
			toast($_('bankNotifications.nativeOnly'));
			return;
		}
		const nextEnabled = !settings.enabled;
		if (nextEnabled) {
			accessEnabled = await isNotificationAccessEnabled();
			if (!accessEnabled) {
				toast($_('bankNotifications.needAccess'));
				persist({ ...settings, enabled: nextEnabled });
				await openAccess();
				return;
			}
		}
		persist({ ...settings, enabled: nextEnabled });
		if (nextEnabled) {
			const r = await reconnectNotificationListener();
			listenerConnected = r.listenerConnected;
			accessEnabled = r.notificationAccess;
		}
		toast($_('common.saved'));
	}

	async function openAccess() {
		await openNotificationAccessSettings();
		setTimeout(() => {
			void (async () => {
				accessEnabled = await isNotificationAccessEnabled();
				const listener = await getNotificationListenerState();
				listenerConnected = Boolean(listener?.listenerConnected);
			})();
		}, 800);
	}

	function bankIdForAccount(accountId: string): string {
		return settings.bankBindings.find((b) => b.accountId === accountId)?.bankId ?? '';
	}

	/** Account → bank: one bank maps to at most one account. */
	function setAccountBank(accountId: string, bankId: string) {
		let bankBindings = settings.bankBindings.filter((b) => b.accountId !== accountId);
		if (bankId) {
			const packageName = packageForBankId(bankId);
			if (!packageName) {
				toast($_('bankNotifications.bank.none'));
				return;
			}
			bankBindings = bankBindings.filter((b) => b.bankId !== bankId);
			bankBindings = [...bankBindings, { bankId, packageName, accountId }];
		}
		persist({ ...settings, bankBindings });
		toast($_('common.saved'));
	}

	function onNewCardAccount(accountId: string) {
		newCardAccountId = accountId;
		const bound = bankIdForAccount(accountId);
		if (bound) {
			newCardBankId = bound;
			return;
		}
		const acc = accounts.find((a) => a.id === accountId);
		if (acc?.bank_id && KNOWN_BANK_APPS.some((b) => b.bankId === acc.bank_id)) {
			newCardBankId = acc.bank_id;
		}
	}

	function addCard() {
		const last4 = normalizeLast4(newCardLast4);
		if (last4.length !== 4) {
			toast($_('bankNotifications.card.last4Invalid'));
			return;
		}
		if (!newCardAccountId) {
			toast($_('bankNotifications.card.accountRequired'));
			return;
		}
		if (!newCardBankId) {
			toast($_('bankNotifications.card.bankRequired'));
			return;
		}
		const without = settings.cardBindings.filter(
			(c) => !(c.bankId === newCardBankId && c.last4 === last4)
		);
		const card: CardBinding = {
			bankId: newCardBankId,
			last4,
			accountId: newCardAccountId
		};
		persist({ ...settings, cardBindings: [...without, card] });
		newCardLast4 = '';
		toast($_('common.saved'));
	}

	function removeCard(card: CardBinding) {
		persist({
			...settings,
			cardBindings: settings.cardBindings.filter(
				(c) => !(c.bankId === card.bankId && c.last4 === card.last4)
			)
		});
		toast($_('common.saved'));
	}
</script>

{#if loading}
	<p class="text-sm" style:color="var(--text-muted)">{$_('common.loading')}</p>
{:else}
	<div class="card space-y-6">
		<p class="text-sm" style:color="var(--text-muted)">{$_('bankNotifications.hint')}</p>

		{#if !isNativeApp()}
			<p class="text-sm" style:color="var(--text-muted)">{$_('bankNotifications.nativeOnly')}</p>
		{/if}

		<div class="flex items-center justify-between gap-4">
			<div>
				<p class="font-medium">{$_('bankNotifications.enable')}</p>
				<p class="text-sm" style:color="var(--text-muted)">{$_('bankNotifications.enableHint')}</p>
			</div>
			<ToggleSwitch
				label={$_('bankNotifications.enable')}
				checked={settings.enabled}
				disabled={!isNativeApp() || !$user}
				onchange={() => void toggleEnabled()}
			/>
		</div>

		<div class="space-y-2 border-t pt-4" style:border-color="var(--border)">
			<p class="font-medium">{$_('bankNotifications.access.title')}</p>
			<p class="text-sm" style:color="var(--text-muted)">
				{accessEnabled ? $_('bankNotifications.access.on') : $_('bankNotifications.access.off')}
			</p>
			<button
				type="button"
				class="btn-ghost"
				onclick={() => void openAccess()}
				disabled={!isNativeApp()}
			>
				{$_('bankNotifications.access.open')}
			</button>
		</div>

		<div class="space-y-2 border-t pt-4" style:border-color="var(--border)">
			<p class="font-medium">{$_('bankNotifications.listener.title')}</p>
			<p class="text-sm" style:color={listenerConnected ? 'var(--primary)' : 'var(--danger)'}>
				{listenerConnected
					? $_('bankNotifications.listener.on')
					: $_('bankNotifications.listener.off')}
			</p>
			<button
				type="button"
				class="btn"
				disabled={!isNativeApp() || reconnecting}
				onclick={() => void reconnectListener()}
			>
				{$_('bankNotifications.listener.reconnect')}
			</button>
		</div>

		<div class="space-y-2 border-t pt-4" style:border-color="var(--border)">
			<p class="font-medium">{$_('bankNotifications.background.title')}</p>
			<p class="text-sm" style:color="var(--text-muted)">
				{$_('bankNotifications.background.hint')}
			</p>
			<button
				type="button"
				class="btn-ghost"
				disabled={!isNativeApp()}
				onclick={() => void openBackgroundRestrictionsSettings()}
			>
				{$_('bankNotifications.background.open')}
			</button>
		</div>

		<div class="space-y-3 border-t pt-4" style:border-color="var(--border)">
			<p class="font-medium">{$_('bankNotifications.banks.title')}</p>
			<p class="text-sm" style:color="var(--text-muted)">{$_('bankNotifications.banks.hint')}</p>
			{#if bindableAccounts.length === 0}
				<p class="text-sm" style:color="var(--text-muted)">
					{$_('bankNotifications.accounts.empty')}
				</p>
			{:else}
				{#each bindableAccounts as acc (acc.id)}
					<label class="block space-y-1.5">
						<span class="text-sm font-medium">{acc.name}</span>
						<Select
							value={bankIdForAccount(acc.id)}
							options={bankOptions}
							onchange={(v) => setAccountBank(acc.id, v)}
						/>
					</label>
				{/each}
			{/if}
		</div>

		<div class="space-y-3 border-t pt-4" style:border-color="var(--border)">
			<p class="font-medium">{$_('bankNotifications.cards.title')}</p>
			<p class="text-sm" style:color="var(--text-muted)">{$_('bankNotifications.cards.hint')}</p>

			{#if settings.cardBindings.length}
				<ul class="space-y-2">
					{#each settings.cardBindings as card (card.bankId + card.last4)}
						<li
							class="flex items-center justify-between gap-2 rounded-lg border px-3 py-2"
							style:border-color="var(--border)"
						>
							<div class="min-w-0 text-sm">
								<p class="font-medium">
									{accounts.find((a) => a.id === card.accountId)?.name ?? card.accountId}
									· *{card.last4}
								</p>
								<p class="truncate" style:color="var(--text-muted)">{bankName(card.bankId)}</p>
							</div>
							<button type="button" class="btn-ghost shrink-0" onclick={() => removeCard(card)}>
								{$_('common.delete')}
							</button>
						</li>
					{/each}
				</ul>
			{/if}

			<div class="grid gap-2 sm:grid-cols-3">
				<label class="block space-y-1">
					<span class="text-xs" style:color="var(--text-muted)"
						>{$_('bankNotifications.card.account')}</span
					>
					<Select
						value={newCardAccountId}
						options={accountSelectOptions(bindableAccounts)}
						onchange={onNewCardAccount}
					/>
				</label>
				<label class="block space-y-1">
					<span class="text-xs" style:color="var(--text-muted)"
						>{$_('bankNotifications.card.last4')}</span
					>
					<input
						class="input w-full"
						inputmode="numeric"
						maxlength="4"
						placeholder="1234"
						bind:value={newCardLast4}
					/>
				</label>
				<label class="block space-y-1">
					<span class="text-xs" style:color="var(--text-muted)"
						>{$_('bankNotifications.card.bank')}</span
					>
					<select class="input w-full" bind:value={newCardBankId}>
						{#each KNOWN_BANK_APPS as bank (bank.bankId)}
							<option value={bank.bankId}>{bankName(bank.bankId)}</option>
						{/each}
					</select>
				</label>
			</div>
			<button type="button" class="btn" onclick={addCard}>{$_('bankNotifications.card.add')}</button
			>
		</div>
	</div>
{/if}
