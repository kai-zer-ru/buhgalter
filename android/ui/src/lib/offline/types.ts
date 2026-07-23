export type EntityKind = 'transaction' | 'transfer' | 'category' | 'debt' | 'account' | 'budget';

export type PendingOp = 'create' | 'update' | 'delete';

export type TransactionPayload = {
	account_id: string;
	type: 'income' | 'expense';
	amount: string;
	description?: string;
	category_id?: string;
	subcategory_id?: string;
	subcategory_name?: string;
	transaction_date: string;
};

export type TransferPayload = {
	from_account_id: string;
	to_account_id: string;
	amount: string;
	commission?: string;
	description?: string;
	transaction_date: string;
};

export type CategoryPayload = {
	name: string;
	type: 'income' | 'expense';
	icon: string;
	sort_order?: number;
};

export type CategoryUpdatePayload = {
	name: string;
	icon: string;
	sort_order?: number;
};

export type DebtPayload = {
	debtor_id?: string;
	debtor_name?: string;
	direction: 'lent' | 'borrowed';
	amount: string;
	debt_date: string;
	due_date: string;
	affects_balance: boolean;
	description?: string;
	account_id?: string;
};

export type AccountCreatePayload = {
	name: string;
	type: 'cash' | 'bank' | 'credit_card';
	bank_id?: string;
	initial_balance: string;
	credit_limit?: string;
	payment_account_id?: string;
};

export type AccountUpdatePayload = {
	name: string;
	bank_id?: string;
	initial_balance?: string;
	credit_limit?: string;
	payment_account_id?: string | null;
	auto_topup_enabled?: boolean;
	auto_topup_threshold?: string;
	auto_topup_target?: string;
	auto_topup_source_account_id?: string;
};

/** Archive / unarchive encoded as update payload for replay. */
export type AccountStatusPayload =
	| { action: 'archive'; transfer_to_account_id?: string }
	| { action: 'unarchive' };

export type BudgetPayload = {
	name: string;
	scope: 'category' | 'subcategory' | 'all_expense';
	category_id?: string;
	subcategory_id?: string;
	account_id?: string;
	amount: string;
	alert_at_percent?: number;
	is_active?: boolean;
	copy_forward?: boolean;
	/** Month query for create/update (`YYYY-MM`). */
	month?: string;
};

export type OutboxPayload =
	| TransactionPayload
	| TransferPayload
	| CategoryPayload
	| CategoryUpdatePayload
	| DebtPayload
	| AccountCreatePayload
	| AccountUpdatePayload
	| AccountStatusPayload
	| BudgetPayload;

export type OutboxEntry = {
	entityKey: string;
	kind: EntityKind;
	op: PendingOp;
	isLocalOnly: boolean;
	payload?: OutboxPayload;
	seq: number;
	failed?: { message: string };
};

export type OutboxSnapshot = {
	entries: OutboxEntry[];
	nextSeq: number;
};

export const LOCAL_KEY_PREFIX = 'local:';

export function isLocalEntityKey(key: string): boolean {
	return key.startsWith(LOCAL_KEY_PREFIX);
}

export function makeLocalKey(): string {
	const id =
		typeof crypto !== 'undefined' && crypto.randomUUID
			? crypto.randomUUID()
			: `${Date.now()}-${Math.random().toString(36).slice(2)}`;
	return `${LOCAL_KEY_PREFIX}${id}`;
}

export function isAccountStatusPayload(p: unknown): p is AccountStatusPayload {
	return (
		!!p &&
		typeof p === 'object' &&
		'action' in p &&
		((p as AccountStatusPayload).action === 'archive' ||
			(p as AccountStatusPayload).action === 'unarchive')
	);
}
