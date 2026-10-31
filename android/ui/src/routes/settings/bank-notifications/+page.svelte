<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { listAccounts, listBanks, type Account, type Bank } from '$lib/api/client';
	import Select from '$lib/components/Select.svelte';
	import RowActionsMenu from '$lib/components/RowActionsMenu.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { accountSelectOptions } from '$lib/select-options';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import {
		KNOWN_BANK_APPS,
		getCurrentInterceptSettings,
		getNotificationListenerState,
		getPostNotificationsGranted,
		getSmsPermissionGranted,
		isNotificationAccessEnabled,
		openAppPermissionSettings,
		openBackgroundRestrictionsSettings,
		openNotificationAccessSettings,
		packageForBankId,
		reconnectNotificationListener,
		requestPostNotificationsPermission,
		requestSmsPermission,
		saveInterceptSettings,
		setAccountBankBinding,
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
	let smsPermission = $state(false);
	let postNotifications = $state(false);
	let reconnecting = $state(false);
	let requestingSms = $state(false);
	let requestingPost = $state(false);
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
		smsPermission = await getSmsPermissionGranted();
		postNotifications = await getPostNotificationsGranted();
		const listener = await getNotificationListenerState();
		listenerConnected = Boolean(listener?.listenerConnected);
		if (listener?.smsPermission != null) {
			smsPermission = Boolean(listener.smsPermission);
		}
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

	async function toggleShadeNotifications() {
		if (!isNativeApp()) {
			toast($_('bankNotifications.nativeOnly'));
			return;
		}
		const next = !settings.shadeNotifications;
		persist({ ...settings, shadeNotifications: next });
		if (next && settings.enabled) {
			postNotifications = await getPostNotificationsGranted();
			if (!postNotifications) {
				await grantPostNotifications();
			}
		}
		toast($_('common.saved'));
	}

	async function grantPostNotifications() {
		requestingPost = true;
		try {
			postNotifications = await requestPostNotificationsPermission();
			if (!postNotifications) {
				toast($_('bankNotifications.shade.needPermission'));
				await openAppPermissionSettings();
			} else {
				toast($_('common.saved'));
			}
		} finally {
			requestingPost = false;
			postNotifications = await getPostNotificationsGranted();
		}
	}

	async function grantSms() {
		requestingSms = true;
		try {
			smsPermission = await requestSmsPermission();
			if (!smsPermission) {
				toast($_('bankNotifications.sms.needPermission'));
				await openAppPermissionSettings();
			} else {
				toast($_('common.saved'));
			}
		} finally {
			requestingSms = false;
			smsPermission = await getSmsPermissionGranted();
		}
	}

	function bankIdForAccount(accountId: string): string {
		return settings.bankBindings.find((b) => b.accountId === accountId)?.bankId ?? '';
	}

	/** Account → bank; same bank may be bound to several accounts. */
	function setAccountBank(accountId: string, bankId: string) {
		let opts: { bankId: string; packageName: string } | null = null;
		if (bankId) {
			const packageName = packageForBankId(bankId);
			if (!packageName) {
				toast($_('bankNotifications.bank.none'));
				return;
			}
			opts = { bankId, packageName };
		}
		persist({
			...settings,
			bankBindings: setAccountBankBinding(settings.bankBindings, accountId, opts)
		});
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

{#snippet status(ok: boolean, text: string, bad: 'danger' | 'muted' = 'muted')}
	<p class="mt-1 flex items-start gap-1.5 text-sm leading-snug">
		<span
			class="mt-1.5 h-2 w-2 shrink-0 rounded-full"
			style:background-color={ok
				? 'var(--primary)'
				: bad === 'danger'
					? 'var(--danger)'
					: 'var(--text-muted)'}
			aria-hidden="true"
		></span>
		<span
			style:color={ok ? 'var(--primary)' : bad === 'danger' ? 'var(--danger)' : 'var(--text-muted)'}
			>{text}</span
		>
	</p>
{/snippet}

{#if loading}
	<p class="text-sm" style:color="var(--text-muted)">{$_('common.loading')}</p>
{:else}
	<div class="space-y-8">
		<section class="space-y-3">
			<GroupHeading
				title={$_('bankNotifications.group.capture')}
				hint={$_('bankNotifications.hint')}
			/>
			{#if !isNativeApp()}
				<p
					class="rounded-xl border p-3 text-sm"
					style:border-color="color-mix(in srgb, var(--warning) 40%, var(--border))"
					style:color="var(--text-muted)"
				>
					{$_('bankNotifications.nativeOnly')}
				</p>
			{/if}
			<div class="card list-divided p-0 sm:p-0">
				<div class="flex items-center justify-between gap-4 px-4 py-3.5">
					<div class="min-w-0">
						<p class="font-medium">{$_('bankNotifications.enable')}</p>
						<p class="mt-0.5 text-sm leading-snug" style:color="var(--text-muted)">
							{$_('bankNotifications.enableHint')}
						</p>
					</div>
					<ToggleSwitch
						label={$_('bankNotifications.enable')}
						checked={settings.enabled}
						disabled={!isNativeApp() || !$user}
						onchange={() => void toggleEnabled()}
					/>
				</div>
				<div class="space-y-3 px-4 py-3.5">
					<div class="flex items-center justify-between gap-4">
						<div class="min-w-0">
							<p class="font-medium">{$_('bankNotifications.shade.enable')}</p>
							<p class="mt-0.5 text-sm leading-snug" style:color="var(--text-muted)">
								{$_('bankNotifications.shade.enableHint')}
							</p>
						</div>
						<ToggleSwitch
							label={$_('bankNotifications.shade.enable')}
							checked={settings.shadeNotifications !== false}
							disabled={!isNativeApp() || !$user || !settings.enabled}
							onchange={() => void toggleShadeNotifications()}
						/>
					</div>
					{#if settings.enabled && settings.shadeNotifications !== false}
						{@render status(
							postNotifications,
							postNotifications
								? $_('bankNotifications.shade.permissionOn')
								: $_('bankNotifications.shade.permissionOff')
						)}
						{#if !postNotifications}
							<button
								type="button"
								class="btn-ghost"
								disabled={!isNativeApp() || requestingPost}
								onclick={() => void grantPostNotifications()}
							>
								{$_('bankNotifications.shade.requestPermission')}
							</button>
						{/if}
					{/if}
				</div>
			</div>
		</section>

		<section class="space-y-3">
			<GroupHeading
				title={$_('bankNotifications.group.system')}
				hint={$_('bankNotifications.group.system_hint')}
			/>
			<div class="card list-divided p-0 sm:p-0">
				<div class="space-y-3 px-4 py-3.5">
					<div>
						<p class="font-medium">{$_('bankNotifications.access.title')}</p>
						{@render status(
							accessEnabled,
							accessEnabled
								? $_('bankNotifications.access.on')
								: $_('bankNotifications.access.off'),
							'danger'
						)}
					</div>
					<button
						type="button"
						class="btn-ghost"
						onclick={() => void openAccess()}
						disabled={!isNativeApp()}
					>
						{$_('bankNotifications.access.open')}
					</button>
				</div>
				<div class="space-y-3 px-4 py-3.5">
					<div>
						<p class="font-medium">{$_('bankNotifications.listener.title')}</p>
						{@render status(
							listenerConnected,
							listenerConnected
								? $_('bankNotifications.listener.on')
								: $_('bankNotifications.listener.off'),
							'danger'
						)}
					</div>
					<button
						type="button"
						class="btn-ghost"
						disabled={!isNativeApp() || reconnecting}
						onclick={() => void reconnectListener()}
					>
						{$_('bankNotifications.listener.reconnect')}
					</button>
				</div>
				<div class="space-y-3 px-4 py-3.5">
					<div>
						<p class="font-medium">{$_('bankNotifications.sms.title')}</p>
						<p class="mt-0.5 text-sm leading-snug" style:color="var(--text-muted)">
							{$_('bankNotifications.sms.hint')}
						</p>
						{@render status(
							smsPermission,
							smsPermission ? $_('bankNotifications.sms.on') : $_('bankNotifications.sms.off')
						)}
					</div>
					{#if !smsPermission}
						<div class="flex flex-wrap gap-2">
							<button
								type="button"
								class="btn-ghost"
								disabled={!isNativeApp() || requestingSms}
								onclick={() => void grantSms()}
							>
								{$_('bankNotifications.sms.request')}
							</button>
							<button
								type="button"
								class="btn-ghost"
								disabled={!isNativeApp()}
								onclick={() => void openAppPermissionSettings()}
							>
								{$_('bankNotifications.sms.openSettings')}
							</button>
						</div>
					{/if}
				</div>
				<div class="space-y-3 px-4 py-3.5">
					<div>
						<p class="font-medium">{$_('bankNotifications.background.title')}</p>
						<p class="mt-0.5 text-sm leading-snug" style:color="var(--text-muted)">
							{$_('bankNotifications.background.hint')}
						</p>
					</div>
					<button
						type="button"
						class="btn-ghost"
						disabled={!isNativeApp()}
						onclick={() => void openBackgroundRestrictionsSettings()}
					>
						{$_('bankNotifications.background.open')}
					</button>
				</div>
			</div>
		</section>

		<section class="space-y-3">
			<GroupHeading
				title={$_('bankNotifications.banks.title')}
				hint={$_('bankNotifications.banks.hint')}
			/>
			{#if bindableAccounts.length === 0}
				<p class="card text-sm" style:color="var(--text-muted)">
					{$_('bankNotifications.accounts.empty')}
				</p>
			{:else}
				<div class="card list-divided p-0 sm:p-0">
					{#each bindableAccounts as acc (acc.id)}
						<label class="block space-y-1.5 px-4 py-3">
							<span class="field-label">{acc.name}</span>
							<Select
								value={bankIdForAccount(acc.id)}
								options={bankOptions}
								onchange={(v) => setAccountBank(acc.id, v)}
							/>
						</label>
					{/each}
				</div>
			{/if}
		</section>

		<section class="space-y-3">
			<GroupHeading
				title={$_('bankNotifications.cards.title')}
				hint={$_('bankNotifications.cards.hint')}
			/>
			{#if settings.cardBindings.length}
				<ul class="card list-divided p-0 sm:p-0">
					{#each settings.cardBindings as card (card.bankId + card.last4)}
						<li class="flex items-center gap-3 py-2 pr-1.5 pl-4">
							<span class="card-glyph" aria-hidden="true">*{card.last4}</span>
							<div class="min-w-0 flex-1">
								<p class="truncate font-medium">
									{accounts.find((a) => a.id === card.accountId)?.name ?? card.accountId}
								</p>
								<p class="truncate text-xs" style:color="var(--text-muted)">
									{bankName(card.bankId)}
								</p>
							</div>
							<RowActionsMenu
								actions={[
									{
										icon: 'delete',
										label: $_('common.delete'),
										variant: 'danger',
										onclick: () => removeCard(card)
									}
								]}
							/>
						</li>
					{/each}
				</ul>
			{/if}
			<div class="card space-y-4">
				<h3 class="font-semibold">{$_('bankNotifications.card.add')}</h3>
				<div class="grid gap-3 sm:grid-cols-3">
					<label class="block space-y-1.5">
						<span class="field-label">{$_('bankNotifications.card.account')}</span>
						<Select
							value={newCardAccountId}
							options={accountSelectOptions(bindableAccounts)}
							onchange={onNewCardAccount}
						/>
					</label>
					<label class="block space-y-1.5">
						<span class="field-label">{$_('bankNotifications.card.last4')}</span>
						<input
							class="input w-full"
							inputmode="numeric"
							maxlength="4"
							placeholder="1234"
							bind:value={newCardLast4}
						/>
					</label>
					<label class="block space-y-1.5">
						<span class="field-label">{$_('bankNotifications.card.bank')}</span>
						<select class="input w-full" bind:value={newCardBankId}>
							{#each KNOWN_BANK_APPS as bank (bank.bankId)}
								<option value={bank.bankId}>{bankName(bank.bankId)}</option>
							{/each}
						</select>
					</label>
				</div>
				<div class="flex justify-end border-t pt-4" style:border-color="var(--border)">
					<button type="button" class="btn-primary" onclick={addCard}
						>{$_('bankNotifications.card.add')}</button
					>
				</div>
			</div>
		</section>
	</div>
{/if}

<style>
	.card-glyph {
		display: inline-flex;
		height: 2.5rem;
		min-width: 3.25rem;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		border-radius: 0.75rem;
		padding: 0 0.375rem;
		font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--primary);
		background-color: color-mix(in srgb, var(--primary) 12%, transparent);
	}
</style>
