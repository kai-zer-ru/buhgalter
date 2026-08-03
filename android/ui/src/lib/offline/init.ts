import { Network } from '@capacitor/network';
import { warmRefCache } from '$lib/offline/sync';
import { getAuthToken } from '$lib/platform/auth-token';
import { hasServerUrl, refreshActiveServerUrl } from '$lib/platform/server-url';
import { probeServerReachability, startServerProbeLoop } from '$lib/offline/server-connectivity';
import { scheduleSyncOutbox } from '$lib/offline/sync';
import { hasPendingOutbox } from '$lib/offline/store';

function warmIfAuthenticated() {
	if (!getAuthToken()) return;
	void warmRefCache().catch(() => undefined);
}

/** Re-check /health and optionally sync when the device network becomes available. */
function onDeviceNetworkAvailable() {
	if (!hasServerUrl()) return;
	void refreshActiveServerUrl().then(() => {
		void probeServerReachability().then((online) => {
			if (!online) return;
			warmIfAuthenticated();
			if (hasPendingOutbox()) scheduleSyncOutbox();
		});
	});
}

export function initNativeOfflineSync() {
	if (!hasServerUrl()) return;

	void refreshActiveServerUrl().then(() => {
		startServerProbeLoop(60_000);

		void probeServerReachability().then((online) => {
			if (online) {
				warmIfAuthenticated();
				if (hasPendingOutbox()) scheduleSyncOutbox();
			}
		});
	});

	void Network.addListener('networkStatusChange', (status) => {
		if (!status.connected) return;
		// Fires on cellular→Wi‑Fi too (connectionType change while already "connected").
		onDeviceNetworkAvailable();
	});

	void import('@capacitor/app').then(({ App }) => {
		void App.addListener('appStateChange', ({ isActive }) => {
			if (!isActive) return;
			onDeviceNetworkAvailable();
		});
	});
}
