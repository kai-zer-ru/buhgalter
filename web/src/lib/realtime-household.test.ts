import { afterEach, describe, expect, it, vi } from 'vitest';

const refreshHouseholdMe = vi.fn().mockResolvedValue({ household: null });

vi.mock('$lib/household/membership', () => ({
	refreshHouseholdMe: () => refreshHouseholdMe()
}));

vi.mock('./ref-cache', async (importOriginal) => {
	const actual = await importOriginal<typeof import('./ref-cache')>();
	return {
		...actual,
		notifyRealtimeInvalidate: vi.fn(),
		notifyRealtimeInvalidatePaths: vi.fn(),
		setRealtimeLive: vi.fn()
	};
});

vi.mock('./tx-changes-catchup', () => ({
	catchUpAfterReconnect: vi.fn(),
	seedTransactionChangesCursor: vi.fn()
}));

describe('realtime invalidate refreshes household membership', () => {
	afterEach(async () => {
		const { resetRealtimeForTests } = await import('./realtime');
		const { resetRefCacheForTests } = await import('./ref-cache');
		resetRealtimeForTests();
		resetRefCacheForTests();
		refreshHouseholdMe.mockClear();
	});

	it('coarse invalidate (member joined/left) refreshes householdMe', async () => {
		const { deliverRealtimeMessageForTests } = await import('./realtime');
		deliverRealtimeMessageForTests(JSON.stringify({ v: 1, type: 'invalidate' }));
		await vi.waitFor(() => expect(refreshHouseholdMe).toHaveBeenCalledOnce());
	});

	it('household hint paths refresh householdMe', async () => {
		const { deliverRealtimeMessageForTests } = await import('./realtime');
		deliverRealtimeMessageForTests(
			JSON.stringify({
				v: 1,
				type: 'invalidate',
				hint_paths: ['/api/v1/households/me/invites']
			})
		);
		await vi.waitFor(() => expect(refreshHouseholdMe).toHaveBeenCalledOnce());
	});

	it('ledger-only hints do not refetch membership', async () => {
		const { deliverRealtimeMessageForTests } = await import('./realtime');
		deliverRealtimeMessageForTests(
			JSON.stringify({ v: 1, type: 'invalidate', hint_paths: ['/api/v1/transactions'] })
		);
		await new Promise((r) => setTimeout(r, 10));
		expect(refreshHouseholdMe).not.toHaveBeenCalled();
	});
});
