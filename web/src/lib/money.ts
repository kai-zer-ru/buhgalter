const GROUP_SEP = ' ';

/** Insert space as thousands separator (e.g. 10000 → 10 000). */
export function addThousandsSep(intPart: string): string {
	if (!intPart) return '0';
	const negative = intPart.startsWith('-');
	const digits = (negative ? intPart.slice(1) : intPart).replace(/\D/g, '');
	if (!digits) return negative ? '-0' : '0';
	const grouped = digits.replace(/\B(?=(\d{3})+(?!\d))/g, GROUP_SEP);
	return negative ? `-${grouped}` : grouped;
}

/** Format decimal money string for display (API or parsed value → 10 000.00). */
export function formatMoneyDisplay(value: string): string {
	const trimmed = value.trim();
	if (!trimmed) return '';

	const negative = trimmed.startsWith('-');
	const raw = trimmed.replace(/\s/g, '').replace(',', '.').replace(/^-/, '');
	const dot = raw.indexOf('.');
	const intRaw = dot === -1 ? raw : raw.slice(0, dot);
	const fracRaw =
		dot === -1
			? ''
			: raw
					.slice(dot + 1)
					.replace(/\D/g, '')
					.slice(0, 2);

	const intPart = addThousandsSep((negative ? '-' : '') + (intRaw.replace(/\D/g, '') || '0'));
	if (dot !== -1 || fracRaw) {
		return `${intPart}.${fracRaw.padEnd(2, '0').slice(0, 2)}`;
	}
	return intPart;
}

/** Parse display/input string to kopecks (strips spaces). */
export function toCents(value: string): number {
	const s = value.trim().replace(/\s/g, '').replace(',', '.');
	if (!s) return 0;
	const negative = s.startsWith('-');
	const raw = negative ? s.slice(1) : s;
	const parts = raw.split('.');
	const rubles = parseInt(parts[0] || '0', 10);
	if (Number.isNaN(rubles)) throw new Error('invalid amount');
	let kopecks = 0;
	if (parts.length > 1) {
		const frac = parts[1];
		if (frac.length > 2) throw new Error('too many decimal places');
		const padded = (frac + '00').slice(0, 2);
		kopecks = parseInt(padded, 10);
		if (Number.isNaN(kopecks)) throw new Error('invalid amount');
	}
	const total = rubles * 100 + kopecks;
	return negative ? -total : total;
}

/** Format kopecks as display string with thousands separator. */
export function fromCents(cents: number): string {
	const negative = cents < 0;
	const abs = Math.abs(cents);
	const rubles = Math.floor(abs / 100);
	const kop = abs % 100;
	const intStr = addThousandsSep(String(rubles));
	const s = `${intStr}.${kop.toString().padStart(2, '0')}`;
	return negative ? `-${s}` : s;
}

export function roundMoney(value: number): number {
	return Math.round(value * 100) / 100;
}

/** Format while typing (allows incomplete decimals). */
export function formatMoneyLive(value: string): string {
	const trimmed = value.trim();
	if (!trimmed) return '';

	const negative = trimmed.startsWith('-');
	let raw = trimmed.replace(/\s/g, '').replace(',', '.');
	if (negative) raw = raw.slice(1);

	const dotIdx = raw.indexOf('.');
	const intDigits = (dotIdx === -1 ? raw : raw.slice(0, dotIdx)).replace(/\D/g, '');
	const fracDigits =
		dotIdx === -1
			? ''
			: raw
					.slice(dotIdx + 1)
					.replace(/\D/g, '')
					.slice(0, 2);

	let result = addThousandsSep(intDigits);
	if (dotIdx !== -1) {
		result += `.${fracDigits}`;
	}
	return negative ? `-${result}` : result;
}

/** Map caret index after live formatting (keeps edit position in the middle). */
export function mapMoneyInputCursor(value: string, cursor: number, formatted: string): number {
	if (!formatted) return 0;

	const clamped = Math.max(0, Math.min(cursor, value.length));
	const dotPos = value.slice(0, clamped).search(/[.,]/);
	const inFraction = dotPos !== -1;

	let intDigits = 0;
	let fracDigits = 0;

	for (let i = 0; i < clamped; i++) {
		const c = value[i];
		if (c === '-' || c === ' ') continue;
		if (c === '.' || c === ',') continue;
		if (!/\d/.test(c)) continue;
		if (inFraction && i > dotPos) fracDigits++;
		else intDigits++;
	}

	if (!inFraction) {
		if (intDigits === 0) return formatted.startsWith('-') ? 1 : 0;

		let digits = 0;
		for (let i = 0; i < formatted.length; i++) {
			const c = formatted[i];
			if (c === '.') return i;
			if (c === '-' || c === ' ') continue;
			if (/\d/.test(c)) {
				digits++;
				if (digits === intDigits) return i + 1;
			}
		}
		const dot = formatted.indexOf('.');
		return dot === -1 ? formatted.length : dot;
	}

	const dotIdx = formatted.indexOf('.');
	if (dotIdx === -1) return formatted.length;
	if (fracDigits === 0) return dotIdx + 1;

	let digits = 0;
	for (let i = dotIdx + 1; i < formatted.length; i++) {
		if (/\d/.test(formatted[i])) {
			digits++;
			if (digits === fracDigits) return i + 1;
		}
	}
	return formatted.length;
}

/** Normalize on blur: valid amount, 2 decimals, thousands separator. */
export function formatMoneyInput(value: string): string {
	if (!value.trim()) return '';
	try {
		return fromCents(toCents(value));
	} catch {
		return formatMoneyDisplay(value);
	}
}

/** API payload: plain decimal without spaces (server accepts both). */
export function toAPIAmount(value: string): string {
	const cents = toCents(value);
	const negative = cents < 0;
	const abs = Math.abs(cents);
	const rubles = Math.floor(abs / 100);
	const kop = abs % 100;
	const s = `${rubles}.${kop.toString().padStart(2, '0')}`;
	return negative ? `-${s}` : s;
}
