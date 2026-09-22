import { describe, expect, it } from 'vitest';
import {
	KNOWN_BANK_APPS,
	KNOWN_WALLET_APPS,
	allKnownPackages,
	allKnownSmsSenderEntries,
	bankIdForPackage,
	bankIdForSmsSender,
	inferBankIdFromText,
	isAllowlistedPackage,
	normalizeSmsSender,
	resolveRawBankNotification,
	walletIdForPackage
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
		expect(bankIdForPackage('ru.nspk.mirpay')).toBeNull();
	});
});

describe('KNOWN_WALLET_APPS', () => {
	it('covers wallets with unique packages', () => {
		expect(KNOWN_WALLET_APPS.length).toBeGreaterThanOrEqual(7);
		const packages = allKnownPackages();
		expect(new Set(packages).size).toBe(packages.length);
		const ids = KNOWN_WALLET_APPS.map((w) => w.walletId);
		expect(new Set(ids).size).toBe(ids.length);
		for (const w of KNOWN_WALLET_APPS) {
			expect(w.packageNames.length).toBeGreaterThan(0);
		}
	});

	it('resolves wallet packages and allowlist', () => {
		expect(walletIdForPackage('ru.nspk.mirpay')).toBe('mir_pay');
		expect(walletIdForPackage('ru.nspk.sbpay')).toBe('sbp_pay');
		expect(walletIdForPackage('com.samsung.android.spay')).toBe('samsung_pay');
		expect(walletIdForPackage('com.samsung.android.spaymini')).toBe('samsung_pay');
		expect(walletIdForPackage('ru.yandex.money')).toBe('yoomoney');
		expect(walletIdForPackage('ru.sberbankmobile')).toBeNull();
		expect(isAllowlistedPackage('ru.nspk.mirpay')).toBe(true);
		expect(isAllowlistedPackage('com.idamob.tinkoff.android')).toBe(true);
		expect(isAllowlistedPackage('com.unknown')).toBe(false);
	});

	it('infers issuer bank from wallet text, not marketplace brands', () => {
		expect(inferBankIdFromText('Оплата 500 ₽ карта *1234 Т-Банк')).toBe('tinkoff');
		expect(inferBankIdFromText('Samsung Pay. Сбербанк *4321')).toBe('sberbank');
		expect(inferBankIdFromText('Оплата 990 ₽ OZON')).toBeNull();
		expect(inferBankIdFromText('Покупка WB')).toBeNull();
	});
});

describe('SMS senders', () => {
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
		expect(normalizeSmsSender('Ваш Т-Банк')).toBe('tbank');
		expect(allKnownSmsSenderEntries().some((e) => e.sender === '900')).toBe(true);
	});

	it('resolves Google Messages notification title to T-Bank', () => {
		const resolved = resolveRawBankNotification({
			packageName: 'com.google.android.apps.messaging',
			title: 'Ваш Т-Банк',
			text: 'Пополнение. Счет RUB. 3535,96 ₽. Др. банк. Доступно 3535,96 ₽',
			bigText: '',
			postedAt: 1,
			dedupeKey: 'k'
		});
		expect(resolved.packageName).toBe('com.idamob.tinkoff.android');
		expect(resolved.channel).toBe('sms');
	});
});
