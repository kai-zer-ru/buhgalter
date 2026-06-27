function parseISODateLocal(period: string): Date | null {
	const m = period.match(/^(\d{4})-(\d{2})-(\d{2})$/);
	if (!m) return null;
	return new Date(+m[1], +m[2] - 1, +m[3]);
}

function capitalize(s: string): string {
	if (!s) return s;
	return s.charAt(0).toUpperCase() + s.slice(1);
}

function stripRuYearSuffix(s: string): string {
	return s.replace(/\s*г\.?\s*$/, '');
}

function formatMonthYear(d: Date, locale: string): string {
	const raw = stripRuYearSuffix(
		new Intl.DateTimeFormat(locale, { month: 'long', year: 'numeric' }).format(d)
	);
	return locale.startsWith('ru') ? capitalize(raw) : raw;
}

function ruMonthInRange(d: Date, locale: string): string {
	const full = new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'long' }).format(d);
	return full.replace(/^\d+\s*/, '');
}

function formatDayMonth(d: Date, locale: string): string {
	return new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'long' }).format(d);
}

function formatDayMonthYear(d: Date, locale: string): string {
	return stripRuYearSuffix(
		new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'long', year: 'numeric' }).format(d)
	);
}

export function formatStatsPeriod(
	period: string,
	groupBy: 'day' | 'week' | 'month',
	locale: string
): string {
	if (groupBy === 'day') return period;

	const start = parseISODateLocal(period);
	if (!start) return period;

	if (groupBy === 'month') {
		return formatMonthYear(start, locale);
	}

	const end = new Date(start);
	end.setDate(end.getDate() + 6);

	const sameMonth =
		start.getMonth() === end.getMonth() && start.getFullYear() === end.getFullYear();
	const sameYear = start.getFullYear() === end.getFullYear();

	if (locale.startsWith('ru')) {
		if (sameMonth && sameYear) {
			return `${start.getDate()}-${end.getDate()} ${ruMonthInRange(end, locale)} ${end.getFullYear()}`;
		}
		if (sameYear) {
			return `${formatDayMonth(start, locale)} – ${formatDayMonthYear(end, locale)}`;
		}
		return `${formatDayMonthYear(start, locale)} – ${formatDayMonthYear(end, locale)}`;
	}

	if (sameMonth && sameYear) {
		const month = new Intl.DateTimeFormat(locale, { month: 'long' }).format(start);
		return `${month} ${start.getDate()}-${end.getDate()}, ${start.getFullYear()}`;
	}
	if (sameYear) {
		const monthDay = (d: Date) =>
			new Intl.DateTimeFormat(locale, { month: 'long', day: 'numeric' }).format(d);
		return `${monthDay(start)} – ${monthDay(end)}, ${end.getFullYear()}`;
	}
	return `${formatDayMonthYear(start, locale)} – ${formatDayMonthYear(end, locale)}`;
}
