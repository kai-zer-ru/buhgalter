import {
	previewSubscriptionUpcoming,
	type SubscriptionPeriod,
	type SubscriptionPreviewUpcomingPayload
} from '$lib/api/client';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '$lib/dates';

export const UPCOMING_COUNT = 3;

export function upcomingToLocalDates(dates: string[] | undefined | null, tz: string): string[] {
	const out = (dates ?? [])
		.slice(0, UPCOMING_COUNT)
		.map((d) => toDatetimeLocalValue(d, tz).slice(0, 10));
	while (out.length < UPCOMING_COUNT) out.push('');
	return out;
}

export function upcomingLocalToAPI(localDates: string[], timeLocal: string, tz: string): string[] {
	const time = (timeLocal || '08:00').trim() || '08:00';
	return localDates.slice(0, UPCOMING_COUNT).map((d) => {
		const day = (d || '').split('T')[0] ?? '';
		return fromDatetimeLocalValue(`${day}T${time}`, tz);
	});
}

function addPeriodDays(isoDay: string, period: SubscriptionPeriod): string {
	const [y, m, d] = isoDay.split('-').map(Number);
	const dt = new Date(Date.UTC(y, (m ?? 1) - 1, d ?? 1));
	switch (period) {
		case 'week':
			dt.setUTCDate(dt.getUTCDate() + 7);
			break;
		case 'two_weeks':
			dt.setUTCDate(dt.getUTCDate() + 14);
			break;
		case 'month':
			dt.setUTCMonth(dt.getUTCMonth() + 1);
			break;
		case 'quarter':
			dt.setUTCMonth(dt.getUTCMonth() + 3);
			break;
		case 'half_year':
			dt.setUTCMonth(dt.getUTCMonth() + 6);
			break;
		case 'year':
			dt.setUTCFullYear(dt.getUTCFullYear() + 1);
			break;
	}
	const yy = dt.getUTCFullYear();
	const mm = String(dt.getUTCMonth() + 1).padStart(2, '0');
	const dd = String(dt.getUTCDate()).padStart(2, '0');
	return `${yy}-${mm}-${dd}`;
}

/** Offline / fallback seed: three local YYYY-MM-DD dates from start + period. */
export function seedUpcomingLocalDates(
	startDate: string,
	period: SubscriptionPeriod
): [string, string, string] {
	const first = (startDate || '').split('T')[0] || '';
	const second = addPeriodDays(first, period);
	const third = addPeriodDays(second, period);
	return [first, second, third];
}

export async function fetchUpcomingLocalDates(
	payload: SubscriptionPreviewUpcomingPayload,
	tz: string
): Promise<string[]> {
	try {
		const res = await previewSubscriptionUpcoming(payload);
		return upcomingToLocalDates(res.upcoming_run_ats, tz);
	} catch {
		return seedUpcomingLocalDates(payload.start_date, payload.period);
	}
}
