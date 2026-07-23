import { get } from 'svelte/store';
import {
	getBudgetSummary,
	getDashboard,
	listAccounts,
	listCredits,
	listDebts,
	listTransactions
} from '$lib/api/client';
import { isAppLockEnabled } from '$lib/platform/app-lock';
import { getAuthToken } from '$lib/platform/auth-token';
import { getApiBase } from '$lib/platform/server-url';
import { isNativeApp } from '$lib/platform/native';
import { user } from '$lib/stores/auth';
import { clearWidgetBridge, publishWidgetBridge } from './bridge';
import { buildWidgetSnapshot } from './snapshot';

/** Fetch home-related APIs and push snapshot to native widgets. */
export async function publishWidgetSnapshot(): Promise<void> {
	if (!isNativeApp()) return;
	const baseUrl = getApiBase();
	const token = getAuthToken();
	if (!baseUrl || !token) {
		await clearWidgetBridge();
		return;
	}

	try {
		const [dashboard, accounts, budgetRes, credits, debts, futureRes] = await Promise.all([
			getDashboard(),
			listAccounts('active'),
			getBudgetSummary(),
			listCredits({ status: 'active' }),
			listDebts({ settled: 'false' }),
			listTransactions({
				kind: 'future',
				sort: 'date_asc',
				page: '1',
				limit: '10'
			})
		]);
		const u = get(user);
		const snapshot = buildWidgetSnapshot({
			dashboard,
			accounts,
			budgetItems: budgetRes.items,
			credits,
			debts,
			futureTx: futureRes.data,
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
	} catch {
		// keep last good snapshot
	}
}

export async function clearWidgetsOnLogout(): Promise<void> {
	await clearWidgetBridge();
}
