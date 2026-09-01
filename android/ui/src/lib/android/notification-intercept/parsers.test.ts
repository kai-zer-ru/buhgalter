import { describe, expect, it } from 'vitest';
import { parseBankNotification } from './parsers';
import type { RawBankNotification } from './types';

function raw(partial: Partial<RawBankNotification> & { packageName: string }): RawBankNotification {
	return {
		title: '',
		text: '',
		bigText: '',
		postedAt: 1_700_000_000_000,
		dedupeKey: 'k1',
		...partial
	};
}

describe('parseBankNotification', () => {
	it('parses T-Bank purchase with last4 and merchant', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.idamob.tinkoff.android',
				title: 'Покупка',
				text: 'Покупка 1 234,50 ₽. Пятёрочка. Карта *4321'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.bankId).toBe('tinkoff');
		expect(parsed!.amount).toBe('1234.50');
		expect(parsed!.last4).toBe('4321');
		expect(parsed!.merchantText.toLowerCase()).toContain('пят');
	});

	it('parses Sber purchase', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'ru.sberbankmobile',
				text: 'Покупка 99.00₽ MAGNIT Карта *1111'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.bankId).toBe('sberbank');
		expect(parsed!.amount).toBe('99.00');
		expect(parsed!.last4).toBe('1111');
		expect(parsed!.merchantText.toUpperCase()).toContain('MAGNIT');
	});

	it('ignores OTP / codes', () => {
		expect(
			parseBankNotification(
				raw({
					packageName: 'com.idamob.tinkoff.android',
					text: 'Код 1234 для входа в приложение'
				})
			)
		).toBeNull();
	});

	it('ignores balance updates', () => {
		expect(
			parseBankNotification(
				raw({
					packageName: 'ru.sberbankmobile',
					text: 'Баланс карты *1234: 10 000 ₽'
				})
			)
		).toBeNull();
	});

	it('parses interest payout as income (not expense)', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.wildberries.ru',
				title: 'WB Банк',
				text: 'Выплата процентов 5,36 ₽.\nДоступно 29 323,51 ₽'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.kind).toBe('income');
		expect(parsed!.amount).toBe('5.36');
		expect(parsed!.bankId).toBe('wbbank');
		expect(parsed!.merchantText.toLowerCase()).toContain('процент');
	});

	it('parses top-up / incoming transfer as income', () => {
		const topup = parseBankNotification(
			raw({
				packageName: 'com.idamob.tinkoff.android',
				text: 'Пополнение на 5 000 ₽. Доступно 12 000 ₽'
			})
		);
		expect(topup).not.toBeNull();
		expect(topup!.kind).toBe('income');
		expect(topup!.amount).toBe('5000.00');

		const transfer = parseBankNotification(
			raw({
				packageName: 'ru.sberbankmobile',
				text: 'Перевод от Иван И. 1 500 ₽. Баланс 10 000 ₽'
			})
		);
		expect(transfer).not.toBeNull();
		expect(transfer!.kind).toBe('income');
		expect(transfer!.amount).toBe('1500.00');
	});

	it('ignores unknown packages', () => {
		expect(
			parseBankNotification(
				raw({
					packageName: 'com.example.other',
					text: 'Покупка 100 ₽ Shop'
				})
			)
		).toBeNull();
	});

	it('parses Yandex Pay purchase with available balance line', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.yandex.bank',
				text: 'Покупка на 802.00 RUB, карта *3349. Доступно 5297.64 RUB'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.bankId).toBe('yandex');
		expect(parsed!.amount).toBe('802.00');
		expect(parsed!.last4).toBe('3349');
	});

	it('parses Yandex purchase 4006 with available balance (no merchant title)', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.yandex.bank',
				text: 'Покупка на 4006.00 RUB, карта *3349. Доступно 828.70 RUB'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.amount).toBe('4006.00');
		expect(parsed!.last4).toBe('3349');
		expect(parsed!.kind).toBe('purchase');
	});

	it('parses Yandex cancel and takes merchant from body when title is Карта Пэй', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.yandex.bank',
				title: 'Карта Пэй',
				text: 'Отмена покупки на 155.00 RUB Поехали!. Доступно 5000.00 RUB'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.kind).toBe('cancel');
		expect(parsed!.amount).toBe('155.00');
		expect(parsed!.merchantText.toLowerCase()).toContain('поехали');
	});

	it('parses matching Yandex taxi purchase for cancel pair', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.yandex.bank',
				title: 'Поехали!',
				text: 'Покупка на 155.00 RUB, карта *3349. Доступно 4845.00 RUB'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.kind).toBe('purchase');
		expect(parsed!.amount).toBe('155.00');
		expect(parsed!.last4).toBe('3349');
		expect(parsed!.merchantText).toBe('Поехали!');
	});

	it('uses Yandex title as merchant name', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.yandex.bank',
				title: 'Народный',
				text: 'Покупка на 276.96 RUB, карта *6517. Доступно 5020.68 RUB'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.amount).toBe('276.96');
		expect(parsed!.last4).toBe('6517');
		expect(parsed!.merchantText).toBe('Народный');
	});

	it('parses Sber SMS purchase (sender as title)', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'ru.sberbankmobile',
				title: '900',
				text: 'Покупка 450.00р MAGNIT Карта *1234 Баланс: 12 340.50р',
				channel: 'sms',
				dedupeKey: 'sms|900|1|abc'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.bankId).toBe('sberbank');
		expect(parsed!.kind).toBe('purchase');
		expect(parsed!.amount).toBe('450.00');
		expect(parsed!.last4).toBe('1234');
		expect(parsed!.merchantText.toUpperCase()).toContain('MAGNIT');
	});

	it('parses T-Bank SMS purchase', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.idamob.tinkoff.android',
				title: 'T-Bank',
				text: 'Покупка 89,90 RUB, карта *4321. Пятёрочка. Доступно 5 000 RUB',
				channel: 'sms'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.bankId).toBe('tinkoff');
		expect(parsed!.amount).toBe('89.90');
		expect(parsed!.last4).toBe('4321');
		expect(parsed!.merchantText.toLowerCase()).toContain('пят');
	});

	it('parses T-Bank SMS purchase with dot-separated fields', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.idamob.tinkoff.android',
				title: 'T-Bank',
				text: 'Покупка, карта *2552. 56 RUB. STOLOVAYA. Доступно 65,3 RUB',
				channel: 'sms'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.bankId).toBe('tinkoff');
		expect(parsed!.kind).toBe('purchase');
		expect(parsed!.amount).toBe('56.00');
		expect(parsed!.last4).toBe('2552');
		expect(parsed!.merchantText).toBe('STOLOVAYA');
	});

	it('parses T-Bank SMS top-up as income', () => {
		const parsed = parseBankNotification(
			raw({
				packageName: 'com.google.android.apps.messaging',
				title: 'Ваш Т-Банк',
				text: 'Пополнение. Счет RUB. 3535,96 ₽. Др. банк. Доступно 3535,96 ₽',
				channel: 'push'
			})
		);
		expect(parsed).not.toBeNull();
		expect(parsed!.bankId).toBe('tinkoff');
		expect(parsed!.kind).toBe('income');
		expect(parsed!.amount).toBe('3535.96');
	});

	it('ignores OTP SMS', () => {
		expect(
			parseBankNotification(
				raw({
					packageName: 'ru.sberbankmobile',
					title: '900',
					text: 'Никому не сообщайте код: 12345. Вход в СберБанк Онлайн.',
					channel: 'sms'
				})
			)
		).toBeNull();
	});
});
