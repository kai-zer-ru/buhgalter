import { describe, expect, it } from 'vitest';
import {
	appInviteUrl,
	compactInviteCode,
	formatInviteCode,
	isCompleteInviteCode
} from './invite-code';

describe('invite-code', () => {
	it('formats a complete code', () => {
		expect(formatInviteCode('ABCDEFGHJ')).toBe('ABC-DEF-GHJ');
		expect(formatInviteCode('abc-def-ghj')).toBe('ABC-DEF-GHJ');
	});

	it('formats a partial code while typing', () => {
		expect(formatInviteCode('ab')).toBe('AB');
		expect(formatInviteCode('abcd')).toBe('ABC-D');
		expect(formatInviteCode('abcdefg')).toBe('ABC-DEF-G');
	});

	it('rejects incomplete or invalid alphabet', () => {
		expect(isCompleteInviteCode('ABC-DEF-GH')).toBe(false);
		expect(isCompleteInviteCode('ABC-DEF-GHI')).toBe(false);
		expect(isCompleteInviteCode('ABC-DEF-GHJ')).toBe(true);
		expect(compactInviteCode(' abc-def ghj ')).toBe('ABCDEFGHJ');
	});

	it('builds the app scheme URL', () => {
		expect(appInviteUrl('ABCDEFGHJ')).toBe('ru.kai_zer.buhgalter://invite/ABC-DEF-GHJ');
	});
});
