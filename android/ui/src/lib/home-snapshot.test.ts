import { beforeEach, describe, expect, it, vi } from 'vitest';

const translations: Record<string, string> = {
	'common.loadFailed': 'Не удалось загрузить данные',
	'common.error': 'Ошибка',
	'common.server_unavailable': 'Сервер недоступен. Сессия сохранена — попробуйте обновить.'
};

vi.mock('svelte/store', async (importOriginal) => {
	const actual = await importOriginal<typeof import('svelte/store')>();
	return {
		...actual,
		get: () => (key: string) => translations[key] ?? key
	};
});

vi.mock('svelte-i18n', () => ({
	_: {}
}));

vi.mock('./toast', () => ({
	toast: {
		fromError: vi.fn()
	}
}));

vi.mock('$lib/platform/server-url', () => ({
	getServerUrl: () => 'http://test.local:8765'
}));

vi.mock('$lib/platform/server-profile', () => ({
	getServerProfile: () => ({
		lanUrl: 'http://lan.local:8765',
		remoteUrl: 'http://test.local:8765',
		homeSsids: [],
		lanFallbackRemote: false,
		trustedOrigins: []
	})
}));

import {
	OfflineCacheMissError,
	resetRefCacheForTests,
	runWithSuppressedRefCacheNotifications,
	setWarmRefCacheActive,
	writeRefCache
} from './offline/ref-cache';
import { toast } from './toast';
import {
	applyCachedDashboardSnapshot,
	hasHomeDashboardSnapshot,
	homeDashboardGateMessage,
	homeNeedsForegroundLoad,
	reportHomeListLoadFailure
} from './home-snapshot';

const shown = { total_balance: 100, accounts: [], debts_summary: { i_owe: 0 } };

describe('home snapshot', () => {
	beforeEach(() => {
		resetRefCacheForTests();
		vi.mocked(toast.fromError).mockClear();
	});

	it('keeps a shown dashboard on a failed refresh and does not take the foreground path', () => {
		expect(hasHomeDashboardSnapshot(shown)).toBe(true);
		expect(homeNeedsForegroundLoad(shown)).toBe(false);
		expect(
			homeDashboardGateMessage(new OfflineCacheMissError('/api/v1/dashboard'), shown)
		).toBeNull();
		expect(toast.fromError).not.toHaveBeenCalled();
	});

	it('treats a ref-cache dashboard as a snapshot when the screen has not rendered one yet', () => {
		writeRefCache('/api/v1/dashboard', shown);
		expect(hasHomeDashboardSnapshot(null)).toBe(true);
		expect(homeNeedsForegroundLoad(null)).toBe(false);
		expect(
			homeDashboardGateMessage(new Error('Failed to connect to /10.0.0.2:8765'), null)
		).toBeNull();
	});

	it('uses a foreground load and one gate, without a toast, when there is no snapshot', () => {
		expect(hasHomeDashboardSnapshot(null)).toBe(false);
		expect(homeNeedsForegroundLoad(null)).toBe(true);
		const miss = new OfflineCacheMissError('/api/v1/dashboard');
		expect(homeDashboardGateMessage(miss, null)).toBe(translations['common.server_unavailable']);
		expect(toast.fromError).not.toHaveBeenCalled();
	});

	it('does not toast cache-miss or connection errors from home lists', () => {
		reportHomeListLoadFailure(new OfflineCacheMissError('/api/v1/transactions?limit=10'));
		reportHomeListLoadFailure(new Error('Failed to connect to /10.0.0.2:8765'));
		expect(toast.fromError).not.toHaveBeenCalled();
	});

	it('toasts a list error that is not server-unavailable', () => {
		const err = new Error('Категория не найдена');
		reportHomeListLoadFailure(err);
		expect(toast.fromError).toHaveBeenCalledOnce();
		expect(toast.fromError).toHaveBeenCalledWith(err);
	});

	it('clears the gate from a dashboard written during suppress and during grace', async () => {
		const duringSuppress = { total_balance: 10, accounts: [], debts_summary: { i_owe: 0 } };
		const gated = { dashBase: null, loadError: 'Сервер недоступен', loading: true };

		await runWithSuppressedRefCacheNotifications(async () => {
			writeRefCache('/api/v1/dashboard', duringSuppress);
		});
		const afterSuppress = applyCachedDashboardSnapshot(gated);
		expect(afterSuppress.dashBase).toEqual(duringSuppress);
		expect(afterSuppress.loadError).toBeNull();
		expect(afterSuppress.loading).toBe(false);

		setWarmRefCacheActive(true);
		setWarmRefCacheActive(false);
		const duringGrace = { total_balance: 11, accounts: [], debts_summary: { i_owe: 0 } };
		writeRefCache('/api/v1/dashboard', duringGrace);
		const afterGrace = applyCachedDashboardSnapshot({
			dashBase: afterSuppress.dashBase,
			loadError: 'Сервер недоступен',
			loading: true
		});
		expect(afterGrace.dashBase).toEqual(duringGrace);
		expect(afterGrace.loadError).toBeNull();
		expect(afterGrace.loading).toBe(false);
	});
});
