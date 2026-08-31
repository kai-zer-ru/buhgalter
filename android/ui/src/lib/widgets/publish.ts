import { get } from 'svelte/store';
import {
	getBudgetSummary,
	getDashboard,
	listAccounts,
	listCredits,
	listDebts,
	listTransactions,
	type BudgetSummaryItem,
	type Dashboard,
	type Account,
	type Credit,
	type Debt,
	type Transaction
} from '$lib/api/client';
import { readRefCache } from '$lib/offline/ref-cache';
import { isAppLockEnabled } from '$lib/platform/app-lock';
import { getAuthToken } from '$lib/platform/auth-token';
import { getApiBase } from '$lib/platform/server-url';
import { isNativeApp } from '$lib/platform/native';
import { user } from '$lib/stores/auth';
import { clearWidgetBridge, publishWidgetBridge } from './bridge';
import { buildWidgetSnapshot } from './snapshot';

const WIDGET_PUBLISH_COOLDOWN_MS = 60_000;
const WIDGET_PUBLISH_DEBOUNCE_MS = 2_000;

let widgetPublishInflight: Promise<void> | null = null;
let lastWidgetPublishAt = 0;
let widgetPublishTimer: ReturnType<typeof setTimeout> | null = null;

function readCachedOrFetch<T>(path: string, fetcher: () => Promise<T>): Promise<T> {
	const cached = readRefCache<T>(path);
	if (cached !== null) return Promise.resolve(cached);
	return fetcher();
}

/** Coalesce widget publishes — one run per debounce window, cooldown between full fetches. */
export function scheduleWidgetSnapshotPublish(): void {
	if (!isNativeApp()) return;
	if (widgetPublishTimer !== null) clearTimeout(widgetPublishTimer);
	widgetPublishTimer = setTimeout(() => {
		widgetPublishTimer = null;
		void publishWidgetSnapshot();
	}, WIDGET_PUBLISH_DEBOUNCE_MS);
}

/** Fetch home-related APIs and push snapshot to native widgets. */
export async function publishWidgetSnapshot(): Promise<void> {
	if (!isNativeApp()) return;
	const baseUrl = getApiBase();
	const token = getAuthToken();
	if (!baseUrl || !token) {
		await clearWidgetBridge();
		return;
	}
	if (widgetPublishInflight) return widgetPublishInflight;
	if (Date.now() - lastWidgetPublishAt < WIDGET_PUBLISH_COOLDOWN_MS) return;

	const job = (async () => {
		try {
			const dashboardPath = '/api/v1/dashboard';
			const accountsPath = '/api/v1/accounts?status=active';
			const budgetPath = '/api/v1/budgets/summary';
			const creditsPath = '/api/v1/credits?status=active';
			const debtsPath = '/api/v1/debts?settled=false';
			const futurePath = '/api/v1/transactions?kind=future&limit=10&page=1&sort=date_asc';

			const [dashboard, accounts, budgetRes, credits, debts, futureRes] = await Promise.all([
				readCachedOrFetch<Dashboard>(dashboardPath, getDashboard),
				readCachedOrFetch<Account[]>(accountsPath, () => listAccounts('active')),
				readCachedOrFetch<{ items: BudgetSummaryItem[] }>(budgetPath, getBudgetSummary),
				readCachedOrFetch<Credit[]>(creditsPath, () => listCredits({ status: 'active' })),
				readCachedOrFetch<Debt[]>(debtsPath, () => listDebts({ settled: 'false' })),
				readCachedOrFetch<{ data: Transaction[] }>(futurePath, () =>
					listTransactions({
						kind: 'future',
						sort: 'date_asc',
						page: '1',
						limit: '10'
					})
				)
			]);
			const futureTx = Array.isArray(futureRes) ? futureRes : futureRes.data;
			const budgetItems = Array.isArray(budgetRes) ? budgetRes : budgetRes.items;
			const u = get(user);
			const snapshot = buildWidgetSnapshot({
				dashboard,
				accounts,
				budgetItems,
				credits,
				debts,
				futureTx,
				currency: u?.currency ?? 'RUB',
				language: u?.language ?? 'ru'
			});
			const lockEnabled = await isAppLockEnabled();
			await publishWidgetBridge({
				baseUrl,
				token,
				lockEnabled,
				snapshot
			});
			lastWidgetPublishAt = Date.now();
		} catch {
			// keep last good snapshot
		}
	})();

	widgetPublishInflight = job;
	try {
		await job;
	} finally {
		if (widgetPublishInflight === job) widgetPublishInflight = null;
	}
}

export function resetWidgetPublishForTests(): void {
	widgetPublishInflight = null;
	lastWidgetPublishAt = 0;
	if (widgetPublishTimer !== null) {
		clearTimeout(widgetPublishTimer);
		widgetPublishTimer = null;
	}
}

export async function clearWidgetsOnLogout(): Promise<void> {
	resetWidgetPublishForTests();
	await clearWidgetBridge();
}
