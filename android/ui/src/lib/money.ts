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

/** True when value has binary +/− (not only a leading unary minus). */
export function hasMoneyExpression(value: string): boolean {
	const s = value.trim().replace(/\s/g, '').replace(/,/g, '.');
	if (!s) return false;
	const body = s.startsWith('-') ? s.slice(1) : s;
	return /[+-]/.test(body);
}

/**
 * Parse a +/− expression into kopecks.
 * Grammar: term (('+'|'-') term)*; first term may have a leading unary minus.
 * Returns null if the expression is incomplete or invalid.
 */
export function evaluateMoneyExpression(value: string): number | null {
	const s = value.trim().replace(/\s/g, '').replace(/,/g, '.');
	if (!s) return null;

	let i = 0;
	const len = s.length;

	const parseTerm = (): number | null => {
		const start = i;
		if (i >= len || !/\d/.test(s[i])) return null;
		while (i < len && /\d/.test(s[i])) i++;
		if (i < len && s[i] === '.') {
			i++;
			const fracStart = i;
			while (i < len && /\d/.test(s[i])) i++;
			if (i - fracStart > 2) return null;
		}
		const literal = s.slice(start, i);
		if (!literal || literal === '.') return null;
		try {
			return toCentsLiteral(literal);
		} catch {
			return null;
		}
	};

	let unary = 1;
	if (s[i] === '-') {
		unary = -1;
		i++;
	} else if (s[i] === '+') {
		i++;
	}

	const first = parseTerm();
	if (first === null) return null;
	let total = unary * first;

	while (i < len) {
		const op = s[i];
		if (op !== '+' && op !== '-') return null;
		i++;
		const term = parseTerm();
		if (term === null) return null;
		total = op === '+' ? total + term : total - term;
	}

	return total;
}

/** Parse a single money literal (no expression) to kopecks. */
function toCentsLiteral(value: string): number {
	const s = value.trim().replace(/\s/g, '').replace(',', '.');
	if (!s) return 0;
	const negative = s.startsWith('-');
	const raw = negative ? s.slice(1) : s;
	const parts = raw.split('.');
	if (parts.length > 2) throw new Error('invalid amount');
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

/** Parse display/input string to kopecks (strips spaces; evaluates +/− expressions). */
export function toCents(value: string): number {
	if (hasMoneyExpression(value)) {
		const evaluated = evaluateMoneyExpression(value);
		if (evaluated === null) throw new Error('invalid amount');
		return evaluated;
	}
	return toCentsLiteral(value);
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

/** Live-format a single money term (no expression operators). */
function formatMoneyLiveTerm(value: string): string {
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

type ExprToken = { kind: 'term'; raw: string } | { kind: 'op'; op: '+' | '-' };

/** Split raw input into money terms and binary +/− operators (preserves trailing op). */
function tokenizeMoneyExpression(value: string): ExprToken[] | null {
	const s = value.replace(/\s/g, '').replace(/,/g, '.');
	if (!s) return null;

	const tokens: ExprToken[] = [];
	let i = 0;
	const len = s.length;

	// optional unary minus on first term
	let firstPrefix = '';
	if (s[i] === '-') {
		firstPrefix = '-';
		i++;
	}

	const readTerm = (prefix: string): string | null => {
		const start = i;
		if (i >= len) return prefix ? prefix : null;
		// empty term after unary minus is allowed while typing ("-")
		if (!/\d/.test(s[i]) && s[i] !== '.') {
			return prefix || null;
		}
		while (i < len && /\d/.test(s[i])) i++;
		if (i < len && s[i] === '.') {
			i++;
			while (i < len && /\d/.test(s[i])) i++;
		}
		const body = s.slice(start, i);
		if (!body && !prefix) return null;
		return prefix + body;
	};

	const first = readTerm(firstPrefix);
	if (first === null) return null;
	tokens.push({ kind: 'term', raw: first });

	while (i < len) {
		const op = s[i];
		if (op !== '+' && op !== '-') return null;
		i++;
		tokens.push({ kind: 'op', op });
		if (i >= len) break; // trailing operator while typing
		const term = readTerm('');
		if (term === null) return null;
		tokens.push({ kind: 'term', raw: term });
	}

	return tokens;
}

/** Format while typing (allows incomplete decimals and +/− expressions). */
export function formatMoneyLive(value: string): string {
	const trimmed = value.trim();
	if (!trimmed) return '';

	if (!hasMoneyExpression(trimmed)) {
		return formatMoneyLiveTerm(trimmed);
	}

	const tokens = tokenizeMoneyExpression(trimmed);
	if (!tokens) {
		// keep characters that look like an expression draft as much as possible
		return trimmed.replace(/\s/g, '').replace(/,/g, '.');
	}

	let result = '';
	for (const t of tokens) {
		if (t.kind === 'op') {
			result += t.op;
		} else {
			result += formatMoneyLiveTerm(t.raw);
		}
	}
	return result;
}

/** Significant chars for cursor mapping: digits, decimal, sign, operators (not spaces). */
function isSignificantMoneyChar(c: string): boolean {
	return c === '+' || c === '-' || c === '.' || c === ',' || /\d/.test(c);
}

/** Map caret index after live formatting (keeps edit position in the middle). */
export function mapMoneyInputCursor(value: string, cursor: number, formatted: string): number {
	if (!formatted) return 0;

	const clamped = Math.max(0, Math.min(cursor, value.length));

	if (hasMoneyExpression(value) || hasMoneyExpression(formatted)) {
		let significant = 0;
		for (let i = 0; i < clamped; i++) {
			if (isSignificantMoneyChar(value[i]) && value[i] !== ' ') significant++;
		}
		// skip spaces in count above already; commas count as decimal
		let seen = 0;
		for (let i = 0; i < formatted.length; i++) {
			const c = formatted[i];
			if (c === ' ') continue;
			if (!isSignificantMoneyChar(c)) continue;
			seen++;
			if (seen === significant) return i + 1;
		}
		return formatted.length;
	}

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

/** Value for MoneyInput: empty when zero/unset, otherwise formatted display. */
export function formatMoneyForInput(value: string): string {
	const trimmed = value.trim();
	if (!trimmed) return '';
	try {
		if (toCents(trimmed) === 0) return '';
	} catch {
		// fall through to display formatting
	}
	return formatMoneyDisplay(trimmed);
}

/** Normalize on blur: valid amount, 2 decimals, thousands separator; zero → empty. */
export function formatMoneyInput(value: string): string {
	if (!value.trim()) return '';
	try {
		const cents = toCents(value);
		if (cents === 0) return '';
		return fromCents(cents);
	} catch {
		// Incomplete expression (e.g. "7899+") — keep live formatting, do not invent a result.
		if (hasMoneyExpression(value)) {
			return formatMoneyLive(value);
		}
		const displayed = formatMoneyDisplay(value);
		try {
			if (toCents(displayed) === 0) return '';
		} catch {
			// keep formatted fallback
		}
		return displayed;
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

/** Keys of the custom Android money keypad (not the system soft keyboard). */
export type MoneyKeypadKey = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | '.' | '+' | '-' | 'backspace';

/**
 * Apply one keypad key at the caret, then live-format.
 * Backspace deletes the nearest significant character before the caret (skips spaces).
 */
export function applyMoneyKeypadKey(
	value: string,
	cursor: number,
	key: MoneyKeypadKey
): { value: string; cursor: number } {
	const clamped = Math.max(0, Math.min(cursor, value.length));

	if (key === 'backspace') {
		if (clamped <= 0) {
			const formatted = formatMoneyLive(value);
			return { value: formatted, cursor: 0 };
		}
		let delAt = clamped - 1;
		while (delAt >= 0 && value[delAt] === ' ') delAt--;
		if (delAt < 0) {
			const formatted = formatMoneyLive(value);
			return { value: formatted, cursor: 0 };
		}
		const raw = value.slice(0, delAt) + value.slice(clamped);
		const formatted = formatMoneyLive(raw);
		return { value: formatted, cursor: mapMoneyInputCursor(raw, delAt, formatted) };
	}

	const raw = value.slice(0, clamped) + key + value.slice(clamped);
	const formatted = formatMoneyLive(raw);
	return { value: formatted, cursor: mapMoneyInputCursor(raw, clamped + key.length, formatted) };
}
