export const INVITE_ALPHABET = 'ABCDEFGHJKMNPQRSTUVWXYZ23456789';
export const INVITE_CODE_LEN = 9;
export const APP_INVITE_SCHEME = 'ru.kai_zer.buhgalter';

/** Uppercase, drop spaces/hyphens, keep A–Z and digits. */
export function compactInviteCode(raw: string): string {
	let out = '';
	for (const ch of raw.toUpperCase()) {
		if (ch === '-' || ch === ' ' || ch === '\t' || ch === '\n') continue;
		if (/[0-9A-Z]/.test(ch)) out += ch;
	}
	return out;
}

export function isCompleteInviteCode(raw: string): boolean {
	const compact = compactInviteCode(raw);
	if (compact.length !== INVITE_CODE_LEN) return false;
	for (const ch of compact) {
		if (!INVITE_ALPHABET.includes(ch)) return false;
	}
	return true;
}

/** Display as `XXX-XXX-XXX` while typing. */
export function formatInviteCode(raw: string): string {
	const compact = compactInviteCode(raw).slice(0, INVITE_CODE_LEN);
	if (compact.length <= 3) return compact;
	if (compact.length <= 6) return `${compact.slice(0, 3)}-${compact.slice(3)}`;
	return `${compact.slice(0, 3)}-${compact.slice(3, 6)}-${compact.slice(6)}`;
}

export function appInviteUrl(code: string): string {
	return `${APP_INVITE_SCHEME}://invite/${formatInviteCode(code)}`;
}
