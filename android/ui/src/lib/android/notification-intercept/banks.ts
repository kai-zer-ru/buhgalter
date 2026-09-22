import type { RawBankNotification } from './types';

/** SMS apps whose notifications mirror bank SMS (not bank push). */
export const MESSAGING_APP_PACKAGES = [
	'com.google.android.apps.messaging',
	'com.samsung.android.messaging',
	'com.android.mms'
] as const;

/** Known RF bank apps for notification intercept (ids match `banks_ru.json`). */
export type KnownBankApp = {
	bankId: string;
	/** Primary retail app package (+ aliases: push sometimes comes from the marketplace app). */
	packageNames: string[];
	/**
	 * SMS originators (short codes / alphanumeric). Empty → SMS for this bank is not captured.
	 * Keep in sync with Java {@code DEFAULT_SMS_SENDERS_JSON}.
	 */
	smsSenders: string[];
	/** i18n key for display name fallback when catalog is unavailable */
	labelKey: string;
};

/**
 * Package names of retail Android apps. Texts differ per bank; purchase
 * heuristics in parsers.ts are shared. Wrong/outdated package → no capture
 * for that bank (safe); update here when a bank renames the app.
 */
export const KNOWN_BANK_APPS: KnownBankApp[] = [
	{
		bankId: 'sberbank',
		packageNames: ['ru.sberbankmobile'],
		smsSenders: ['900'],
		labelKey: 'bankNotifications.bank.sberbank'
	},
	{
		bankId: 'tinkoff',
		packageNames: ['com.idamob.tinkoff.android'],
		smsSenders: ['T-Bank', 'TBank', 'Tinkoff', '7555'],
		labelKey: 'bankNotifications.bank.tinkoff'
	},
	{
		bankId: 'vtb',
		packageNames: ['ru.vtb24.mobilebanking.android'],
		smsSenders: ['VTB', '1000'],
		labelKey: 'bankNotifications.bank.vtb'
	},
	{
		bankId: 'alfabank',
		packageNames: ['ru.alfabank.mobile.android'],
		smsSenders: ['Alfabank', 'Alfa-Bank', '2265'],
		labelKey: 'bankNotifications.bank.alfabank'
	},
	{
		bankId: 'gazprombank',
		packageNames: ['ru.gazprombank.android.mobilebank.app'],
		smsSenders: ['Gazprombank'],
		labelKey: 'bankNotifications.bank.gazprombank'
	},
	{
		bankId: 'raiffeisen',
		packageNames: ['ru.raiffeisennews'],
		smsSenders: ['Raiffeisen'],
		labelKey: 'bankNotifications.bank.raiffeisen'
	},
	{
		bankId: 'rosbank',
		packageNames: ['ru.rosbank.android'],
		smsSenders: ['Rosbank'],
		labelKey: 'bankNotifications.bank.rosbank'
	},
	{
		bankId: 'mkb',
		packageNames: ['ru.mkb.mobile'],
		smsSenders: ['MKB'],
		labelKey: 'bankNotifications.bank.mkb'
	},
	{
		bankId: 'rshb',
		packageNames: ['ru.rshb.dbo'],
		smsSenders: ['RSHB'],
		labelKey: 'bankNotifications.bank.rshb'
	},
	{
		bankId: 'open',
		packageNames: ['com.openbank'],
		smsSenders: ['Otkritie', 'Open'],
		labelKey: 'bankNotifications.bank.open'
	},
	{
		bankId: 'sovcombank',
		packageNames: ['ru.sovcomcard.halva.v1', 'ru.sovcombank.mobile'],
		smsSenders: ['Sovcombank', 'Halva'],
		labelKey: 'bankNotifications.bank.sovcombank'
	},
	{
		bankId: 'psb',
		packageNames: ['ru.ftc.faktura.psb', 'logo.com.mbanking'],
		smsSenders: ['PSB'],
		labelKey: 'bankNotifications.bank.psb'
	},
	{
		bankId: 'uralsib',
		packageNames: ['ru.uralsib.mb'],
		smsSenders: ['Uralsib'],
		labelKey: 'bankNotifications.bank.uralsib'
	},
	{
		bankId: 'homecredit',
		packageNames: ['ru.homecredit.mycredit'],
		smsSenders: ['HomeCredit'],
		labelKey: 'bankNotifications.bank.homecredit'
	},
	{
		bankId: 'ozon',
		packageNames: ['ru.ozon.fintech.finance', 'ru.ozon.app.android'],
		smsSenders: ['Ozon'],
		labelKey: 'bankNotifications.bank.ozon'
	},
	{
		bankId: 'yandex',
		packageNames: ['com.yandex.bank'],
		smsSenders: ['Yandex'],
		labelKey: 'bankNotifications.bank.yandex'
	},
	{
		bankId: 'wbbank',
		packageNames: ['ru.wildberries.fintech', 'com.wildberries.ru'],
		smsSenders: ['WB', 'WBBank'],
		labelKey: 'bankNotifications.bank.wbbank'
	},
	{
		bankId: 'otpbank',
		packageNames: ['ru.otpbank.mobile'],
		smsSenders: ['OTPBank'],
		labelKey: 'bankNotifications.bank.otpbank'
	},
	{
		bankId: 'atb',
		packageNames: ['ru.atb.mobilbank'],
		smsSenders: ['ATB'],
		labelKey: 'bankNotifications.bank.atb'
	}
];

/** NFC / in-app wallets that post their own purchase push (not a bank app). */
export type KnownWalletApp = {
	walletId: string;
	packageNames: string[];
	labelKey: string;
};

/**
 * RF-relevant tap-to-pay / wallet apps. Keep packages in sync with Java
 * {@code DEFAULT_BANK_PACKAGES}. Not shown in «счёт → банк» — resolve via last4
 * or a bank name in the notification text.
 */
export const KNOWN_WALLET_APPS: KnownWalletApp[] = [
	{
		walletId: 'mir_pay',
		packageNames: ['ru.nspk.mirpay'],
		labelKey: 'bankNotifications.wallet.mir_pay'
	},
	{
		walletId: 'sbp_pay',
		packageNames: ['ru.nspk.sbpay'],
		labelKey: 'bankNotifications.wallet.sbp_pay'
	},
	{
		walletId: 'samsung_pay',
		packageNames: ['com.samsung.android.spay', 'com.samsung.android.spaymini'],
		labelKey: 'bankNotifications.wallet.samsung_pay'
	},
	{
		walletId: 'google_wallet',
		packageNames: ['com.google.android.apps.walletnfcrelay'],
		labelKey: 'bankNotifications.wallet.google_wallet'
	},
	{
		walletId: 'huawei_wallet',
		packageNames: ['com.huawei.wallet'],
		labelKey: 'bankNotifications.wallet.huawei_wallet'
	},
	{
		walletId: 'xiaomi_wallet',
		packageNames: ['com.mipay.wallet', 'com.xiaomi.payment'],
		labelKey: 'bankNotifications.wallet.xiaomi_wallet'
	},
	{
		walletId: 'yoomoney',
		packageNames: ['ru.yandex.money'],
		labelKey: 'bankNotifications.wallet.yoomoney'
	}
];

/** Bank names as card issuer in wallet push — not marketplace brands (Ozon, WB). */
const BANK_TEXT_ALIASES: { bankId: string; re: RegExp }[] = [
	{ bankId: 'tinkoff', re: /т-?банк|тинькофф|t-?bank|tinkoff/i },
	{ bankId: 'sberbank', re: /сбер(?:банк|пей|\s*pay)?|\bsber(?:bank|pay)?\b/i },
	{ bankId: 'alfabank', re: /альфа[-\s]?банк|alfa[-\s]?bank|\balfabank\b/i },
	{ bankId: 'gazprombank', re: /газпромбанк|gazprombank/i },
	{ bankId: 'raiffeisen', re: /райффайзен|raiffeisen/i },
	{ bankId: 'rosbank', re: /росбанк|rosbank/i },
	{ bankId: 'rshb', re: /россельхоз(?:банк)?|\brshb\b/i },
	{ bankId: 'open', re: /открытие|otkritie/i },
	{ bankId: 'sovcombank', re: /совкомбанк|sovcombank|\bхалва\b|\bhalva\b/i },
	{ bankId: 'uralsib', re: /уралсиб|uralsib/i },
	{ bankId: 'homecredit', re: /хоум\s*кредит|home\s*credit/i },
	{ bankId: 'vtb', re: /\bвтб\b|\bvtb\b/i },
	{ bankId: 'mkb', re: /\bмкб\b|\bmkb\b/i },
	{ bankId: 'psb', re: /\bпсб\b|\bpsb\b|промсвязьбанк/i },
	{ bankId: 'otpbank', re: /отп\s*банк|otp\s*bank/i },
	{ bankId: 'atb', re: /\bатб\b/i },
	{ bankId: 'ozon', re: /озон\s*банк|ozon\s*bank/i },
	{ bankId: 'yandex', re: /яндекс\s*банк|yandex\s*bank/i },
	{ bankId: 'wbbank', re: /wb\s*банк|wildberries\s*банк/i }
];

/** Normalize SMS originator the same way as native {@code NotificationInterceptStore}. */
export function normalizeSmsSender(raw: string): string {
	let s = raw.trim().toLowerCase();
	s = s.replace(/^(ваш|your)\s+/i, '');
	s = s.replace(/т-?банк/g, 'tbank');
	s = s.replace(/тинькофф/g, 'tinkoff');
	s = s.replace(/[\s-]/g, '');
	if (s.startsWith('+7') && s.length > 2) {
		s = s.slice(2);
	} else if (s.startsWith('8') && s.length === 11 && /^\d+$/.test(s)) {
		s = s.slice(1);
	}
	return s;
}

export function isMessagingAppPackage(packageName: string): boolean {
	return (MESSAGING_APP_PACKAGES as readonly string[]).includes(packageName);
}

/**
 * Google Messages / Samsung Messages show bank SMS as their own notification.
 * Remap to the bank package + sms channel so parsers and allowlist work.
 */
export function resolveRawBankNotification(raw: RawBankNotification): RawBankNotification {
	if (raw.channel === 'sms' && bankIdForPackage(raw.packageName)) {
		return raw;
	}
	if (!isMessagingAppPackage(raw.packageName)) {
		return raw;
	}
	const bankId =
		bankIdForSmsSender(raw.title) ?? bankIdForSmsSender(raw.text.split(/[\n.]/)[0]?.trim() ?? '');
	if (!bankId) return raw;
	const pkg = packageForBankId(bankId);
	if (!pkg) return raw;
	return {
		...raw,
		packageName: pkg,
		channel: 'sms'
	};
}

export function bankIdForPackage(packageName: string): string | null {
	const hit = KNOWN_BANK_APPS.find((b) => b.packageNames.includes(packageName));
	return hit?.bankId ?? null;
}

export function walletIdForPackage(packageName: string): string | null {
	const hit = KNOWN_WALLET_APPS.find((w) => w.packageNames.includes(packageName));
	return hit?.walletId ?? null;
}

export function isWalletPackage(packageName: string): boolean {
	return walletIdForPackage(packageName) != null;
}

export function isWalletId(id: string): boolean {
	return KNOWN_WALLET_APPS.some((w) => w.walletId === id);
}

/** Bank app or wallet — native allowlist / history. */
export function isAllowlistedPackage(packageName: string): boolean {
	return bankIdForPackage(packageName) != null || walletIdForPackage(packageName) != null;
}

/** Issuer bank mentioned in wallet (or other) notification text. */
export function inferBankIdFromText(text: string): string | null {
	if (!text.trim()) return null;
	for (const { bankId, re } of BANK_TEXT_ALIASES) {
		if (re.test(text)) return bankId;
	}
	return null;
}

export function bankIdForSmsSender(sender: string): string | null {
	const key = normalizeSmsSender(sender);
	if (!key) return null;
	for (const b of KNOWN_BANK_APPS) {
		for (const s of b.smsSenders) {
			if (normalizeSmsSender(s) === key) return b.bankId;
		}
	}
	return null;
}

/** Primary package for settings storage / display. */
export function packageForBankId(bankId: string): string | null {
	const hit = KNOWN_BANK_APPS.find((b) => b.bankId === bankId);
	return hit?.packageNames[0] ?? null;
}

export function allKnownPackages(): string[] {
	const out: string[] = [];
	for (const b of KNOWN_BANK_APPS) {
		for (const pkg of b.packageNames) {
			if (!out.includes(pkg)) out.push(pkg);
		}
	}
	for (const w of KNOWN_WALLET_APPS) {
		for (const pkg of w.packageNames) {
			if (!out.includes(pkg)) out.push(pkg);
		}
	}
	return out;
}

/** Flat list for native sync: sender → primary package. */
export function allKnownSmsSenderEntries(): { sender: string; packageName: string }[] {
	const out: { sender: string; packageName: string }[] = [];
	const seen = new Set<string>();
	for (const b of KNOWN_BANK_APPS) {
		const pkg = b.packageNames[0];
		if (!pkg) continue;
		for (const raw of b.smsSenders) {
			const sender = normalizeSmsSender(raw);
			if (!sender || seen.has(sender)) continue;
			seen.add(sender);
			out.push({ sender, packageName: pkg });
		}
	}
	return out;
}

export function allKnownSmsSenders(): string[] {
	return allKnownSmsSenderEntries().map((e) => e.sender);
}
