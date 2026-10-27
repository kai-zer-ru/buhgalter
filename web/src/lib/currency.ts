const STATS_CURRENCY_KEY = 'stats.currency';

export type CurrencyOption = { value: string; label: string };

export function currencySelectOptions(items: { code: string; symbol: string }[]): CurrencyOption[] {
	return items.map((item) => ({
		value: item.code,
		label: `${item.code} ${item.symbol}`
	}));
}

export type OwnFundsTotal = {
	currency: string;
	total_balance: number;
	total_forecast: number;
};

/** Own funds (cash/bank) grouped by account currency. Credit cards are excluded. */
export function ownFundsTotalsByCurrency(
	accounts:
		| {
				type: string;
				currency?: string;
				balance: number;
				forecast_balance: number;
		  }[]
		| null
		| undefined,
	profileCurrency: string
): OwnFundsTotal[] {
	const by = new Map<string, OwnFundsTotal>();
	for (const acc of accounts ?? []) {
		if (acc.type === 'credit_card') continue;
		const code = acc.currency || profileCurrency;
		const row = by.get(code) ?? { currency: code, total_balance: 0, total_forecast: 0 };
		row.total_balance += acc.balance;
		row.total_forecast += acc.forecast_balance;
		by.set(code, row);
	}
	return sortRowsByProfileCurrency([...by.values()], profileCurrency);
}

export function sortRowsByProfileCurrency<T extends { currency: string }>(
	rows: T[] | null | undefined,
	profileCurrency: string
): T[] {
	if (!rows?.length) return [];
	return [...rows].sort((a, b) => {
		if (a.currency === profileCurrency && b.currency !== profileCurrency) return -1;
		if (b.currency === profileCurrency && a.currency !== profileCurrency) return 1;
		return a.currency.localeCompare(b.currency);
	});
}

export function readStoredStatsCurrency(): string {
	try {
		return localStorage.getItem(STATS_CURRENCY_KEY) ?? '';
	} catch {
		return '';
	}
}

export function writeStoredStatsCurrency(code: string) {
	try {
		if (code) localStorage.setItem(STATS_CURRENCY_KEY, code);
		else localStorage.removeItem(STATS_CURRENCY_KEY);
	} catch {
		/* ignore quota / private mode */
	}
}

export function resolveStoredStatsCurrency(
	available: string[] | null | undefined,
	stored: string,
	fallback: string
): string {
	if (available?.length) {
		if (stored && available.includes(stored)) return stored;
		return available[0] ?? fallback;
	}
	return stored || fallback;
}

/** Active account currencies plus API `available_currencies`, profile first. */
export function mergeAvailableCurrencies(
	fromApi: string[] | null | undefined,
	accounts: { currency?: string }[] | null | undefined,
	profileCurrency: string
): string[] {
	const codes = new Set<string>();
	for (const code of fromApi ?? []) {
		if (code) codes.add(code);
	}
	for (const acc of accounts ?? []) {
		codes.add(acc.currency || profileCurrency);
	}
	if (codes.size === 0 && profileCurrency) codes.add(profileCurrency);
	return sortRowsByProfileCurrency(
		[...codes].map((currency) => ({ currency })),
		profileCurrency
	).map((row) => row.currency);
}
