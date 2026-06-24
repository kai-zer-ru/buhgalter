export function pad2(n: number): string {
	return n.toString().padStart(2, '0');
}

export function parseDatetimeLocal(value: string): {
	year: number;
	month: number;
	day: number;
	hour: number;
	minute: number;
} | null {
	const match = value.match(/^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2}))?$/);
	if (!match) return null;
	return {
		year: +match[1],
		month: +match[2],
		day: +match[3],
		hour: +(match[4] ?? '0'),
		minute: +(match[5] ?? '0')
	};
}

export function buildDatetimeLocal(
	year: number,
	month: number,
	day: number,
	hour: number,
	minute: number
): string {
	return `${year}-${pad2(month)}-${pad2(day)}T${pad2(hour)}:${pad2(minute)}`;
}

export function formatDateButtonLabel(value: string): string {
	const parsed = parseDatetimeLocal(value);
	if (!parsed) return '';
	return `${pad2(parsed.day)}.${pad2(parsed.month)}.${parsed.year}`;
}

export function formatDatetimeButtonLabel(value: string): string {
	const parsed = parseDatetimeLocal(value);
	if (!parsed) return '';
	const date = `${pad2(parsed.day)}.${pad2(parsed.month)}.${parsed.year}`;
	if (value.includes('T')) {
		return `${date} ${pad2(parsed.hour)}:${pad2(parsed.minute)}`;
	}
	return date;
}

export function daysInMonth(year: number, month: number): number {
	return new Date(year, month, 0).getDate();
}

export function weekdayMondayFirst(year: number, month: number, day: number): number {
	const js = new Date(year, month - 1, day).getDay();
	return js === 0 ? 6 : js - 1;
}

export function calendarCells(
	year: number,
	month: number
): Array<{ day: number; inMonth: boolean }> {
	const firstWeekday = weekdayMondayFirst(year, month, 1);
	const totalDays = daysInMonth(year, month);
	const prevMonthDays = daysInMonth(year, month - 1);
	const cells: Array<{ day: number; inMonth: boolean }> = [];

	for (let i = firstWeekday - 1; i >= 0; i--) {
		cells.push({ day: prevMonthDays - i, inMonth: false });
	}
	for (let day = 1; day <= totalDays; day++) {
		cells.push({ day, inMonth: true });
	}
	let nextDay = 1;
	while (cells.length % 7 !== 0) {
		cells.push({ day: nextDay++, inMonth: false });
	}
	return cells;
}
