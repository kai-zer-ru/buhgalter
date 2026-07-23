import { registerPlugin } from '@capacitor/core';
import { parseIpv4, subnetBaseFromIp } from '$lib/platform/lan-discovery';

export type WifiIpv4Subnet = {
	/** Device IPv4 on Wi‑Fi, e.g. 192.168.1.42 */
	ip: string;
	/** Network base for scanning, e.g. 192.168.1.0 */
	base: string;
	prefix: number;
};

export type WifiSsidResult = {
	ssid: string | null;
	permissionDenied?: boolean;
};

type WifiSubnetPlugin = {
	getIpv4Subnet(): Promise<{ ip: string; prefix: number } | null>;
	getSsid(options?: { requestPermission?: boolean }): Promise<{
		ssid: string | null;
		permissionDenied?: boolean;
	}>;
	requestLocationPermission(): Promise<void>;
};

const plugin = registerPlugin<WifiSubnetPlugin>('WifiSubnet', {
	web: () => import('$lib/platform/wifi-subnet.web').then((m) => new m.WifiSubnetWeb())
});

/** Best-effort Wi‑Fi IPv4 subnet from native layer; null on cellular or unsupported. */
export async function getWifiIpv4Subnet(): Promise<WifiIpv4Subnet | null> {
	try {
		const raw = await plugin.getIpv4Subnet();
		if (!raw?.ip || !parseIpv4(raw.ip)) return null;
		const prefix = raw.prefix > 0 ? raw.prefix : 24;
		const base = subnetBaseFromIp(raw.ip, prefix);
		if (!base) return null;
		return { ip: raw.ip, base, prefix };
	} catch {
		return null;
	}
}

export function normalizeSsid(raw: string | null): string | null {
	if (!raw) return null;
	const ssid = raw.trim();
	if (!ssid || ssid === '<unknown ssid>' || ssid === '0x' || ssid === 'unknown ssid') {
		return null;
	}
	return ssid;
}

/** Current Wi‑Fi SSID; null on cellular, denied permission, or unsupported. */
export async function getCurrentWifiSsid(opts?: {
	requestPermission?: boolean;
}): Promise<WifiSsidResult> {
	try {
		if (opts?.requestPermission) {
			await plugin.requestLocationPermission();
		}
		const raw = await plugin.getSsid({ requestPermission: Boolean(opts?.requestPermission) });
		return {
			ssid: normalizeSsid(raw?.ssid ?? null),
			permissionDenied: Boolean(raw?.permissionDenied)
		};
	} catch {
		return { ssid: null };
	}
}
