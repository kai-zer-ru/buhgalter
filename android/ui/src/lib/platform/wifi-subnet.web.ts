import { WebPlugin } from '@capacitor/core';

export class WifiSubnetWeb extends WebPlugin {
	async getIpv4Subnet(): Promise<{ ip: string; prefix: number } | null> {
		return null;
	}

	async getSsid(): Promise<{ ssid: string | null; permissionDenied?: boolean }> {
		return { ssid: null };
	}

	async requestLocationPermission(): Promise<void> {
		// no-op on web
	}
}
