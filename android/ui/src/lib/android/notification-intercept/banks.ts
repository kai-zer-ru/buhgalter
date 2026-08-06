/** Known RF bank apps for notification intercept (ids match `banks_ru.json`). */
export type KnownBankApp = {
	bankId: string;
	/** Primary retail app package (+ aliases: push sometimes comes from the marketplace app). */
	packageNames: string[];
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
		labelKey: 'bankNotifications.bank.sberbank'
	},
	{
		bankId: 'tinkoff',
		packageNames: ['com.idamob.tinkoff.android'],
		labelKey: 'bankNotifications.bank.tinkoff'
	},
	{
		bankId: 'vtb',
		packageNames: ['ru.vtb24.mobilebanking.android'],
		labelKey: 'bankNotifications.bank.vtb'
	},
	{
		bankId: 'alfabank',
		packageNames: ['ru.alfabank.mobile.android'],
		labelKey: 'bankNotifications.bank.alfabank'
	},
	{
		bankId: 'gazprombank',
		packageNames: ['ru.gazprombank.android.mobilebank.app'],
		labelKey: 'bankNotifications.bank.gazprombank'
	},
	{
		bankId: 'raiffeisen',
		packageNames: ['ru.raiffeisennews'],
		labelKey: 'bankNotifications.bank.raiffeisen'
	},
	{
		bankId: 'rosbank',
		packageNames: ['ru.rosbank.android'],
		labelKey: 'bankNotifications.bank.rosbank'
	},
	{
		bankId: 'mkb',
		packageNames: ['ru.mkb.mobile'],
		labelKey: 'bankNotifications.bank.mkb'
	},
	{
		bankId: 'rshb',
		packageNames: ['ru.rshb.dbo'],
		labelKey: 'bankNotifications.bank.rshb'
	},
	{
		bankId: 'open',
		packageNames: ['com.openbank'],
		labelKey: 'bankNotifications.bank.open'
	},
	{
		bankId: 'sovcombank',
		packageNames: ['ru.sovcomcard.halva.v1', 'ru.sovcombank.mobile'],
		labelKey: 'bankNotifications.bank.sovcombank'
	},
	{
		bankId: 'psb',
		packageNames: ['ru.ftc.faktura.psb', 'logo.com.mbanking'],
		labelKey: 'bankNotifications.bank.psb'
	},
	{
		bankId: 'uralsib',
		packageNames: ['ru.uralsib.mb'],
		labelKey: 'bankNotifications.bank.uralsib'
	},
	{
		bankId: 'homecredit',
		packageNames: ['ru.homecredit.mycredit'],
		labelKey: 'bankNotifications.bank.homecredit'
	},
	{
		bankId: 'ozon',
		packageNames: ['ru.ozon.fintech.finance', 'ru.ozon.app.android'],
		labelKey: 'bankNotifications.bank.ozon'
	},
	{
		bankId: 'yandex',
		packageNames: ['com.yandex.bank'],
		labelKey: 'bankNotifications.bank.yandex'
	},
	{
		bankId: 'wbbank',
		packageNames: ['ru.wildberries.fintech', 'com.wildberries.ru'],
		labelKey: 'bankNotifications.bank.wbbank'
	},
	{
		bankId: 'otpbank',
		packageNames: ['ru.otpbank.mobile'],
		labelKey: 'bankNotifications.bank.otpbank'
	},
	{
		bankId: 'atb',
		packageNames: ['ru.atb.mobilbank'],
		labelKey: 'bankNotifications.bank.atb'
	}
];

export function bankIdForPackage(packageName: string): string | null {
	const hit = KNOWN_BANK_APPS.find((b) => b.packageNames.includes(packageName));
	return hit?.bankId ?? null;
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
	return out;
}
