import { registerPlugin } from '@capacitor/core';
import {
	buildLanOrigin,
	DEFAULT_API_PORT,
	probeLanOrigin,
	type DiscoveredServer
} from '$lib/platform/lan-discovery';

export const MDNS_SERVICE_TYPE = '_buhgalter._tcp';

type MdnsHost = {
	host: string;
	port: number;
};

type LanDiscoveryPlugin = {
	discover(options?: { timeoutMs?: number }): Promise<{ servers: MdnsHost[] }>;
};

const plugin = registerPlugin<LanDiscoveryPlugin>('LanDiscovery', {
	web: () => import('$lib/platform/mdns-discovery.web').then((m) => new m.LanDiscoveryWeb())
});

export type DiscoverMdnsOptions = {
	timeoutMs?: number;
	port?: number;
};

/** Browse mDNS for Buhgalter servers and validate via /api/v1/health. */
export async function discoverMdnsServers(
	opts: DiscoverMdnsOptions = {}
): Promise<DiscoveredServer[]> {
	const timeoutMs = opts.timeoutMs ?? 4000;
	const port = opts.port ?? DEFAULT_API_PORT;

	try {
		const raw = await plugin.discover({ timeoutMs });
		const hosts = raw?.servers ?? [];
		const found: DiscoveredServer[] = [];

		await Promise.all(
			hosts.map(async (entry) => {
				const hostPort = entry.port > 0 ? entry.port : port;
				const origin = buildLanOrigin(entry.host, hostPort);
				const hit = await probeLanOrigin(origin);
				if (hit) found.push(hit);
			})
		);

		found.sort((a, b) => a.latencyMs - b.latencyMs || a.origin.localeCompare(b.origin));
		return found;
	} catch {
		return [];
	}
}
