import { registerPlugin } from '@capacitor/core';
import { isNativeApp } from '$lib/platform/native';
import { clearLocalHistory, listLocalHistory, mergeHistoryLists } from './history-local';
import type { NotificationHistoryItem, RawBankNotification } from './types';

type CaptureState = {
	captureEnabled: boolean;
	notificationAccess: boolean;
	allowedPackages: string[];
	smsPermission?: boolean;
};

interface NotificationInterceptPlugin {
	isNotificationAccessEnabled(): Promise<{ enabled: boolean }>;
	openNotificationAccessSettings(): Promise<void>;
	openBackgroundRestrictionsSettings(): Promise<{
		opened: boolean;
		batteryOptimizationIgnored?: boolean;
	}>;
	setCaptureEnabled(opts: { enabled: boolean }): Promise<void>;
	setAllowedPackages(opts: { packages: string[] }): Promise<void>;
	setAllowedSmsSenders(opts: { senders: { sender: string; packageName: string }[] }): Promise<void>;
	getSmsPermissionStatus(): Promise<{ granted: boolean }>;
	requestSmsPermission(): Promise<{ granted: boolean }>;
	openAppPermissionSettings(): Promise<void>;
	getCaptureState(): Promise<CaptureState>;
	consumePending(): Promise<{ items: RawBankNotification[] }>;
	peekPending(): Promise<{ items: RawBankNotification[] }>;
	acknowledgePending(opts: { dedupeKeys: string[] }): Promise<void>;
	/** Preferred: JSON array string (reliable across OEM WebViews). */
	listHistory(): Promise<{ itemsJson?: string }>;
	/** Legacy; native maps this to listHistory. */
	getHistory(): Promise<{ items?: NotificationHistoryItem[]; itemsJson?: string }>;
	clearHistory(): Promise<void>;
	scanActiveNotifications(): Promise<{
		scanned: number;
		listenerConnected: boolean;
		notificationAccess?: boolean;
	}>;
	reconnectListener(): Promise<{ listenerConnected: boolean; notificationAccess: boolean }>;
	getListenerState(): Promise<{
		listenerConnected: boolean;
		notificationAccess: boolean;
		captureEnabled: boolean;
		smsPermission?: boolean;
	}>;
	addListener(
		eventName: 'pendingAvailable',
		listenerFunc: () => void
	): Promise<{ remove: () => void }>;
}

const Native = registerPlugin<NotificationInterceptPlugin>('NotificationIntercept');

export async function isNotificationAccessEnabled(): Promise<boolean> {
	if (!isNativeApp()) return false;
	try {
		const r = await Native.isNotificationAccessEnabled();
		return Boolean(r.enabled);
	} catch {
		return false;
	}
}

export async function openNotificationAccessSettings(): Promise<void> {
	if (!isNativeApp()) return;
	try {
		await Native.openNotificationAccessSettings();
	} catch {
		// ignore
	}
}

/** OEM autostart / battery screens (MIUI etc.) so NLS survives when the app is swiped away. */
export async function openBackgroundRestrictionsSettings(): Promise<boolean> {
	if (!isNativeApp()) return false;
	try {
		const r = await Native.openBackgroundRestrictionsSettings();
		return Boolean(r.opened);
	} catch {
		return false;
	}
}

export async function syncNativeCapture(opts: {
	enabled: boolean;
	packages: string[];
	smsSenders?: { sender: string; packageName: string }[];
}): Promise<void> {
	if (!isNativeApp()) return;
	try {
		await Native.setAllowedPackages({ packages: opts.packages });
		if (opts.smsSenders) {
			await Native.setAllowedSmsSenders({ senders: opts.smsSenders });
		}
		await Native.setCaptureEnabled({ enabled: opts.enabled });
	} catch {
		// ignore
	}
}

export async function getSmsPermissionGranted(): Promise<boolean> {
	if (!isNativeApp()) return false;
	try {
		const r = await Native.getSmsPermissionStatus();
		return Boolean(r.granted);
	} catch {
		return false;
	}
}

export async function requestSmsPermission(): Promise<boolean> {
	if (!isNativeApp()) return false;
	try {
		const r = await Native.requestSmsPermission();
		return Boolean(r.granted);
	} catch {
		return false;
	}
}

export async function openAppPermissionSettings(): Promise<void> {
	if (!isNativeApp()) return;
	try {
		await Native.openAppPermissionSettings();
	} catch {
		// ignore
	}
}

export async function getNativeCaptureState(): Promise<CaptureState | null> {
	if (!isNativeApp()) return null;
	try {
		return await Native.getCaptureState();
	} catch {
		return null;
	}
}

export async function consumeNativePending(): Promise<RawBankNotification[]> {
	if (!isNativeApp()) return [];
	try {
		const r = await Native.consumePending();
		return Array.isArray(r.items) ? r.items : [];
	} catch {
		return [];
	}
}

export async function peekNativePending(): Promise<RawBankNotification[]> {
	if (!isNativeApp()) return [];
	try {
		const r = await Native.peekPending();
		return Array.isArray(r.items) ? r.items : [];
	} catch {
		return [];
	}
}

export async function acknowledgeNativePending(dedupeKeys: string[]): Promise<void> {
	if (!isNativeApp() || !dedupeKeys.length) return;
	try {
		await Native.acknowledgePending({ dedupeKeys });
	} catch {
		// ignore
	}
}

type HistoryJsBridge = {
	peekJson?: () => string;
	clear?: () => void;
	listenerStateJson?: () => string;
};

/** Sync WebView interface — does not use Capacitor MessageHandler (can hang on OEM WebViews). */
function historyJsBridge(): HistoryJsBridge | null {
	if (typeof window === 'undefined') return null;
	const bridge = (window as Window & { BuhgalterNotificationHistory?: HistoryJsBridge })
		.BuhgalterNotificationHistory;
	// On some WebViews `typeof javaMethod` is not "function" — probe by call instead.
	if (!bridge) return null;
	return bridge;
}

function parseHistoryJson(raw: string): NotificationHistoryItem[] {
	try {
		const parsed: unknown = JSON.parse(raw || '[]');
		return Array.isArray(parsed) ? (parsed as NotificationHistoryItem[]) : [];
	} catch {
		return [];
	}
}

/**
 * Synchronous history read for the UI. Never awaits Capacitor (bridge can hang forever
 * and block the JS thread so loading spinners never clear).
 * Merges native prefs + JS mirror (filled when the pending queue is processed).
 */
export function readNotificationHistorySync(): NotificationHistoryItem[] {
	const local = listLocalHistory();
	if (!isNativeApp()) return local;

	let native: NotificationHistoryItem[] = [];
	const sync = historyJsBridge();
	if (sync) {
		try {
			// Call directly — on OEM WebViews `typeof javaMethod` may not be "function".
			native = parseHistoryJson(String(sync.peekJson?.() ?? '[]'));
		} catch {
			native = [];
		}
	}
	return mergeHistoryLists(native, local);
}

export async function getNotificationHistory(): Promise<NotificationHistoryItem[]> {
	return readNotificationHistorySync();
}

export async function clearNotificationHistory(): Promise<void> {
	clearLocalHistory();
	if (!isNativeApp()) return;
	const sync = historyJsBridge();
	if (sync) {
		try {
			sync.clear?.();
			return;
		} catch {
			// fall through
		}
	}
	try {
		await Native.clearHistory();
	} catch {
		// ignore
	}
}

export type NativeListenerDebugState = {
	listenerConnected: boolean;
	captureEnabled: boolean;
	notificationAccess: boolean;
	allowedPackageCount: number;
	historyCount: number;
};

/** Best-effort debug snapshot (sync bridge preferred). */
export function getNativeListenerDebugState(): NativeListenerDebugState | null {
	if (!isNativeApp()) return null;
	const sync = historyJsBridge();
	const localCount = listLocalHistory().length;
	if (!sync) {
		return {
			listenerConnected: false,
			captureEnabled: false,
			notificationAccess: false,
			allowedPackageCount: 0,
			historyCount: localCount
		};
	}
	try {
		const o = JSON.parse(
			String(sync.listenerStateJson?.() ?? '{}')
		) as Partial<NativeListenerDebugState>;
		const nativeCount = Number(o.historyCount) || 0;
		return {
			listenerConnected: Boolean(o.listenerConnected),
			captureEnabled: Boolean(o.captureEnabled),
			notificationAccess: Boolean(o.notificationAccess),
			allowedPackageCount: Number(o.allowedPackageCount) || 0,
			historyCount: Math.max(nativeCount, localCount)
		};
	} catch {
		return {
			listenerConnected: false,
			captureEnabled: false,
			notificationAccess: false,
			allowedPackageCount: 0,
			historyCount: localCount
		};
	}
}

export type ScanActiveResult = {
	scanned: number;
	listenerConnected: boolean;
};

/** Pull notifications currently in the shade into history (+ queue if allowlisted). */
export async function scanActiveNotifications(): Promise<ScanActiveResult> {
	if (!isNativeApp()) {
		return { scanned: 0, listenerConnected: false };
	}
	try {
		const r = await Native.scanActiveNotifications();
		return {
			scanned: typeof r.scanned === 'number' ? r.scanned : 0,
			listenerConnected: Boolean(r.listenerConnected)
		};
	} catch {
		return { scanned: 0, listenerConnected: false };
	}
}

export async function reconnectNotificationListener(): Promise<{
	listenerConnected: boolean;
	notificationAccess: boolean;
}> {
	if (!isNativeApp()) {
		return { listenerConnected: false, notificationAccess: false };
	}
	try {
		const r = await Native.reconnectListener();
		return {
			listenerConnected: Boolean(r.listenerConnected),
			notificationAccess: Boolean(r.notificationAccess)
		};
	} catch {
		return { listenerConnected: false, notificationAccess: false };
	}
}

export async function getNotificationListenerState(): Promise<{
	listenerConnected: boolean;
	notificationAccess: boolean;
	captureEnabled: boolean;
	smsPermission?: boolean;
} | null> {
	if (!isNativeApp()) return null;
	try {
		return await Native.getListenerState();
	} catch {
		return null;
	}
}

export async function addPendingAvailableListener(onEvent: () => void): Promise<() => void> {
	if (!isNativeApp()) return () => undefined;
	try {
		const handle = await Native.addListener('pendingAvailable', onEvent);
		return () => void handle.remove();
	} catch {
		return () => undefined;
	}
}
