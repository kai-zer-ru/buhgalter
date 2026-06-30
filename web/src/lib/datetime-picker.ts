import { formatDatePartsForDisplay, formatDisplayDateTimeShort } from '$lib/dates';

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
	return formatDatePartsForDisplay(parsed.year, parsed.month, parsed.day);
}

export function formatDatetimeButtonLabel(value: string): string {
	const parsed = parseDatetimeLocal(value);
	if (!parsed) return '';
	if (!value.includes('T')) {
		return formatDatePartsForDisplay(parsed.year, parsed.month, parsed.day);
	}
	return formatDisplayDateTimeShort(
		new Date(parsed.year, parsed.month - 1, parsed.day, parsed.hour, parsed.minute)
	);
}

export function daysInMonth(year: number, month: number): number {
	return new Date(year, month, 0).getDate();
}

export function weekdayMondayFirst(year: number, month: number, day: number): number {
	const js = new Date(year, month - 1, day).getDay();
	return js === 0 ? 6 : js - 1;
}

function shiftMonth(year: number, month: number, delta: number): { year: number; month: number } {
	let m = month + delta;
	let y = year;
	while (m < 1) {
		m += 12;
		y -= 1;
	}
	while (m > 12) {
		m -= 12;
		y += 1;
	}
	return { year: y, month: m };
}

export type CalendarCell = {
	day: number;
	month: number;
	year: number;
	inMonth: boolean;
};

export function calendarCells(year: number, month: number): CalendarCell[] {
	const firstWeekday = weekdayMondayFirst(year, month, 1);
	const totalDays = daysInMonth(year, month);
	const prev = shiftMonth(year, month, -1);
	const next = shiftMonth(year, month, 1);
	const prevMonthDays = daysInMonth(prev.year, prev.month);
	const cells: CalendarCell[] = [];

	for (let i = firstWeekday - 1; i >= 0; i--) {
		cells.push({ day: prevMonthDays - i, month: prev.month, year: prev.year, inMonth: false });
	}
	for (let day = 1; day <= totalDays; day++) {
		cells.push({ day, month, year, inMonth: true });
	}
	let nextDay = 1;
	while (cells.length % 7 !== 0) {
		cells.push({ day: nextDay++, month: next.month, year: next.year, inMonth: false });
	}
	// Keep calendar popover height stable: always render 6 full weeks (42 cells).
	while (cells.length < 42) {
		cells.push({ day: nextDay++, month: next.month, year: next.year, inMonth: false });
	}
	return cells;
}
