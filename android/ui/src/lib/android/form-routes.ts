/** Dynamically built href; cast to a single Pathname literal for Kit's typed `resolve`. */
export type AppHref = '/';

function href(path: string): AppHref {
	return path as AppHref;
}

/** Safe return path from `from` query param (internal routes only). */
export function parseFormReturnPath(raw: string | null, fallback = '/'): AppHref {
	if (!raw || !raw.startsWith('/') || raw.startsWith('//')) return href(fallback);
	return href(raw);
}

export function transactionNewPath(opts: {
	type: 'expense' | 'income';
	accountId?: string;
	repeatId?: string;
	from?: string;
}): AppHref {
	const params = new URLSearchParams({ type: opts.type });
	if (opts.accountId) params.set('account', opts.accountId);
	if (opts.repeatId) params.set('repeat', opts.repeatId);
	if (opts.from) params.set('from', opts.from);
	return href(`/transactions/new?${params}`);
}

export function transactionEditPath(id: string, from?: string): AppHref {
	const params = from ? `?from=${encodeURIComponent(from)}` : '';
	return href(`/transactions/${id}/edit${params}`);
}

export function transferNewPath(opts: {
	accountId?: string;
	payCardId?: string;
	repeatId?: string;
	from?: string;
}): AppHref {
	const params = new URLSearchParams();
	if (opts.accountId) params.set('account', opts.accountId);
	if (opts.payCardId) params.set('payCard', opts.payCardId);
	if (opts.repeatId) params.set('repeat', opts.repeatId);
	if (opts.from) params.set('from', opts.from);
	const q = params.toString();
	return href(q ? `/transfers/new?${q}` : '/transfers/new');
}

export function transferEditPath(groupId: string, from?: string): AppHref {
	const params = from ? `?from=${encodeURIComponent(from)}` : '';
	return href(`/transfers/${groupId}/edit${params}`);
}

export function accountNewPath(from?: string): AppHref {
	if (!from) return href('/accounts/new');
	return href(`/accounts/new?from=${encodeURIComponent(from)}`);
}

export function debtNewPath(opts: {
	direction: 'lent' | 'borrowed';
	debtorId?: string;
	from?: string;
}): AppHref {
	const params = new URLSearchParams({ direction: opts.direction });
	if (opts.debtorId) params.set('debtor', opts.debtorId);
	if (opts.from) params.set('from', opts.from);
	return href(`/debts/new?${params.toString()}`);
}

function withFromQuery(base: string, from?: string): AppHref {
	if (!from) return href(base);
	return href(`${base}?from=${encodeURIComponent(from)}`);
}

export function debtSettlePath(debtId: string, from?: string): AppHref {
	return withFromQuery(`/debts/${debtId}/settle`, from);
}

export function accountChargeFeePath(accountId: string, from?: string): AppHref {
	return withFromQuery(`/accounts/${accountId}/charge-fee`, from);
}

export function accountAutoTopupPath(accountId: string, from?: string): AppHref {
	return withFromQuery(`/accounts/${accountId}/auto-topup`, from);
}

export type CreditFormAction =
	| 'pay'
	| 'complete'
	| 'change-account'
	| 'debit-time'
	| 'change-name'
	| 'change-bank';

export type CreditCreateStep = 'basics' | 'options' | 'schedule';

export function creditNewPath(from?: string): AppHref {
	return withFromQuery('/credits/new', from);
}

export function creditCreateStepPath(step: CreditCreateStep, from?: string): AppHref {
	return withFromQuery(`/credits/new/${step}`, from);
}

export function creditActionPath(
	creditId: string,
	action: CreditFormAction,
	opts?: { from?: string; amount?: string; date?: string }
): AppHref {
	const base = `/credits/${creditId}/${action}`;
	const params = new URLSearchParams();
	if (opts?.from) params.set('from', opts.from);
	if (opts?.amount) params.set('amount', opts.amount);
	if (opts?.date) params.set('date', opts.date);
	const q = params.toString();
	return href(q ? `${base}?${q}` : base);
}
