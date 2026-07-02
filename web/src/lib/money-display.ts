import { formatBalance } from './finance';
import { formatMoneyDisplay, fromCents } from './money';

export type MoneyDisplayInput = {
	value?: string;
	cents?: number;
	currency?: string;
};

/** Format amount for read-only display (templates, titles, i18n values). */
export function formatMoneyForDisplay(input: MoneyDisplayInput): string {
	const raw = input.value ?? (input.cents != null ? fromCents(input.cents) : '');
	if (!raw) return '';
	if (input.currency) return formatBalance(raw, input.currency);
	return formatMoneyDisplay(raw);
}
