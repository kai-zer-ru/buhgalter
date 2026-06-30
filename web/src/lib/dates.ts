export const API_DATETIME_FORMAT = 'yyyy-MM-dd HH:mm:ss';

function pad2(n: number): string {
	return n.toString().padStart(2, '0');
}

function formatUTC(d: Date): string {
	return `${d.getUTCFullYear()}-${pad2(d.getUTCMonth() + 1)}-${pad2(d.getUTCDate())} ${pad2(d.getUTCHours())}:${pad2(d.getUTCMinutes())}:${pad2(d.getUTCSeconds())}`;
}

/** Wall-clock in tz → UTC Date (no external TZ library). */
function zonedComponentsToUtc(
	y: number,
	mo: number,
	d: number,
	h: number,
	mi: number,
	s: number,
	tz: string
): Date {
	const guess = Date.UTC(y, mo - 1, d, h, mi, s);
	const formatter = new Intl.DateTimeFormat('en-US', {
		timeZone: tz,
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit',
		hour12: false
	});
	const partsAt = (ms: number) => {
		const map: Record<string, string> = {};
		for (const p of formatter.formatToParts(new Date(ms))) {
			if (p.type !== 'literal') map[p.type] = p.value;
		}
		return {
			y: +map.year,
			mo: +map.month,
			d: +map.day,
			h: +map.hour,
			mi: +map.minute,
			s: +(map.second || '0')
		};
	};
	let utc = guess;
	for (let i = 0; i < 3; i++) {
		const got = partsAt(utc);
		const target = Date.UTC(y, mo - 1, d, h, mi, s);
		const gotMs = Date.UTC(got.y, got.mo - 1, got.d, got.h, got.mi, got.s);
		utc += target - gotMs;
	}
	return new Date(utc);
}

function parseAPIDateTime(s: string): Date {
	const m = s.match(/^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/);
	if (m) return new Date(Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], +m[6]));
	const d = new Date(s);
	if (Number.isNaN(d.getTime())) throw new Error('invalid datetime');
	return d;
}

/** UI datetime (interpreted in user TZ) → API UTC string. */
export function toAPIDateTime(date: Date, tz: string): string {
	const y = date.getFullYear();
	const mo = date.getMonth() + 1;
	const d = date.getDate();
	const h = date.getHours();
	const mi = date.getMinutes();
	const s = date.getSeconds();
	return formatUTC(zonedComponentsToUtc(y, mo, d, h, mi, s, tz));
}

/** API UTC string → Date for UI in user timezone. */
export function fromAPIDateTime(s: string, tz: string): Date {
	const utc = parseAPIDateTime(s);
	const formatter = new Intl.DateTimeFormat('en-US', {
		timeZone: tz,
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit',
		hour12: false
	});
	const map: Record<string, string> = {};
	for (const p of formatter.formatToParts(utc)) {
		if (p.type !== 'literal') map[p.type] = p.value;
	}
	return new Date(+map.year, +map.month - 1, +map.day, +map.hour, +map.minute, +(map.second ?? 0));
}

export function formatAPIDateTimeForDisplay(s: string, tz: string): string {
	try {
		const d = fromAPIDateTime(s, tz);
		return `${pad2(d.getDate())}.${pad2(d.getMonth() + 1)}.${d.getFullYear()} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
	} catch {
		return s;
	}
}

/** Payment schedule date with optional auto-debit local time (payment_date is often stored at 00:00). */
export function formatCreditPaymentDateForDisplay(
	paymentDate: string,
	tz: string,
	debitTimeLocal?: string | null
): string {
	try {
		const local = fromAPIDateTime(paymentDate, tz);
		if (local.getHours() !== 0 || local.getMinutes() !== 0) {
			return formatAPIDateTimeForDisplay(paymentDate, tz);
		}
		const debitTime = (debitTimeLocal ?? '').trim();
		if (/^\d{2}:\d{2}$/.test(debitTime)) {
			return `${formatAPIDateForDisplay(paymentDate, tz)} ${debitTime}`;
		}
		return formatAPIDateTimeForDisplay(paymentDate, tz);
	} catch {
		return paymentDate;
	}
}

export function formatAPIDateForDisplay(s: string, tz: string): string {
	try {
		const d = fromAPIDateTime(s, tz);
		return `${pad2(d.getDate())}.${pad2(d.getMonth() + 1)}.${d.getFullYear()}`;
	} catch {
		return s;
	}
}

/** @deprecated use isFutureDatetimeLocal */
export function isFutureInTZ(): boolean {
	return false;
}

/** True if datetime-local value is in the future (user TZ). */
export function isFutureDatetimeLocal(value: string, tz: string): boolean {
	if (!value) return false;
	try {
		const api = fromDatetimeLocalValue(value, tz);
		const utc = parseAPIDateTime(api);
		return utc.getTime() > Date.now();
	} catch {
		return false;
	}
}

export function toDatetimeLocalValue(s: string, tz: string): string {
	try {
		const d = fromAPIDateTime(s, tz);
		return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
	} catch {
		return '';
	}
}

export function fromDatetimeLocalValue(value: string, tz: string): string {
	const m = value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/);
	if (!m) throw new Error('invalid datetime-local');
	const local = new Date(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], 0);
	return toAPIDateTime(local, tz);
}

export function nowDatetimeLocal(tz: string): string {
	return toDatetimeLocalValue(formatUTC(new Date()), tz);
}

/** Date-only datetime-local (`T00:00`) in user TZ. */
export function todayDateLocal(tz: string): string {
	const v = nowDatetimeLocal(tz);
	const date = v.split('T')[0];
	return `${date}T00:00`;
}

/** Strip time from datetime-local value. */
export function dateOnlyLocalValue(value: string): string {
	const m = value.match(/^(\d{4}-\d{2}-\d{2})/);
	return m ? `${m[1]}T00:00` : value;
}

/** Date-only filter start → API UTC (00:00:00 in user TZ). */
export function fromDateLocalStart(value: string, tz: string): string {
	return fromDatetimeLocalValue(dateOnlyLocalValue(value), tz);
}

/** Date-only filter end → API UTC (23:59:59 in user TZ). */
export function fromDateLocalEnd(value: string, tz: string): string {
	const m = value.match(/^(\d{4})-(\d{2})-(\d{2})/);
	if (!m) throw new Error('invalid date-local');
	const local = new Date(+m[1], +m[2] - 1, +m[3], 23, 59, 59);
	return toAPIDateTime(local, tz);
}
