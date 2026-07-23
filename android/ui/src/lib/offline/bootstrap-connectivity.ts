import { get } from 'svelte/store';
import { isNetworkOnline } from '$lib/offline/network';
import {
	markServerOffline,
	probeServerReachability,
	serverReachability
} from '$lib/offline/server-connectivity';

/** Quick connectivity check before auth bootstrap — avoids long hangs on unreachable LAN URL. */
export async function prepareBootstrapConnectivity(): Promise<void> {
	if (!(await isNetworkOnline())) {
		markServerOffline();
		return;
	}
	// Await full probe (per-origin timeout in server-connectivity). Callers with a cached
	// /auth/me must not block the lock screen on this — run it in the background instead.
	const ok = await probeServerReachability();
	if (!ok && get(serverReachability) !== 'online') {
		markServerOffline();
	}
}
