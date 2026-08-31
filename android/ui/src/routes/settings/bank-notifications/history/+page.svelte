<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { _ } from 'svelte-i18n';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import { toast } from '$lib/toast';
	import { bankIdForPackage } from '$lib/android/notification-intercept/banks';
	import { parseBankNotification } from '$lib/android/notification-intercept/parsers';
	import {
		clearNotificationHistory,
		getNativeListenerDebugState,
		getNotificationListenerState,
		processPendingBankNotifications,
		readNotificationHistorySync,
		reconnectNotificationListener,
		scanActiveNotifications,
		type NativeListenerDebugState,
		type NotificationHistoryItem
	} from '$lib/android/notification-intercept';
	import { isNativeApp } from '$lib/platform/native';

	let items = $state<NotificationHistoryItem[]>([]);
	let scanning = $state(false);
	let debug = $state<NativeListenerDebugState | null>(null);

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
</script>

{#if !isNativeApp()}
	<p class="text-sm" style:color="var(--text-muted)">{$_('bankNotifications.nativeOnly')}</p>
{:else}
	<p class="mb-4 text-sm" style:color="var(--text-muted)">{$_('bankNotifications.history.hint')}</p>
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
			{#each items as row (row.dedupeKey + String(row.postedAt))}
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
				</li>
			{/each}
		</ul>
	{/if}
{/if}
