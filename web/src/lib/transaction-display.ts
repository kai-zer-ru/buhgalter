import type { Transaction } from '$lib/api/client';

/** Out leg of a transfer (first by created_at). */
export function transferOutLeg(tx: Transaction, siblings: Transaction[]): Transaction {
	if (!tx.transfer_group_id) return tx;
	const legs = siblings.filter((t) => t.transfer_group_id === tx.transfer_group_id);
	if (legs.length === 0) return tx;
	return legs.reduce((best, cur) => (cur.created_at < best.created_at ? cur : best));
}

/** Hide in-leg rows when both legs appear in the same list (e.g. dashboard). */
export function dedupeTransferLegs(transactions: Transaction[]): Transaction[] {
	const groups = new Map<string, Transaction[]>();
	for (const tx of transactions) {
		if (tx.type !== 'transfer' || !tx.transfer_group_id) continue;
		const list = groups.get(tx.transfer_group_id) ?? [];
		list.push(tx);
		groups.set(tx.transfer_group_id, list);
	}
	const outIds = new Set<string>();
	for (const legs of groups.values()) {
		outIds.add(transferOutLeg(legs[0], legs).id);
	}
	return transactions.filter((tx) => {
		if (tx.type !== 'transfer' || !tx.transfer_group_id) return true;
		return outIds.has(tx.id);
	});
}

export function transferRoute(
	tx: Transaction,
	siblings: Transaction[] = []
): { from: string; to: string } {
	if (tx.type !== 'transfer') {
		return { from: tx.account_name ?? '', to: '' };
	}

	const legs = siblings.filter(
		(t) => t.transfer_group_id && t.transfer_group_id === tx.transfer_group_id
	);

	if (legs.length >= 2) {
		const out = transferOutLeg(tx, legs);
		return {
			from: out.account_name ?? '',
			to: out.transfer_account_name ?? ''
		};
	}

	// One leg in list (e.g. account filter): direction from API, not from local min(created_at)
	if (tx.transfer_is_out) {
		return {
			from: tx.account_name ?? '',
			to: tx.transfer_account_name ?? ''
		};
	}
	return {
		from: tx.transfer_account_name ?? '',
		to: tx.account_name ?? ''
	};
}

export type AccountLabelMode = 'plain' | 'prefix';

/** Account column text: transfer route, expense (from), income (to). */
export function formatTransactionAccount(
	tx: Transaction,
	siblings: Transaction[] = [],
	mode: AccountLabelMode = 'plain'
): string {
	if (tx.type === 'transfer') {
		const { from, to } = transferRoute(tx, siblings);
		if (from && to) return `${from} → ${to}`;
		return from || to;
	}
	const name = tx.account_name ?? '';
	if (!name) return '';
	if (mode === 'plain') return name;
	if (tx.type === 'expense') return `с ${name}`;
	return `на ${name}`;
}

/** Prefix before amount: +/− for income/expense; on single-account lists transfers use +/− by leg. */
export function transactionAmountSign(tx: Transaction, opts?: { singleAccount?: boolean }): string {
	if (tx.type === 'income') return '+';
	if (tx.type === 'expense') return '−';
	if (tx.type === 'transfer' && opts?.singleAccount) {
		return tx.transfer_is_out ? '−' : '+';
	}
	return '';
}
