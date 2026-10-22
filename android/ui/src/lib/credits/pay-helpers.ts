import type { Credit, CreditPayment } from '$lib/api/client';
import { dateOnlyLocalValue, todayDateLocal, toDatetimeLocalValue } from '$lib/dates';
import { fromCents } from '$lib/money';

export function nextPendingPayment(c: Credit): CreditPayment | undefined {
	return c.schedule?.find((p) => !p.is_applied && p.kind === 'scheduled');
}

export function defaultPayAmount(c: Credit): string {
	const next = nextPendingPayment(c);
	let cents = next?.amount ?? c.next_payment_amount ?? c.monthly_payment;
	if (cents > c.remaining_amount) {
		cents = c.remaining_amount;
	}
	return fromCents(cents);
}

export function defaultPayDate(c: Credit, tz: string): string {
	const next = nextPendingPayment(c);
	if (next) {
		return dateOnlyLocalValue(toDatetimeLocalValue(next.payment_date, tz));
	}
	return todayDateLocal(tz);
}
