import { beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { pullFromServer, pullStatus, syncState } from '$lib/offline/sync';
import { resetOutboxForTests } from '$lib/offline/store';
import * as connectivity from '$lib/offline/server-connectivity';

vi.mock('$lib/api/cache', () => ({
	invalidateApiCache: vi.fn()
}));

vi.mock('$lib/widgets/publish', () => ({
	publishWidgetSnapshot: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('$lib/api/client', async (importOriginal) => {
	const actual = await importOriginal<typeof import('$lib/api/client')>();
	const ok = vi.fn().mockResolvedValue({});
	return {
		...actual,
		getDashboard: ok,
		getUIMeta: ok,
		getDebtsSummary: ok,
		getBudgetSummary: ok,
		listAccounts: ok,
		listCredits: ok,
		listDebts: ok,
		listTransactions: ok
	};
});

vi.mock('$lib/offline/server-connectivity', async (importOriginal) => {
	const actual = await importOriginal<typeof connectivity>();
	return {
		...actual,
		shouldTryServer: vi.fn(),
		probeServerReachability: vi.fn(),
		markServerOnline: vi.fn(),
		markServerOffline: vi.fn()
	};
});

beforeEach(() => {
	resetOutboxForTests();
	pullStatus.set({ kind: 'idle', lastOkAt: null, pendingCount: 0 });
	syncState.set('idle');
	vi.mocked(connectivity.shouldTryServer).mockReset();
	vi.mocked(connectivity.probeServerReachability).mockReset();
	vi.mocked(connectivity.markServerOnline).mockReset();
	vi.mocked(connectivity.markServerOffline).mockReset();
});

describe('pullFromServer offline reconnect', () => {
	it('probes when offline and stays offline if health fails', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(false);
		vi.mocked(connectivity.probeServerReachability).mockResolvedValue(false);

		const ok = await pullFromServer();

		expect(ok).toBe(false);
		expect(connectivity.probeServerReachability).toHaveBeenCalledOnce();
		expect(get(pullStatus).kind).toBe('offline');
		expect(connectivity.markServerOnline).not.toHaveBeenCalled();
	});

	it('probes when offline and syncs if health succeeds', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(false);
		vi.mocked(connectivity.probeServerReachability).mockResolvedValue(true);

		const ok = await pullFromServer();

		expect(ok).toBe(true);
		expect(connectivity.probeServerReachability).toHaveBeenCalledOnce();
		expect(get(pullStatus).kind).toBe('ok');
		expect(connectivity.markServerOnline).toHaveBeenCalled();
	});

	it('skips probe when already online', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);

		const ok = await pullFromServer();

		expect(ok).toBe(true);
		expect(connectivity.probeServerReachability).not.toHaveBeenCalled();
		expect(get(pullStatus).kind).toBe('ok');
	});
});
