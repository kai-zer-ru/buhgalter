/** Default Buhgalter API port (BUHGALTER_ADDR). */
export const DEFAULT_API_PORT = 8765;

export type HealthPayload = {
	status: string;
	version: string;
	db: string;
	external_url?: string;
};

export type DiscoveredServer = {
	origin: string;
	version: string;
	dbStatus: string;
	latencyMs: number;
	/** Configured public URL from admin settings (reverse proxy), if any. */
	externalUrl?: string;
};

export type DiscoverLanServersOptions = {
	/** Network base address, e.g. 192.168.1.0 */
	subnetBase: string;
	prefix?: number;
	port?: number;
	concurrency?: number;
	hostTimeoutMs?: number;
	deadlineMs?: number;
	/** Skip probing this host (usually the phone's own IP). */
	excludeIp?: string;
	probe?: (origin: string, timeoutMs: number) => Promise<DiscoveredServer | null>;
};

/** True when JSON looks like Buhgalter GET /api/v1/health. */
export function isBuhgalterHealthPayload(body: unknown): body is HealthPayload {
	if (!body || typeof body !== 'object') return false;
	const record = body as Record<string, unknown>;
	return (
		typeof record.status === 'string' &&
		typeof record.version === 'string' &&
		typeof record.db === 'string'
	);
}

/** Parse dotted IPv4 into four octets. */
export function parseIpv4(ip: string): [number, number, number, number] | null {
	const parts = ip.trim().split('.');
	if (parts.length !== 4) return null;
	const octets: number[] = [];
	for (const part of parts) {
		if (!/^\d{1,3}$/.test(part)) return null;
		const n = Number(part);
		if (n < 0 || n > 255) return null;
		octets.push(n);
	}
	return octets as [number, number, number, number];
}

/** /24 network base from device IP (MVP: only prefix 24). */
export function subnetBaseFromIp(ip: string, prefix = 24): string | null {
	const octets = parseIpv4(ip);
	if (!octets) return null;
	if (prefix !== 24) return null;
	return `${octets[0]}.${octets[1]}.${octets[2]}.0`;
}

/** Host addresses to scan in a /24 (1–254). */
export function hostsInSubnet(subnetBase: string, prefix = 24): string[] {
	const octets = parseIpv4(subnetBase);
	if (!octets || prefix !== 24) return [];
	const hosts: string[] = [];
	for (let host = 1; host <= 254; host++) {
		hosts.push(`${octets[0]}.${octets[1]}.${octets[2]}.${host}`);
	}
	return hosts;
}

export function buildLanOrigin(ip: string, port: number): string {
	return `http://${ip}:${port}`;
}

/** Probe one host; null if not Buhgalter or timeout. */
export async function probeLanHost(
	ip: string,
	port = DEFAULT_API_PORT,
	timeoutMs = 800
): Promise<DiscoveredServer | null> {
	return probeLanOrigin(buildLanOrigin(ip, port), timeoutMs);
}

export async function probeLanOrigin(
	origin: string,
	timeoutMs = 800
): Promise<DiscoveredServer | null> {
	const started = Date.now();
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), timeoutMs);
	try {
		const res = await fetch(`${origin}/api/v1/health`, {
			method: 'GET',
			headers: { Accept: 'application/json' },
			credentials: 'omit',
			signal: controller.signal
		});
		if (!res.ok) return null;
		const body: unknown = await res.json();
		if (!isBuhgalterHealthPayload(body)) return null;
		const externalUrl = body.external_url?.trim();
		return {
			origin,
			version: body.version,
			dbStatus: body.db,
			latencyMs: Date.now() - started,
			...(externalUrl ? { externalUrl } : {})
		};
	} catch {
		return null;
	} finally {
		clearTimeout(timer);
	}
}

async function mapWithConcurrency<T, R>(
	items: T[],
	limit: number,
	fn: (item: T) => Promise<R>
): Promise<R[]> {
	if (!items.length) return [];
	const results: R[] = new Array(items.length);
	let index = 0;

	async function worker() {
		while (index < items.length) {
			const current = index++;
			results[current] = await fn(items[current]);
		}
	}

	const workers = Array.from({ length: Math.min(limit, items.length) }, () => worker());
	await Promise.all(workers);
	return results;
}

/** Parallel subnet scan; sorted by latency ascending. */
export async function discoverLanServers(
	opts: DiscoverLanServersOptions
): Promise<DiscoveredServer[]> {
	const {
		subnetBase,
		prefix = 24,
		port = DEFAULT_API_PORT,
		concurrency = 24,
		hostTimeoutMs = 800,
		deadlineMs = 12_000,
		excludeIp,
		probe = (origin, timeout) => probeLanOrigin(origin, timeout)
	} = opts;

	const hosts = hostsInSubnet(subnetBase, prefix).filter((ip) => ip !== excludeIp);
	if (!hosts.length) return [];

	const deadline = Date.now() + deadlineMs;
	const found: DiscoveredServer[] = [];
	const origins = hosts.map((ip) => buildLanOrigin(ip, port));

	await mapWithConcurrency(origins, concurrency, async (origin) => {
		if (Date.now() >= deadline) return null;
		const hit = await probe(origin, hostTimeoutMs);
		if (hit) found.push(hit);
		return hit;
	});

	found.sort((a, b) => a.latencyMs - b.latencyMs || a.origin.localeCompare(b.origin));
	return found;
}

/** Merge discovery results; keep fastest probe per origin. */
export function mergeDiscoveredServers(...lists: DiscoveredServer[][]): DiscoveredServer[] {
	const byOrigin = new Map<string, DiscoveredServer>();
	for (const list of lists) {
		for (const server of list) {
			const existing = byOrigin.get(server.origin);
			if (!existing || server.latencyMs < existing.latencyMs) {
				byOrigin.set(server.origin, server);
			}
		}
	}
	const merged = [...byOrigin.values()];
	merged.sort((a, b) => a.latencyMs - b.latencyMs || a.origin.localeCompare(b.origin));
	return merged;
}
