import { afterEach, describe, expect, it, vi } from 'vitest';

const clearRefCache = vi.fn();
const notifyRealtimeInvalidate = vi.fn();

vi.mock('./ref-cache', async (importOriginal) => {
	const actual = await importOriginal<typeof import('./ref-cache')>();
	return {
		...actual,
		clearRefCache: (...args: unknown[]) => clearRefCache(...args),
		notifyRealtimeInvalidate: (...args: unknown[]) => notifyRealtimeInvalidate(...args),
		setRealtimeLive: vi.fn()
	};
});

vi.mock('./tx-changes-catchup', () => ({
	catchUpAfterReconnect: vi.fn(),
	seedTransactionChangesCursor: vi.fn()
}));

describe('import realtime sync vs invalidate', () => {
	afterEach(async () => {
		const { resetRealtimeForTests } = await import('./realtime');
		const { resetRefCacheForTests } = await import('./ref-cache');
		resetRealtimeForTests();
		resetRefCacheForTests();
		clearRefCache.mockClear();
		notifyRealtimeInvalidate.mockClear();
	});

	it('runs import.done sync listener before invalidate echo is skipped', async () => {
		const { deliverRealtimeMessageForTests, subscribeImportRealtime } = await import('./realtime');
		const { markLocalMutation, wasRecentLocalMutation } = await import('./ref-cache');

		const seen: string[] = [];
		subscribeImportRealtime((ev) => {
			seen.push(ev.type);
			if (ev.type === 'import.done') {
				markLocalMutation();
			}
		});

		deliverRealtimeMessageForTests(
			JSON.stringify({
				v: 1,
				type: 'import.done',
				job_id: 'job-1',
				status: 'done'
			})
		);
		expect(seen).toEqual(['import.done']);
		expect(wasRecentLocalMutation()).toBe(true);

		deliverRealtimeMessageForTests(JSON.stringify({ v: 1, type: 'invalidate' }));
		expect(notifyRealtimeInvalidate).not.toHaveBeenCalled();
		expect(clearRefCache).not.toHaveBeenCalled();
	});

	it('invalidates when there was no recent local mutation', async () => {
		const { deliverRealtimeMessageForTests } = await import('./realtime');
		deliverRealtimeMessageForTests(JSON.stringify({ v: 1, type: 'invalidate' }));
		expect(notifyRealtimeInvalidate).toHaveBeenCalled();
		expect(clearRefCache).toHaveBeenCalled();
	});
});
