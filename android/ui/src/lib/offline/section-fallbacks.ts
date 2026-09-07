import type {
	Account,
	AccountBalanceSummary,
	Credit,
	Dashboard,
	Debt,
	Debtor,
	DebtorDetail,
	DebtsSummary,
	UIMeta
} from '$lib/api/client';
import { readAccountsFromOfflineCache, readRefCache } from '$lib/offline/ref-cache';

export function debtorDetailPath(id: string): string {
	return `/api/v1/debtors/${id}`;
}

export function readCachedDebts(): Debt[] {
	const byId = new Map<string, Debt>();
	const buckets = [
		readRefCache<Debt[]>('/api/v1/debts?settled=false'),
		readRefCache<Debt[]>('/api/v1/debts?settled=true'),
		readRefCache<Debt[]>('/api/v1/debts')
	];
	for (const rows of buckets) {
		if (!rows) continue;
		for (const debt of rows) {
			byId.set(debt.id, debt);
		}
	}
	return [...byId.values()];
}

export function findCachedDebt(id: string): Debt | null {
	const direct = readRefCache<Debt>(`/api/v1/debts/${id}`);
	if (direct) return direct;
	return readCachedDebts().find((debt) => debt.id === id) ?? null;
}

export function findCachedDebtor(id: string): Debtor | null {
	const fromList = readRefCache<Debtor[]>('/api/v1/debtors')?.find((row) => row.id === id);
	if (fromList) return fromList;
	const fromMeta = readRefCache<UIMeta>('/api/v1/ui/meta')?.debtors.find((row) => row.id === id);
	if (fromMeta) return fromMeta;
	const fromDebt = readCachedDebts().find((debt) => debt.debtor_id === id);
	if (!fromDebt) return null;
	return { id, name: fromDebt.debtor_name, created_at: fromDebt.created_at };
}

export function resolveDebtorDetailOffline(id: string): DebtorDetail | null {
	const cached = readRefCache<DebtorDetail>(debtorDetailPath(id));
	const debts = readCachedDebts().filter((debt) => debt.debtor_id === id);
	const debtor = findCachedDebtor(id);
	if (!cached && !debtor && debts.length === 0) return null;

	const mergedDebts = debts.length > 0 ? debts : (cached?.debts ?? []);
	const active = mergedDebts.filter((debt) => !debt.is_settled);
	let iOwe = 0;
	let owedToMe = 0;
	for (const debt of active) {
		if (debt.direction === 'borrowed') iOwe += debt.amount;
		else if (debt.direction === 'lent') owedToMe += debt.amount;
	}

	return {
		id,
		name: debtor?.name ?? cached?.name ?? mergedDebts[0]?.debtor_name ?? '',
		created_at: debtor?.created_at ?? cached?.created_at ?? mergedDebts[0]?.created_at ?? '',
		i_owe: iOwe,
		owed_to_me: owedToMe,
		debts: mergedDebts,
		transactions: cached?.transactions ?? []
	};
}

export function recomputeDebtsSummaryFromCache(): DebtsSummary | null {
	const activeCached = readRefCache<Debt[]>('/api/v1/debts?settled=false');
	const settledCached = readRefCache<Debt[]>('/api/v1/debts?settled=true');
	if (activeCached === null && settledCached === null) return null;

	const active = (activeCached ?? []).filter((debt) => !debt.is_settled);
	let iOwe = 0;
	let owedToMe = 0;
	let overdueIOwe = 0;
	let overdueOwedToMe = 0;
	for (const debt of active) {
		if (debt.direction === 'borrowed') {
			iOwe += debt.amount;
			if (debt.is_overdue) overdueIOwe += debt.amount;
		} else if (debt.direction === 'lent') {
			owedToMe += debt.amount;
			if (debt.is_overdue) overdueOwedToMe += debt.amount;
		}
	}
	return {
		i_owe: iOwe,
		owed_to_me: owedToMe,
		overdue_i_owe: overdueIOwe,
		overdue_owed_to_me: overdueOwedToMe,
		active_count: active.length
	};
}

export function findCachedAccount(id: string): Account | null {
	const direct = readRefCache<Account>(`/api/v1/accounts/${id}`);
	if (direct) return direct;
	for (const status of [undefined, 'active', 'archived', 'deleted'] as const) {
		const found = readAccountsFromOfflineCache(status)?.find((row) => row.id === id);
		if (found) return found;
	}
	return null;
}

export function findCachedAccountBalance(id: string): AccountBalanceSummary | null {
	const direct = readRefCache<AccountBalanceSummary>(`/api/v1/accounts/${id}/balance`);
	if (direct) return direct;
	const dash = readRefCache<Dashboard>('/api/v1/dashboard');
	return dash?.accounts.find((row) => row.id === id) ?? null;
}

export function findCachedCredit(id: string): Credit | null {
	const detail = readRefCache<Credit>(`/api/v1/credits/${id}`);
	if (detail) return detail;
	for (const path of [
		'/api/v1/credits?status=active',
		'/api/v1/credits?status=closed',
		'/api/v1/credits'
	]) {
		const found = readRefCache<Credit[]>(path)?.find((row) => row.id === id);
		if (found) return found;
	}
	const meta = readRefCache<UIMeta>('/api/v1/ui/meta');
	return (
		meta?.active_credits.find((row) => row.id === id) ??
		meta?.closed_credits.find((row) => row.id === id) ??
		null
	);
}
