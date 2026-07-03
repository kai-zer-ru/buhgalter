import type { Account } from '$lib/api/client';
import { accountSelectOption } from '$lib/select-options';

type AutoTopupStatusAccount = Pick<Account, 'auto_topup_enabled' | 'auto_topup_source_account_id'>;
type AccountNameLookup = Pick<Account, 'id' | 'name'>;

export function resolveAutoTopupSourceName(
	acc: AutoTopupStatusAccount,
	accounts: AccountNameLookup[]
): string | null {
	if (!acc.auto_topup_enabled || !acc.auto_topup_source_account_id) {
		return null;
	}
	return accounts.find((a) => a.id === acc.auto_topup_source_account_id)?.name ?? null;
}

export function isAutoTopupEligible(acc: Pick<Account, 'type' | 'status'>): boolean {
	return acc.type === 'bank' && acc.status === 'active';
}

export function autoTopupSourceOptions(accounts: Account[], beneficiaryId: string) {
	return accounts
		.filter((a) => a.status === 'active' && a.type === 'bank' && a.id !== beneficiaryId)
		.map((a) => accountSelectOption(a));
}

export function defaultAutoTopupSourceId(accounts: Account[], beneficiaryId: string): string {
	const banks = accounts.filter(
		(a) => a.status === 'active' && a.type === 'bank' && a.id !== beneficiaryId
	);
	const primary = banks.find((a) => a.is_primary);
	return primary?.id ?? banks[0]?.id ?? '';
}

export function validateAutoTopupForm(
	enabled: boolean,
	threshold: string,
	target: string,
	sourceId: string
): string | null {
	if (!enabled) return null;
	if (!threshold.trim() || !target.trim() || !sourceId) {
		return 'required';
	}
	const thresholdNum = Number(threshold.replace(/\s/g, '').replace(',', '.'));
	const targetNum = Number(target.replace(/\s/g, '').replace(',', '.'));
	if (!Number.isFinite(thresholdNum) || !Number.isFinite(targetNum)) {
		return 'invalid';
	}
	if (thresholdNum < 0 || targetNum <= 0 || thresholdNum >= targetNum) {
		return 'range';
	}
	return null;
}
