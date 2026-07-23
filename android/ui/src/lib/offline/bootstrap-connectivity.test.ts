import { describe, expect, it, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { prepareBootstrapConnectivity } from '$lib/offline/bootstrap-connectivity';
import { isNetworkOnline } from '$lib/offline/network';
import {
	isServerOfflineMode,
	probeServerReachability,
	serverReachability,
	stopServerProbeLoopForTests
} from '$lib/offline/server-connectivity';
import { setServerUrl, clearServerUrl } from '$lib/platform/server-url';

vi.mock('$lib/offline/network', () => ({
	isNetworkOnline: vi.fn()
}));

vi.mock('$lib/offline/server-connectivity', async (importOriginal) => {
	const actual = await importOriginal<typeof import('$lib/offline/server-connectivity')>();
	return {
		...actual,
		probeServerReachability: vi.fn()
	};
});

beforeEach(() => {
	clearServerUrl();
	stopServerProbeLoopForTests();
	vi.mocked(isNetworkOnline).mockReset();
	vi.mocked(probeServerReachability).mockReset();
	setServerUrl('http://192.168.1.10:8765');
});

describe('prepareBootstrapConnectivity', () => {
	it('marks offline when device has no network', async () => {
		vi.mocked(isNetworkOnline).mockResolvedValue(false);

		await prepareBootstrapConnectivity();

		expect(isServerOfflineMode()).toBe(true);
		expect(probeServerReachability).not.toHaveBeenCalled();
	});

	it('marks offline when probe returns unreachable', async () => {
		vi.mocked(isNetworkOnline).mockResolvedValue(true);
		vi.mocked(probeServerReachability).mockImplementation(async () => {
			serverReachability.set('offline');
			return false;
		});

		await prepareBootstrapConnectivity();

		expect(isServerOfflineMode()).toBe(true);
	});

	it('keeps online when probe succeeds', async () => {
		vi.mocked(isNetworkOnline).mockResolvedValue(true);
		vi.mocked(probeServerReachability).mockImplementation(async () => {
			serverReachability.set('online');
			return true;
		});

		await prepareBootstrapConnectivity();

		expect(get(serverReachability)).toBe('online');
	});
});
