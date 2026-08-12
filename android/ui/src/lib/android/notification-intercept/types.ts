export type RawBankNotification = {
	packageName: string;
	title: string;
	text: string;
	bigText: string;
	postedAt: number;
	dedupeKey: string;
};

/** Debug history row (all posts while capture is on). */
export type NotificationHistoryItem = RawBankNotification & {
	inAllowlist: boolean;
	queued: boolean;
};

export type BankBinding = {
	packageName: string;
	bankId: string;
	accountId: string;
};

export type CardBinding = {
	bankId: string;
	last4: string;
	accountId: string;
};

export type InterceptSettings = {
	enabled: boolean;
	bankBindings: BankBinding[];
	cardBindings: CardBinding[];
};

export type ParsedPurchase = {
	bankId: string;
	packageName: string;
	amount: string;
	currency?: string;
	occurredAt: string;
	merchantText: string;
	last4?: string;
	rawHash: string;
	/** Purchase / income draft, or cancel/refund that drops a matching purchase draft. */
	kind?: 'purchase' | 'income' | 'cancel';
};

export type InterceptDraft = {
	id: string;
	createdAt: string;
	parsed: ParsedPurchase;
	accountId?: string;
	merchantId?: string;
	merchantName?: string;
};

export type TransactionCreatePrefill = {
	description?: string;
	amount?: string;
	accountId?: string;
	merchantId?: string;
	merchantName?: string;
	/** Suggested from prior txs of this merchant (same type). */
	categoryId?: string;
	subcategoryId?: string;
	/** ISO datetime */
	occurredAt?: string;
	/** expense (default) or income — selects create form type. */
	type?: 'expense' | 'income';
	/** Remove this intercept draft after successful create. */
	draftId?: string;
};
