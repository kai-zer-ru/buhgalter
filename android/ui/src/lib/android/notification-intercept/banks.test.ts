import { describe, expect, it } from 'vitest';
import { KNOWN_BANK_APPS, allKnownPackages, bankIdForPackage } from './banks';

describe('KNOWN_BANK_APPS', () => {
	it('covers catalog banks with unique packages', () => {
		expect(KNOWN_BANK_APPS.length).toBeGreaterThanOrEqual(19);
		const packages = allKnownPackages();
		expect(new Set(packages).size).toBe(packages.length);
		const ids = KNOWN_BANK_APPS.map((b) => b.bankId);
		expect(new Set(ids).size).toBe(ids.length);
	});

	it('resolves known packages and aliases', () => {
		expect(bankIdForPackage('ru.alfabank.mobile.android')).toBe('alfabank');
		expect(bankIdForPackage('ru.vtb24.mobilebanking.android')).toBe('vtb');
		expect(bankIdForPackage('com.yandex.bank')).toBe('yandex');
		expect(bankIdForPackage('com.wildberries.ru')).toBe('wbbank');
		expect(bankIdForPackage('com.unknown')).toBeNull();
	});
});
