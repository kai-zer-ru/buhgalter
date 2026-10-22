export { CATEGORY_ICON_IDS } from './category-icons';

export function categoryIconUrl(icon: string): string {
	return `/icons/categories/${icon}.svg`;
}

export function bankIconUrl(iconPath: string): string {
	return `/${iconPath}`;
}

import { formatMoneyDisplay } from './money';

export function formatBalance(display: string, currency = 'RUB'): string {
	const symbol = currency === 'RUB' ? '₽' : currency;
	return `${formatMoneyDisplay(display)} ${symbol}`;
}
