import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
	warmRefCache,
	resetWarmRefCacheForTests,
	WARM_BACKGROUND_COOLDOWN_MS
} from '$lib/offline/sync';
import { resetOutboxForTests } from '$lib/offline/store';
import * as client from '$lib/api/client';

vi.mock('$lib/api/cache', () => ({
	invalidateApiCache: vi.fn()
}));

vi.mock('$lib/widgets/publish', () => ({
	publishWidgetSnapshot: vi.fn().mockResolvedValue(undefined),
	scheduleWidgetSnapshotPublish: vi.fn(),
	resetWidgetPublishForTests: vi.fn()
}));

vi.mock('$lib/api/client', async (importOriginal) => {
	const actual = await importOriginal<typeof import('$lib/api/client')>();
	return {
		...actual,
		getDashboard: vi.fn().mockResolvedValue({}),
		getUIMeta: vi.fn().mockResolvedValue({}),
		getDebtsSummary: vi.fn().mockResolvedValue({}),
		getBudgetSummary: vi.fn().mockResolvedValue({}),
		listAccounts: vi.fn().mockResolvedValue([]),
		listCredits: vi.fn(),
		getCredit: vi.fn(),
		listBanks: vi.fn().mockResolvedValue([]),
		listRecurringOperations: vi.fn().mockResolvedValue([]),
		listSubscriptions: vi.fn().mockResolvedValue([]),
		getSubscriptionsSummary: vi.fn().mockResolvedValue({}),
		listDebts: vi.fn().mockResolvedValue([]),
		listMerchants: vi.fn().mockResolvedValue([]),
		listTags: vi.fn().mockResolvedValue([]),
		listTransactions: vi.fn().mockResolvedValue({ data: [], meta: { total: 0 } })
	};
});

beforeEach(() => {
	resetWarmRefCacheForTests();
	resetOutboxForTests();
	vi.mocked(client.listCredits).mockReset();
	vi.mocked(client.getCredit).mockReset();
	vi.mocked(client.listBanks).mockReset().mockResolvedValue([]);
	vi.mocked(client.listCredits).mockImplementation(async (params?: { status?: string }) => {
		if (params?.status === 'closed') {
			return [
				{ id: 'c-closed', created_at: '1', updated_at: '1' } as client.Credit,
				{ id: 'c-dup', created_at: '1', updated_at: '1' } as client.Credit
			];
		}
		return [
			{ id: 'c-active', created_at: '1', updated_at: '1' } as client.Credit,
			{ id: 'c-dup', created_at: '1', updated_at: '1' } as client.Credit
		];
	});
	vi.mocked(client.getCredit).mockResolvedValue({ id: 'x' } as client.Credit);
});

describe('warmRefCache credit details', () => {
	it('fetches each credit detail once after listing active and closed', async () => {
		await warmRefCache({ force: true });

		expect(client.listCredits).toHaveBeenCalledWith({ status: 'active' });
		expect(client.listCredits).toHaveBeenCalledWith({ status: 'closed' });
		expect(client.getCredit).toHaveBeenCalledWith('c-active');
		expect(client.getCredit).toHaveBeenCalledWith('c-closed');
		expect(client.getCredit).toHaveBeenCalledWith('c-dup');
		expect(client.getCredit).toHaveBeenCalledTimes(3);
	});

	it('deduplicates concurrent warmRefCache calls', async () => {
		let bodies = 0;
		let resolveDash!: () => void;
		const gate = new Promise<void>((r) => {
			resolveDash = r;
		});
		vi.mocked(client.getDashboard).mockImplementation(async () => {
			bodies++;
			await gate;
			return {} as client.Dashboard;
		});

		const a = warmRefCache();
		const b = warmRefCache();
		resolveDash();
		await Promise.all([a, b]);

		expect(bodies).toBe(1);
	});

	it('skips background warm within cooldown', async () => {
		await warmRefCache();
		vi.mocked(client.getDashboard).mockClear();
		await warmRefCache({ background: true });
		expect(client.getDashboard).not.toHaveBeenCalled();
	});

	it('runs background warm after cooldown', async () => {
		await warmRefCache();
		vi.mocked(client.getDashboard).mockClear();
		vi.spyOn(Date, 'now').mockReturnValue(Date.now() + WARM_BACKGROUND_COOLDOWN_MS + 1);
		await warmRefCache({ background: true });
		expect(client.getDashboard).toHaveBeenCalled();
		vi.mocked(Date.now).mockRestore();
	});
});
