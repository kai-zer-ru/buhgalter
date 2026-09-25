<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { get } from 'svelte/store';
	import { _ } from 'svelte-i18n';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import { transactionNewPath, transferNewPath } from '$lib/android/form-routes';
	import { toast } from '$lib/toast';
	import { listMerchants } from '$lib/api/client';
	import { bankIdForPackage, walletIdForPackage } from '$lib/android/notification-intercept/banks';
	import { parseBankNotification } from '$lib/android/notification-intercept/parsers';
	import {
		canCreateFromHistory,
		clearNotificationHistory,
		getCurrentInterceptSettings,
		getNativeListenerDebugState,
		getNotificationListenerState,
		HISTORY_CREATE_FROM,
		historyTransferComplement,
		prefillFromHistoryItem,
		processPendingBankNotifications,
		readNotificationHistorySync,
		reconnectNotificationListener,
		scanActiveNotifications,
		setInterceptPrefill,
		setInterceptTransferPrefill,
		transferPrefillFromHistory,
		type NativeListenerDebugState,
		type NotificationHistoryItem
	} from '$lib/android/notification-intercept';
	import { isNativeApp } from '$lib/platform/native';

	let items = $state<NotificationHistoryItem[]>([]);
	let scanning = $state(false);
	let creatingKey = $state<string | null>(null);
	let debug = $state<NativeListenerDebugState | null>(null);
	let merchants = $state<{ id: string; name: string }[]>([]);

	function refresh() {
		if (!isNativeApp()) {
			items = [];
			debug = null;
			return;
		}
		// Fully synchronous — no await. Capacitor history calls were hanging the WebView
		// so «Загрузка…» never cleared (even with setTimeout watchdog).
		debug = getNativeListenerDebugState();
		items = readNotificationHistorySync();
	}

	onMount(() => {
		refresh();
		void listMerchants()
			.then((rows) => {
				merchants = rows.map((m) => ({ id: m.id, name: m.name }));
			})
			.catch(() => {
				merchants = [];
			});
	});

	async function clear() {
		await clearNotificationHistory();
		items = [];
		toast($_('bankNotifications.history.cleared'));
	}

	async function scanActive() {
		scanning = true;
		try {
			let result = await scanActiveNotifications();
			if (!result.listenerConnected) {
				const re = await reconnectNotificationListener();
				if (!re.listenerConnected) {
					toast.error($_('bankNotifications.history.scanNoListener'));
					refresh();
					return;
				}
				result = await scanActiveNotifications();
			}
			if (!result.listenerConnected) {
				toast.error($_('bankNotifications.history.scanNoListener'));
				refresh();
				return;
			}
			const added = await processPendingBankNotifications();
			refresh();
			toast(
				$_('bankNotifications.history.scanResult', { values: { n: result.scanned } }) +
					(added > 0
						? ` · ${get(_)('bankNotifications.drafts.toastNew', { values: { n: added } })}`
						: '')
			);
		} finally {
			scanning = false;
			refresh();
			const st = await getNotificationListenerState();
			if (st && !debug) {
				debug = {
					listenerConnected: st.listenerConnected,
					captureEnabled: st.captureEnabled,
					notificationAccess: st.notificationAccess,
					allowedPackageCount: 0,
					historyCount: items.length
				};
			}
		}
	}

	function walletLabel(packageName: string): string | null {
		const id = walletIdForPackage(packageName);
		if (!id) return null;
		return $_(`bankNotifications.wallet.${id}`);
	}

	function formatWhen(ms: number): string {
		if (!ms) return '—';
		try {
			return new Date(ms).toLocaleString();
		} catch {
			return String(ms);
		}
	}

	function statusLabel(row: NotificationHistoryItem): string {
		if (!row.inAllowlist) return $_('bankNotifications.history.status.notAllowlisted');
		const parsed = parseBankNotification(row);
		if (!parsed) return $_('bankNotifications.history.status.queuedNoParse');
		if (parsed.kind === 'income') return $_('bankNotifications.history.status.wouldParseIncome');
		if (parsed.kind === 'cancel') return $_('bankNotifications.history.status.wouldParseCancel');
		return $_('bankNotifications.history.status.wouldParse');
	}

	function body(row: NotificationHistoryItem): string {
		return [row.text, row.bigText].filter(Boolean).join('\n').trim() || '—';
	}

	function rowKey(row: NotificationHistoryItem): string {
		return row.dedupeKey || `${row.packageName}|${row.postedAt}|${row.title}|${row.text}`;
	}

	function isTransferPair(row: NotificationHistoryItem): boolean {
		return Boolean(historyTransferComplement(row, items, getCurrentInterceptSettings(), merchants));
	}

	async function createOperation(row: NotificationHistoryItem) {
		const key = rowKey(row);
		creatingKey = key;
		try {
			const prefill = await prefillFromHistoryItem(row, getCurrentInterceptSettings(), merchants);
			if (!prefill) {
				toast.error($_('bankNotifications.history.createFailed'));
				return;
			}
			setInterceptPrefill(prefill);
			await goto(
				resolve(
					transactionNewPath({
						type: prefill.type ?? 'expense',
						from: HISTORY_CREATE_FROM
					})
				)
			);
		} finally {
			creatingKey = null;
		}
	}

	async function createTransfer(row: NotificationHistoryItem) {
		const key = rowKey(row);
		creatingKey = key;
		try {
			if (!isTransferPair(row)) {
				toast.error($_('bankNotifications.drafts.pairInvalid'));
				return;
			}
			const prefill = transferPrefillFromHistory(
				row,
				items,
				getCurrentInterceptSettings(),
				merchants
			);
			if (!prefill) {
				toast.error($_('bankNotifications.drafts.pairInvalid'));
				return;
			}
			setInterceptTransferPrefill(prefill);
			await goto(resolve(transferNewPath({ from: HISTORY_CREATE_FROM })));
		} finally {
			creatingKey = null;
		}
	}
</script>

{#if !isNativeApp()}
	<p class="text-sm" style:color="var(--text-muted)">{$_('bankNotifications.nativeOnly')}</p>
{:else}
	<p class="mb-4 text-sm" style:color="var(--text-muted)">{$_('bankNotifications.history.hint')}</p>
	<p class="mb-4 text-sm" style:color="var(--text-muted)">
		{$_('bankNotifications.history.createHint')}
	</p>
	<p class="mb-4 text-sm" style:color="var(--text-muted)">
		{$_('bankNotifications.history.scanActiveHint')}
	</p>

	{#if debug}
		<p class="mb-4 text-xs" style:color="var(--text-muted)">
			{$_('bankNotifications.history.debug', {
				values: {
					listener: debug.listenerConnected
						? $_('bankNotifications.history.debugOn')
						: $_('bankNotifications.history.debugOff'),
					capture: debug.captureEnabled
						? $_('bankNotifications.history.debugOn')
						: $_('bankNotifications.history.debugOff'),
					access: debug.notificationAccess
						? $_('bankNotifications.history.debugOn')
						: $_('bankNotifications.history.debugOff'),
					packages: debug.allowedPackageCount,
					history: debug.historyCount
				}
			})}
		</p>
	{/if}

	<div class="mb-4 flex flex-wrap gap-2">
		<button type="button" class="btn" disabled={scanning} onclick={() => void scanActive()}>
			{$_('bankNotifications.history.scanActive')}
		</button>
		<button type="button" class="btn-ghost" disabled={scanning} onclick={() => refresh()}>
			{$_('bankNotifications.history.refresh')}
		</button>
		<button type="button" class="btn-ghost" onclick={() => void clear()} disabled={!items.length}>
			{$_('bankNotifications.history.clear')}
		</button>
	</div>

	{#if items.length === 0}
		<EmptyStateCard message={$_('bankNotifications.history.empty')} />
	{:else}
		<ul class="space-y-3">
			{#each items as row (rowKey(row))}
				<li class="card space-y-1.5 text-sm">
					<div class="flex flex-wrap items-baseline justify-between gap-2">
						<p class="font-semibold break-all">{row.packageName}</p>
						<p class="shrink-0 tabular-nums" style:color="var(--text-muted)">
							{formatWhen(row.postedAt)}
						</p>
					</div>
					{#if bankIdForPackage(row.packageName)}
						<p style:color="var(--text-muted)">
							{$_('bankNotifications.history.bankId')}: {bankIdForPackage(row.packageName)}
						</p>
					{:else if walletLabel(row.packageName)}
						<p style:color="var(--text-muted)">
							{$_('bankNotifications.history.walletId')}: {walletLabel(row.packageName)}
						</p>
					{/if}
					{#if row.channel}
						<p style:color="var(--text-muted)">
							{$_('bankNotifications.history.channel')}:
							{row.channel === 'sms'
								? $_('bankNotifications.history.channel.sms')
								: $_('bankNotifications.history.channel.push')}
						</p>
					{/if}
					{#if row.title}
						<p>
							<span style:color="var(--text-muted)"
								>{$_('bankNotifications.history.titleLabel')}:</span
							>
							{row.title}
						</p>
					{/if}
					<p class="whitespace-pre-wrap break-words">{body(row)}</p>
					<p
						class="text-xs font-medium"
						style:color={row.inAllowlist ? 'var(--primary)' : 'var(--danger)'}
					>
						{statusLabel(row)}
					</p>
					{#if canCreateFromHistory(row)}
						<div class="pt-1">
							{#if isTransferPair(row)}
								<button
									type="button"
									class="btn-primary"
									disabled={creatingKey === rowKey(row)}
									onclick={() => void createTransfer(row)}
								>
									{$_('bankNotifications.drafts.createTransfer')}
								</button>
							{:else}
								<button
									type="button"
									class="btn-primary"
									disabled={creatingKey === rowKey(row)}
									onclick={() => void createOperation(row)}
								>
									{$_('bankNotifications.drafts.create')}
								</button>
							{/if}
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
{/if}
