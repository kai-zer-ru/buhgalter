import { describe, expect, it, vi } from 'vitest';
import {
	DEFAULT_API_PORT,
	buildLanOrigin,
	discoverLanServers,
	mergeDiscoveredServers,
	hostsInSubnet,
	isBuhgalterHealthPayload,
	parseIpv4,
	probeLanOrigin,
	subnetBaseFromIp
} from '$lib/platform/lan-discovery';

describe('isBuhgalterHealthPayload', () => {
	it('accepts valid health JSON', () => {
		expect(isBuhgalterHealthPayload({ status: 'ok', version: '1.4.0', db: 'connected' })).toBe(
			true
		);
	});

	it('rejects unrelated JSON', () => {
		expect(isBuhgalterHealthPayload({ foo: 'bar' })).toBe(false);
		expect(isBuhgalterHealthPayload(null)).toBe(false);
	});
});

describe('subnet helpers', () => {
	it('parses IPv4', () => {
		expect(parseIpv4('192.168.1.42')).toEqual([192, 168, 1, 42]);
		expect(parseIpv4('bad')).toBeNull();
	});

	it('builds /24 base and host list', () => {
		expect(subnetBaseFromIp('192.168.1.42')).toBe('192.168.1.0');
		const hosts = hostsInSubnet('192.168.1.0');
		expect(hosts).toHaveLength(254);
		expect(hosts[0]).toBe('192.168.1.1');
		expect(hosts[253]).toBe('192.168.1.254');
	});

	it('builds LAN origin on default port', () => {
		expect(buildLanOrigin('10.0.0.5', DEFAULT_API_PORT)).toBe('http://10.0.0.5:8765');
	});
});

describe('probeLanOrigin', () => {
	it('validates health response', async () => {
		const fetchMock = vi.fn(async () => ({
			ok: true,
			json: async () => ({
				status: 'ok',
				version: '1.4.0',
				db: 'connected',
				external_url: 'https://buhgalter-demo.example.com'
			})
		}));
		vi.stubGlobal('fetch', fetchMock);

		const hit = await probeLanOrigin('http://192.168.1.10:8765', 500);
		expect(hit).toEqual({
			origin: 'http://192.168.1.10:8765',
			version: '1.4.0',
			dbStatus: 'connected',
			latencyMs: expect.any(Number),
			externalUrl: 'https://buhgalter-demo.example.com'
		});

		vi.unstubAllGlobals();
	});
});

describe('mergeDiscoveredServers', () => {
	it('dedupes by origin and keeps fastest latency', () => {
		const merged = mergeDiscoveredServers(
			[
				{
					origin: 'http://192.168.1.10:8765',
					version: '1.0.0',
					dbStatus: 'connected',
					latencyMs: 20
				}
			],
			[
				{
					origin: 'http://192.168.1.10:8765',
					version: '1.0.0',
					dbStatus: 'connected',
					latencyMs: 5
				},
				{
					origin: 'http://192.168.1.11:8765',
					version: '1.0.0',
					dbStatus: 'connected',
					latencyMs: 10
				}
			]
		);
		expect(merged).toHaveLength(2);
		expect(merged[0].origin).toBe('http://192.168.1.10:8765');
		expect(merged[0].latencyMs).toBe(5);
	});
});

describe('discoverLanServers', () => {
	it('collects matches and sorts by latency', async () => {
		const probe = vi.fn(async (origin: string) => {
			if (origin.endsWith(':8765')) {
				return {
					origin,
					version: '1.0.0',
					dbStatus: 'connected',
					latencyMs: origin.includes('.2:') ? 5 : 20
				};
			}
			return null;
		});

		const found = await discoverLanServers({
			subnetBase: '192.168.0.0',
			probe,
			concurrency: 4,
			deadlineMs: 2000
		});

		expect(found).toHaveLength(254);
		expect(found[0].latencyMs).toBeLessThanOrEqual(found[1].latencyMs);
	});
});
