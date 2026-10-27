export { CATEGORY_ICON_IDS } from './category-icons';

export function categoryIconUrl(icon: string): string {
	return `/icons/categories/${icon}.svg`;
}

export function bankIconUrl(iconPath: string): string {
	return `/${iconPath}`;
}

import { writable } from 'svelte/store';
import { formatMoneyDisplay } from './money';

export type CurrencyInfo = { code: string; symbol: string };

let currencyCatalog = new Map<string, string>();

/** Bumped when `GET /currencies` lands so MoneyDisplay re-reads symbols. */
export const currencyCatalogVersion = writable(0);

/** Last successful `GET /currencies` (tests seed this; production fills via `listCurrencies`). */
export function setCurrencyCatalog(items: CurrencyInfo[] | null | undefined) {
	currencyCatalog = new Map((items ?? []).map((item) => [item.code, item.symbol]));
	currencyCatalogVersion.update((n) => n + 1);
}

export function currencySymbol(code: string): string {
	return currencyCatalog.get(code) ?? code;
}

export function formatBalance(display: string, currency = 'RUB'): string {
	return `${formatMoneyDisplay(display)} ${currencySymbol(currency)}`;
}

/** Informational FX rate `to_cents / from_cents`, 6 decimal places (same as the API). */
export function formatTransferRate(fromCents: number, toCents: number): string | null {
	if (!(fromCents > 0) || !(toCents > 0)) return null;
	return (toCents / fromCents).toFixed(6);
}
