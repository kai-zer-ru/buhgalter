import { isNativeApp } from '$lib/platform/native';

/** Best-effort online check for native app sync decisions. */
export async function isNetworkOnline(): Promise<boolean> {
	if (!isNativeApp()) return true;
	try {
		const { Network } = await import('@capacitor/network');
		const status = await Network.getStatus();
		return status.connected;
	} catch {
		return typeof navigator !== 'undefined' ? navigator.onLine : true;
	}
}

export function shouldUseOfflineQueue(): boolean {
	return true;
}
