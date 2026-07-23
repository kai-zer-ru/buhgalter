import { Network } from '@capacitor/network';
import { warmRefCache } from '$lib/offline/sync';
import { hasServerUrl, refreshActiveServerUrl } from '$lib/platform/server-url';
import { probeServerReachability, startServerProbeLoop } from '$lib/offline/server-connectivity';
import { scheduleSyncOutbox } from '$lib/offline/sync';
import { hasPendingOutbox } from '$lib/offline/store';

/** Re-check /health and optionally sync when the device network becomes available. */
function onDeviceNetworkAvailable() {
	if (!hasServerUrl()) return;
	void refreshActiveServerUrl().then(() => {
		void probeServerReachability().then((online) => {
			if (!online) return;
			void warmRefCache().catch(() => undefined);
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
				void warmRefCache().catch(() => undefined);
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
