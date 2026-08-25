export type EntityKind =
	| 'transaction'
	| 'transfer'
	| 'category'
	| 'debt'
	| 'account'
	| 'budget'
	| 'credit'
	| 'recurring'
	| 'subscription';

export type PendingOp = 'create' | 'update' | 'delete';

export type TransactionPayload = {
	account_id: string;
	type: 'income' | 'expense';
	amount: string;
	description?: string;
	category_id?: string;
	subcategory_id?: string;
	subcategory_name?: string;
	merchant_id?: string;
	merchant_name?: string;
	tag_ids?: string[];
	tag_names?: string[];
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

export type DebtSettlePayload = {
	action: 'settle';
	amount?: string;
	settled_at: string;
	affects_balance: boolean;
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

export type CreditMetaUpdatePayload = {
	action: 'update';
	credit_id: string;
	name?: string | null;
	debit_account_id?: string;
	debit_time_local?: string | null;
	bank_id?: string | null;
};

export type CreditPayPayload = {
	action: 'pay';
	credit_id: string;
	amount: string;
	payment_date: string;
	account_id?: string;
};

export type CreditCompletePayload = {
	action: 'complete';
	credit_id: string;
	affects_balance: boolean;
	payment_date: string;
};

export type CreditSchedulePayload = {
	action: 'schedule';
	credit_id: string;
	payments: { id: string; amount: string }[];
};

export type CreditDeletePaymentPayload = {
	action: 'delete_payment';
	credit_id: string;
	payment_id: string;
};

export type CreditDeletePayload = {
	action: 'delete';
	credit_id: string;
	mode: 'cascade' | 'keep_transactions';
};

export type CreditActionPayload =
	| CreditMetaUpdatePayload
	| CreditPayPayload
	| CreditCompletePayload
	| CreditSchedulePayload
	| CreditDeletePaymentPayload
	| CreditDeletePayload;

export type RecurringPayload = {
	type: 'income' | 'expense';
	amount: string;
	description?: string;
	account_id: string;
	category_id: string;
	subcategory_id?: string;
	period: 'week' | 'two_weeks' | 'month' | 'year';
	weekday?: number;
	day_of_month?: number;
	start_date: string;
	time_local?: string;
	active?: boolean;
};

export type SubscriptionPayload = {
	name: string;
	amount: string;
	description?: string;
	icon?: string;
	website_url?: string;
	account_id: string;
	period: 'week' | 'two_weeks' | 'month' | 'quarter' | 'half_year' | 'year';
	weekday?: number;
	day_of_month?: number;
	start_date: string;
	time_local?: string;
	active?: boolean;
	upcoming_run_ats?: string[];
	/** Online-only; stripped from offline outbox creates. */
	attach_transaction_id?: string;
};

export type OutboxPayload =
	| TransactionPayload
	| TransferPayload
	| CategoryPayload
	| CategoryUpdatePayload
	| DebtPayload
	| DebtSettlePayload
	| AccountCreatePayload
	| AccountUpdatePayload
	| AccountStatusPayload
	| BudgetPayload
	| CreditActionPayload
	| RecurringPayload
	| SubscriptionPayload;

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

export function isDebtSettlePayload(p: unknown): p is DebtSettlePayload {
	return !!p && typeof p === 'object' && (p as DebtSettlePayload).action === 'settle';
}

export function isCreditActionPayload(p: unknown): p is CreditActionPayload {
	if (!p || typeof p !== 'object' || !('action' in p) || !('credit_id' in p)) return false;
	const action = (p as CreditActionPayload).action;
	return (
		action === 'update' ||
		action === 'pay' ||
		action === 'complete' ||
		action === 'schedule' ||
		action === 'delete_payment' ||
		action === 'delete'
	);
}

export function creditMetaEntityKey(creditId: string): string {
	return `credit:${creditId}`;
}

export function creditPayEntityKey(creditId: string, payId: string): string {
	return `credit:${creditId}:pay:${payId}`;
}

export function creditCompleteEntityKey(creditId: string): string {
	return `credit:${creditId}:complete`;
}

export function creditScheduleEntityKey(creditId: string): string {
	return `credit:${creditId}:schedule`;
}

export function creditDeletePaymentEntityKey(creditId: string, paymentId: string): string {
	return `credit:${creditId}:delpay:${paymentId}`;
}

export function creditDeleteEntityKey(creditId: string): string {
	return `credit:${creditId}:delete`;
}
