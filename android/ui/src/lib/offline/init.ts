import { Network } from '@capacitor/network';
import { warmRefCache } from '$lib/offline/sync';
import {
	flushRefCacheDisk,
	reconcileOfflineCatalogsOnUnlock,
	refCacheReadyAny
} from '$lib/offline/ref-cache';
import { getAuthToken } from '$lib/platform/auth-token';
import { hasServerUrl, refreshActiveServerUrl } from '$lib/platform/server-url';
import { probeServerReachability, startServerProbeLoop } from '$lib/offline/server-connectivity';
import { scheduleSyncOutbox } from '$lib/offline/sync';
import { hasPendingOutbox } from '$lib/offline/store';

const CORE_WARM_PATHS = [
	'/api/v1/dashboard',
	'/api/v1/transactions?kind=manual&limit=10&page=1&sort=date_desc',
	'/api/v1/transactions?kind=future&limit=10&page=1&sort=date_desc'
];

let listenersRegistered = false;
let syncStarted = false;

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
				flushRefCacheDisk();
				return;
			}
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
	// Defer one frame so first paint / tap handlers register before network storm.
	requestAnimationFrame(() => startProbeAndWarm(false));
}

/** @deprecated use initNativeOfflineSyncListeners + startOfflineSyncAfterUnlock */
export function initNativeOfflineSync() {
	initNativeOfflineSyncListeners();
	startOfflineSyncAfterUnlock();
}

export function resetNativeOfflineSyncForTests(): void {
	listenersRegistered = false;
	syncStarted = false;
}
