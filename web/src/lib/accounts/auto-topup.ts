import type { Account } from '$lib/api/client';

export function isAutoTopupEligible(acc: Pick<Account, 'type' | 'status'>): boolean {
	return acc.type === 'bank' && acc.status === 'active';
}

export function autoTopupSourceOptions(accounts: Account[], beneficiaryId: string) {
	return accounts
		.filter((a) => a.status === 'active' && a.type === 'bank' && a.id !== beneficiaryId)
		.map((a) => ({ value: a.id, label: a.name }));
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
