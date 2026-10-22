import { Network } from '@capacitor/network';
import {
	HOME_PAST_TRANSACTIONS_PATH,
	HOME_PLANNED_TRANSACTIONS_PATH
} from '$lib/api/transactions-path';
import { warmRefCache } from '$lib/offline/sync';
import {
	flushRefCacheDisk,
	reconcileOfflineCatalogsOnUnlock,
	refCacheReadyAny
} from '$lib/offline/ref-cache';
import { getAuthToken } from '$lib/platform/auth-token';
import { isBiometricPromptActive } from '$lib/platform/app-lock';
import { hasServerUrl, refreshActiveServerUrl } from '$lib/platform/server-url';
import { probeServerReachability, startServerProbeLoop } from '$lib/offline/server-connectivity';
import { scheduleSyncOutbox } from '$lib/offline/sync';
import { hasPendingOutbox } from '$lib/offline/store';

const CORE_WARM_PATHS = [
	'/api/v1/dashboard',
	HOME_PAST_TRANSACTIONS_PATH,
	HOME_PLANNED_TRANSACTIONS_PATH
];

const DISK_FLUSH_DEFER_MS = import.meta.env.MODE === 'test' ? 0 : 2_000;
const UNLOCK_WARM_DEFER_MS = import.meta.env.MODE === 'test' ? 0 : 400;

let listenersRegistered = false;
let syncStarted = false;
let diskFlushTimer: ReturnType<typeof setTimeout> | null = null;
let unlockWarmTimer: ReturnType<typeof setTimeout> | null = null;

function scheduleRefCacheDiskFlush() {
	if (diskFlushTimer !== null) return;
	diskFlushTimer = setTimeout(() => {
		diskFlushTimer = null;
		flushRefCacheDisk();
	}, DISK_FLUSH_DEFER_MS);
}

function cancelRefCacheDiskFlush() {
	if (diskFlushTimer === null) return;
	clearTimeout(diskFlushTimer);
	diskFlushTimer = null;
}

function warmIfAuthenticated(background = false) {
	if (!getAuthToken()) return;
	const useBackground = background || refCacheReadyAny(CORE_WARM_PATHS);
	void warmRefCache({ background: useBackground }).catch(() => undefined);
}

function startProbeAndWarm(background = false) {
	void refreshActiveServerUrl().then(() => {
		// /health every 3 min — enough for offline banner, does not fight UI.
		startServerProbeLoop(180_000);
		void probeServerReachability().then((online) => {
			if (!online) return;
			warmIfAuthenticated(background);
			if (hasPendingOutbox()) scheduleSyncOutbox();
		});
	});
}

/** Re-check /health and optionally sync when the device network becomes available. */
function onDeviceNetworkAvailable(background = false) {
	if (!hasServerUrl()) return;
	startProbeAndWarm(background);
}

/**
 * Register network/resume listeners only — no probe/warm until UI is unlocked.
 * Call {@link startOfflineSyncAfterUnlock} after PIN/biometrics (or when lock is off).
 */
export function initNativeOfflineSyncListeners() {
	if (!hasServerUrl() || listenersRegistered) return;
	listenersRegistered = true;

	void Network.addListener('networkStatusChange', (status) => {
		if (!status.connected) return;
		onDeviceNetworkAvailable(true);
	});

	void import('@capacitor/app').then(({ App }) => {
		void App.addListener('appStateChange', ({ isActive }) => {
			if (!isActive) {
				if (isBiometricPromptActive()) return;
				scheduleRefCacheDiskFlush();
				return;
			}
			cancelRefCacheDiskFlush();
			onDeviceNetworkAvailable(true);
		});
	});
}

/** First probe + warm after session unlock — keeps startup and PIN screen responsive. */
export function startOfflineSyncAfterUnlock() {
	if (!hasServerUrl() || syncStarted) return;
	syncStarted = true;
	// Re-seed form catalogs before warm — cold start after days offline must not wait on /health.
	reconcileOfflineCatalogsOnUnlock();
	// Defer so first paint / tap handlers register before network storm.
	if (unlockWarmTimer !== null) clearTimeout(unlockWarmTimer);
	unlockWarmTimer = setTimeout(() => {
		unlockWarmTimer = null;
		startProbeAndWarm(false);
	}, UNLOCK_WARM_DEFER_MS);
}

/** @deprecated use initNativeOfflineSyncListeners + startOfflineSyncAfterUnlock */
export function initNativeOfflineSync() {
	initNativeOfflineSyncListeners();
	startOfflineSyncAfterUnlock();
}

export function resetNativeOfflineSyncForTests(): void {
	listenersRegistered = false;
	syncStarted = false;
	cancelRefCacheDiskFlush();
	if (unlockWarmTimer !== null) {
		clearTimeout(unlockWarmTimer);
		unlockWarmTimer = null;
	}
}
