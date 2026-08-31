import { describe, expect, it } from 'vitest';
import {
	KNOWN_BANK_APPS,
	allKnownPackages,
	allKnownSmsSenderEntries,
	bankIdForPackage,
	bankIdForSmsSender,
	normalizeSmsSender
} from './banks';

describe('KNOWN_BANK_APPS', () => {
	it('covers catalog banks with unique packages', () => {
		expect(KNOWN_BANK_APPS.length).toBeGreaterThanOrEqual(19);
		const packages = allKnownPackages();
		expect(new Set(packages).size).toBe(packages.length);
		const ids = KNOWN_BANK_APPS.map((b) => b.bankId);
		expect(new Set(ids).size).toBe(ids.length);
		for (const b of KNOWN_BANK_APPS) {
			expect(Array.isArray(b.smsSenders)).toBe(true);
		}
	});

	it('resolves known packages and aliases', () => {
		expect(bankIdForPackage('ru.alfabank.mobile.android')).toBe('alfabank');
		expect(bankIdForPackage('ru.vtb24.mobilebanking.android')).toBe('vtb');
		expect(bankIdForPackage('com.yandex.bank')).toBe('yandex');
		expect(bankIdForPackage('com.wildberries.ru')).toBe('wbbank');
		expect(bankIdForPackage('com.unknown')).toBeNull();
	});

	it('resolves SMS senders to bankId', () => {
		expect(bankIdForSmsSender('900')).toBe('sberbank');
		expect(bankIdForSmsSender('T-Bank')).toBe('tinkoff');
		expect(bankIdForSmsSender('tinkoff')).toBe('tinkoff');
		expect(bankIdForSmsSender('Alfa-Bank')).toBe('alfabank');
		expect(bankIdForSmsSender('unknown-sender')).toBeNull();
	});

	it('normalizes phone-like senders', () => {
		expect(normalizeSmsSender('+79001234567')).toBe('9001234567');
		expect(normalizeSmsSender('T-Bank')).toBe('tbank');
		expect(allKnownSmsSenderEntries().some((e) => e.sender === '900')).toBe(true);
	});
});
