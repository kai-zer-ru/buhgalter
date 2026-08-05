import { beforeEach, describe, expect, it, vi } from 'vitest';
import { warmRefCache } from '$lib/offline/sync';
import { resetOutboxForTests } from '$lib/offline/store';
import * as client from '$lib/api/client';

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
		listCredits: vi.fn(),
		getCredit: vi.fn(),
		listBanks: vi.fn().mockResolvedValue([]),
		listRecurringOperations: ok,
		listSubscriptions: ok,
		getSubscriptionsSummary: ok,
		listDebts: ok,
		listMerchants: ok,
		listTags: ok,
		listTransactions: ok
	};
});

beforeEach(() => {
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
		await warmRefCache();

		expect(client.listCredits).toHaveBeenCalledWith({ status: 'active' });
		expect(client.listCredits).toHaveBeenCalledWith({ status: 'closed' });
		expect(client.getCredit).toHaveBeenCalledWith('c-active');
		expect(client.getCredit).toHaveBeenCalledWith('c-closed');
		expect(client.getCredit).toHaveBeenCalledWith('c-dup');
		expect(client.getCredit).toHaveBeenCalledTimes(3);
	});
});
