import { get, writable } from 'svelte/store';
import { getServerProfile, isHomeSsid } from '$lib/platform/server-profile';
import {
	refreshActiveServerUrl,
	getServerUrl,
	setActiveServerUrlForProbe
} from '$lib/platform/server-url';
import { isNetworkOnline } from '$lib/offline/network';
import { abortTimeout } from '$lib/platform/abort-timeout';

export type ServerReachability = 'unknown' | 'online' | 'offline';

/** True when API calls should use cache/outbox instead of hitting the server. */
export const serverReachability = writable<ServerReachability>('unknown');

const PROBE_TIMEOUT_MS = 5_000;

/** Capacitor / fetch errors when the host is unreachable (device may still be "online"). */
export function isConnectionError(err: unknown): boolean {
	if (err && typeof err === 'object' && 'status' in err) {
		const status = Number((err as { status: number }).status);
		if (status === 0) return true;
	}
	if (err instanceof TypeError) return true;
	if (err instanceof Error) {
		const msg = err.message.toLowerCase();
		if (err.name === 'AbortError' || err.name === 'TimeoutError') return true;
		return (
			msg.includes('failed to connect') ||
			msg.includes('network request failed') ||
			msg.includes('network error') ||
			msg.includes('timeout') ||
			msg.includes('timed out')
		);
	}
	return false;
}

export function markServerOffline(): void {
	serverReachability.set('offline');
}

export function markServerOnline(): void {
	serverReachability.set('online');
}

export function isServerOfflineMode(): boolean {
	return get(serverReachability) === 'offline';
}

/** Whether mutation/sync paths may call the API (not cache-only reads). */
export async function shouldTryServer(): Promise<boolean> {
	if (!(await isNetworkOnline())) {
		markServerOffline();
		return false;
	}
	return get(serverReachability) === 'online';
}

async function probeOrigin(origin: string): Promise<boolean> {
	try {
		const res = await fetch(`${origin}/api/v1/health`, {
			method: 'GET',
			headers: { Accept: 'application/json' },
			credentials: 'omit',
			signal: abortTimeout(PROBE_TIMEOUT_MS)
		});
		return res.ok;
	} catch {
		return false;
	}
}

export async function probeServerReachability(): Promise<boolean> {
	const { ssid, mode } = await refreshActiveServerUrl();
	const origin = getServerUrl();
	if (!origin) {
		markServerOffline();
		return false;
	}
	if (!(await isNetworkOnline())) {
		markServerOffline();
		return false;
	}

	if (await probeOrigin(origin)) {
		markServerOnline();
		return true;
	}

	const profile = getServerProfile();
	if (
		profile.lanFallbackRemote &&
		profile.remoteUrl &&
		mode === 'lan' &&
		isHomeSsid(ssid, profile.homeSsids)
	) {
		setActiveServerUrlForProbe(profile.remoteUrl, 'remote');
		if (await probeOrigin(profile.remoteUrl)) {
			markServerOnline();
			return true;
		}
	}

	markServerOffline();
	return false;
}

let probeTimer: ReturnType<typeof setInterval> | null = null;

export function startServerProbeLoop(intervalMs = 60_000): void {
	if (probeTimer) return;
	void probeServerReachability();
	probeTimer = setInterval(() => void probeServerReachability(), intervalMs);
}

export function stopServerProbeLoopForTests(): void {
	if (probeTimer) {
		clearInterval(probeTimer);
		probeTimer = null;
	}
	serverReachability.set('unknown');
}
