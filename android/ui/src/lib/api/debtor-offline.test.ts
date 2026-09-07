import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('svelte-i18n', () => ({
	locale: { subscribe: () => () => {} }
}));

vi.mock('$lib/platform/server-url', () => ({
	getApiBase: () => 'http://test.local:8765',
	getServerUrl: () => 'http://test.local:8765'
}));

vi.mock('$lib/platform/auth-token', () => ({
	authHeaders: () => ({}),
	getAuthToken: () => 'token',
	getAuthServerOrigin: () => 'http://test.local:8765'
}));

vi.mock('$lib/platform/native', () => ({
	isNativeApp: () => false
}));

vi.mock('$lib/offline/network', () => ({
	shouldUseOfflineQueue: () => true
}));

vi.mock('$lib/offline/server-connectivity', () => ({
	isServerOfflineMode: () => true,
	isConnectionError: () => false,
	markServerOffline: vi.fn(),
	markServerOnline: vi.fn()
}));

vi.mock('$lib/auth/session-expired', () => ({
	notifySessionExpired: vi.fn(),
	shouldLogoutOnApi401: () => false
}));

vi.mock('$lib/platform/debug-log', () => ({
	logApiRequest: () => 0,
	logApiResponse: vi.fn(),
	debugLogInfo: vi.fn(),
	debugLogWarn: vi.fn()
}));

describe('getDebtor / getDebt offline fallbacks', () => {
	beforeEach(async () => {
		vi.resetModules();
		const { resetRefCacheForTests } = await import('$lib/offline/ref-cache');
		resetRefCacheForTests();
	});

	it('opens a debtor card from cached debt lists when GET /debtors/{id} was never warmed', async () => {
		const { writeRefCache } = await import('$lib/offline/ref-cache');
		writeRefCache('/api/v1/debtors', [
			{ id: 'person-1', name: 'Иван', created_at: '2026-01-01T00:00:00Z' }
		]);
		writeRefCache('/api/v1/debts?settled=false', [
			{
				id: 'debt-1',
				debtor_id: 'person-1',
				debtor_name: 'Иван',
				direction: 'lent',
				amount: 150_000,
				amount_display: '1500.00',
				affects_balance: false,
				debt_date: '2026-07-08 10:00:00',
				due_date: '2026-07-15 23:59:59',
				description: null,
				transaction_id: null,
				is_settled: false,
				settled_at: null,
				is_overdue: false,
				created_at: '2026-07-08T07:00:00Z'
			}
		]);
		writeRefCache('/api/v1/debts?settled=true', []);

		const { getDebtor, getDebt } = await import('./client');
		await expect(getDebtor('person-1')).resolves.toMatchObject({
			id: 'person-1',
			name: 'Иван',
			owed_to_me: 150_000,
			i_owe: 0,
			debts: [{ id: 'debt-1' }]
		});
		await expect(getDebt('debt-1')).resolves.toMatchObject({
			id: 'debt-1',
			debtor_id: 'person-1'
		});
	});

	it('opens an account from the list cache when GET /accounts/{id} is missing', async () => {
		const { writeRefCache } = await import('$lib/offline/ref-cache');
		writeRefCache('/api/v1/accounts?status=active', [
			{
				id: 'a1',
				name: 'Наличные',
				type: 'cash',
				bank_id: null,
				initial_balance: 0,
				balance: 100,
				balance_display: '1.00',
				status: 'active',
				is_primary: true,
				created_at: '',
				updated_at: ''
			}
		]);
		writeRefCache('/api/v1/dashboard', {
			total_balance: 100,
			total_forecast: 100,
			accounts: [
				{
					id: 'a1',
					name: 'Наличные',
					type: 'cash',
					balance: 100,
					balance_display: '1.00',
					forecast_balance: 100,
					forecast_display: '1.00',
					has_future_this_month: false,
					is_primary: true
				}
			],
			recent_transactions: [],
			debts_summary: {
				i_owe: 0,
				owed_to_me: 0,
				overdue_i_owe: 0,
				overdue_owed_to_me: 0,
				active_count: 0
			}
		});

		const { getAccount, getAccountBalance } = await import('./client');
		await expect(getAccount('a1')).resolves.toMatchObject({ id: 'a1', name: 'Наличные' });
		await expect(getAccountBalance('a1')).resolves.toMatchObject({ id: 'a1', balance: 100 });
	});
});
