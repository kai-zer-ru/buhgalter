import { isWalletId, isWalletPackage } from './banks';
import type { InterceptSettings, ParsedPurchase } from './types';

function isWalletSource(parsed: ParsedPurchase): boolean {
	return isWalletId(parsed.bankId) || isWalletPackage(parsed.packageName);
}

/** last4 binding wins; else first bank/package default account (many accounts may share a bank). */
export function resolveAccountId(
	parsed: ParsedPurchase,
	settings: InterceptSettings
): string | undefined {
	if (parsed.last4) {
		const card = settings.cardBindings.find(
			(c) => c.bankId === parsed.bankId && c.last4 === parsed.last4 && c.accountId
		);
		if (card?.accountId) return card.accountId;
		// Wallet push often has last4 but bankId is mir_pay / samsung_pay.
		if (isWalletSource(parsed)) {
			const hits = settings.cardBindings.filter((c) => c.last4 === parsed.last4 && c.accountId);
			if (hits.length === 1) return hits[0].accountId;
		}
	}
	const bank = settings.bankBindings.find(
		(b) => b.accountId && (b.bankId === parsed.bankId || b.packageName === parsed.packageName)
	);
	return bank?.accountId || undefined;
}
