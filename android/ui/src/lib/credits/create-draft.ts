import { get, writable } from 'svelte/store';
import {
	ApiError,
	createCredit,
	listAccounts,
	listBanks,
	previewCreditSchedule,
	type Account,
	type Bank,
	type Credit
} from '$lib/api/client';
import { defaultAccountId } from '$lib/accounts';
import { defaultAutoDebitTimeLocal } from '$lib/datetime-picker-standards';
import {
	dateOnlyLocalValue,
	fromDatetimeLocalValue,
	isFutureDatetimeLocal,
	todayDateLocal,
	toDatetimeLocalValue
} from '$lib/dates';
import { formatMoneyForInput, formatMoneyInput, fromCents, toAPIAmount, toCents } from '$lib/money';

export type ProductType = 'credit' | 'installment' | 'mortgage';
export type PaymentInterval = 'month' | 'week' | 'two_weeks' | 'manual';
export type ScheduleRow = { date: string; amount: string };

export type CreditCreateDraft = {
	productType: ProductType;
	name: string;
	principal: string;
	propertyPrice: string;
	downPayment: string;
	downPaymentAffectsBalance: boolean;
	downPaymentAccountId: string;
	issueDateLocal: string;
	termMonths: string;
	interestRate: string;
	interval: PaymentInterval;
	calculatedPayment: string;
	paymentOverride: string | null;
	firstPaymentToday: boolean;
	debitAccountId: string;
	createTransactions: boolean;
	retroactive: boolean;
	retroactiveDebitCount: number;
	principalAffectsBalance: boolean;
	bankId: string;
	debitTimeLocal: string;
	scheduleRows: ScheduleRow[];
	scheduleLoading: boolean;
	scheduleError: string;
	lastScheduleKey: string;
	lastBaseScheduleKey: string;
	schedulePage: number;
	accounts: Account[];
	banks: Bank[];
	saving: boolean;
};

export const schedulePageSize = 10;

export const creditCreateDraft = writable<CreditCreateDraft | null>(null);

export function emptyCreditCreateDraft(tz: string): CreditCreateDraft {
	return {
		productType: 'credit',
		name: '',
		principal: '',
		propertyPrice: '',
		downPayment: '',
		downPaymentAffectsBalance: false,
		downPaymentAccountId: '',
		issueDateLocal: todayDateLocal(tz),
		termMonths: '12',
		interestRate: '12',
		interval: 'month',
		calculatedPayment: '',
		paymentOverride: null,
		firstPaymentToday: false,
		debitAccountId: '',
		createTransactions: true,
		retroactive: false,
		retroactiveDebitCount: 0,
		principalAffectsBalance: false,
		bankId: '',
		debitTimeLocal: '',
		scheduleRows: [],
		scheduleLoading: false,
		scheduleError: '',
		lastScheduleKey: '',
		lastBaseScheduleKey: '',
		schedulePage: 1,
		accounts: [],
		banks: [],
		saving: false
	};
}

export function beginCreditCreate(tz: string) {
	creditCreateDraft.set(emptyCreditCreateDraft(tz));
}

export function clearCreditCreate() {
	creditCreateDraft.set(null);
}

export function patchCreditCreate(patch: Partial<CreditCreateDraft>) {
	creditCreateDraft.update((d) => (d ? { ...d, ...patch } : d));
}

export function effectivePrincipal(d: CreditCreateDraft): string {
	if (d.productType !== 'mortgage') return d.principal;
	if (!d.propertyPrice.trim()) return '';
	try {
		const property = toCents(d.propertyPrice);
		const down = d.downPayment.trim() ? toCents(d.downPayment) : 0;
		if (down >= property) return '';
		return fromCents(property - down);
	} catch {
		return '';
	}
}

export function termCount(d: CreditCreateDraft): number {
	return Math.max(1, Number(d.termMonths) || 1);
}

export function isManualInterval(d: CreditCreateDraft): boolean {
	return d.interval === 'manual';
}

export function supportsFirstPaymentToday(d: CreditCreateDraft): boolean {
	return d.productType !== 'mortgage';
}

export function hasDownPayment(d: CreditCreateDraft): boolean {
	if (!d.downPayment.trim()) return false;
	try {
		return toCents(d.downPayment) > 0;
	} catch {
		return false;
	}
}

export function baseScheduleParamsKey(d: CreditCreateDraft): string {
	return [
		effectivePrincipal(d),
		d.termMonths,
		d.issueDateLocal,
		d.interval,
		d.productType,
		d.interestRate
	].join('|');
}

export function scheduleParamsKey(d: CreditCreateDraft): string {
	return [
		baseScheduleParamsKey(d),
		d.paymentOverride ?? '',
		supportsFirstPaymentToday(d) && d.firstPaymentToday ? '1' : '0'
	].join('|');
}

export function averageFromScheduleRows(d: CreditCreateDraft): string {
	const cents: number[] = [];
	for (const r of d.scheduleRows) {
		if (!r.amount.trim()) continue;
		try {
			cents.push(toCents(r.amount));
		} catch {
			/* skip */
		}
	}
	if (cents.length > 0) {
		const sum = cents.reduce((a, b) => a + b, 0);
		return fromCents(Math.round(sum / cents.length));
	}
	const principal = effectivePrincipal(d);
	if (!principal.trim()) return '—';
	try {
		return fromCents(Math.floor(toCents(principal) / termCount(d)));
	} catch {
		return '—';
	}
}

export function displayedPayment(d: CreditCreateDraft): string {
	if (isManualInterval(d)) return averageFromScheduleRows(d);
	return (d.paymentOverride ?? d.calculatedPayment) || '—';
}

export function hasPastSchedulePayments(d: CreditCreateDraft, tz: string): boolean {
	if (d.scheduleRows.length === 0) return false;
	const todayDay = todayDateLocal(tz).slice(0, 10);
	return d.scheduleRows.some((row) => {
		if (!row.date.trim()) return false;
		return dateOnlyLocalValue(row.date).slice(0, 10) < todayDay;
	});
}

/** Same as past payments in schedule — blocks «principal as income» when any installment date is before today. */
export function principalIncomeBlocked(d: CreditCreateDraft, tz: string): boolean {
	return hasPastSchedulePayments(d, tz);
}

export function rowStatus(
	d: CreditCreateDraft,
	row: ScheduleRow,
	tz: string
): 'retroactive' | 'pending' | null {
	if (!row.date.trim()) return null;
	if (!d.retroactive) return 'pending';
	return isFutureDatetimeLocal(row.date, tz) ? 'pending' : 'retroactive';
}

export function retroRowIndices(d: CreditCreateDraft, tz: string): number[] {
	const out: number[] = [];
	for (let i = 0; i < d.scheduleRows.length; i++) {
		if (rowStatus(d, d.scheduleRows[i], tz) === 'retroactive') out.push(i);
	}
	return out;
}

export function isRetroDebited(d: CreditCreateDraft, rowIndex: number, tz: string): boolean {
	const indices = retroRowIndices(d, tz);
	const pos = indices.indexOf(rowIndex);
	if (pos < 0) return false;
	return pos >= indices.length - d.retroactiveDebitCount;
}

export function canToggleRetroDebit(d: CreditCreateDraft, rowIndex: number, tz: string): boolean {
	const indices = retroRowIndices(d, tz);
	const pos = indices.indexOf(rowIndex);
	if (pos < 0) return false;
	const len = indices.length;
	const n = d.retroactiveDebitCount;
	if (n > 0 && pos === len - n) return true;
	if (n < len && pos === len - n - 1) return true;
	return false;
}

export function toggleRetroDebit(
	d: CreditCreateDraft,
	rowIndex: number,
	tz: string
): CreditCreateDraft {
	const indices = retroRowIndices(d, tz);
	const pos = indices.indexOf(rowIndex);
	if (pos < 0 || !canToggleRetroDebit(d, rowIndex, tz)) return d;
	const len = indices.length;
	const n = d.retroactiveDebitCount;
	let next = n;
	if (n > 0 && pos === len - n) next = n - 1;
	else if (n < len && pos === len - n - 1) next = n + 1;
	return { ...d, retroactiveDebitCount: next };
}

export function scheduleRowsComplete(d: CreditCreateDraft): boolean {
	return d.scheduleRows.length > 0 && d.scheduleRows.every((r) => r.date.trim() && r.amount.trim());
}

export function buildScheduleSeed(d: CreditCreateDraft, tz: string) {
	if (!isManualInterval(d)) return [];
	return d.scheduleRows
		.filter((r) => r.date && r.amount)
		.map((r) => ({
			payment_date: fromDatetimeLocalValue(r.date, tz),
			amount: toAPIAmount(r.amount)
		}));
}

export function buildCreatePayload(d: CreditCreateDraft, tz: string): Record<string, unknown> {
	const principal = effectivePrincipal(d);
	const showDebitTime = d.createTransactions;
	return {
		name: d.name.trim() || null,
		credit_kind: d.productType === 'mortgage' ? 'mortgage' : 'consumer',
		principal_amount: toAPIAmount(principal),
		property_price: d.productType === 'mortgage' ? toAPIAmount(d.propertyPrice) : null,
		down_payment: d.productType === 'mortgage' ? toAPIAmount(d.downPayment || '0') : null,
		down_payment_affects_balance:
			d.productType === 'mortgage' ? d.downPaymentAffectsBalance : false,
		down_payment_account_id:
			d.productType === 'mortgage' && d.downPaymentAffectsBalance
				? d.downPaymentAccountId || d.debitAccountId
				: null,
		issue_date: fromDatetimeLocalValue(d.issueDateLocal, tz),
		term_months: Number(d.termMonths),
		interest_rate: d.productType === 'installment' ? 0 : Number(d.interestRate) || 0,
		payment_interval: d.interval,
		paid_amount: '0',
		monthly_payment:
			!isManualInterval(d) && d.paymentOverride ? toAPIAmount(d.paymentOverride) : null,
		debit_account_id: d.debitAccountId,
		debit_time_local: showDebitTime ? d.debitTimeLocal.trim() || defaultAutoDebitTimeLocal : null,
		bank_id: d.bankId || null,
		added_retroactively: d.retroactive,
		retroactive_debit_count: d.retroactive ? d.retroactiveDebitCount : 0,
		principal_affects_balance: d.productType === 'credit' ? d.principalAffectsBalance : false,
		first_payment_today: supportsFirstPaymentToday(d) ? d.firstPaymentToday : false,
		create_transactions: d.createTransactions,
		schedule_seed: buildScheduleSeed(d, tz)
	};
}

/** Validation error i18n key, or null if OK. */
export function validateBasics(d: CreditCreateDraft): string | null {
	if (!effectivePrincipal(d).trim()) return 'credits.error.mortgageFields';
	if (!d.issueDateLocal.trim()) return 'credits.error.mortgageFields';
	if (!(Number(d.termMonths) > 0)) return 'credits.error.mortgageFields';
	return null;
}

export function validateReadyToSave(d: CreditCreateDraft): string | null {
	if (!d.debitAccountId) return 'credits.error.noAccount';
	const basics = validateBasics(d);
	if (basics) return basics;
	if (!scheduleRowsComplete(d)) {
		return isManualInterval(d) ? 'credits.error.manualIncomplete' : 'credits.schedule.empty';
	}
	return null;
}

export async function loadCreditCreateRefs() {
	const d = get(creditCreateDraft);
	if (!d) return;
	try {
		const [accounts, banks] = await Promise.all([listAccounts(), listBanks()]);
		const active = accounts.filter((a) => a.status === 'active');
		patchCreditCreate({
			accounts: active,
			banks,
			debitAccountId: defaultAccountId(active, d.debitAccountId),
			downPaymentAccountId: defaultAccountId(active, d.downPaymentAccountId || d.debitAccountId)
		});
	} catch {
		patchCreditCreate({ accounts: [], banks: [] });
	}
}

export function ensureManualScheduleRows(d: CreditCreateDraft): CreditCreateDraft {
	if (!isManualInterval(d)) return d;
	const n = termCount(d);
	let rows = d.scheduleRows;
	if (rows.length < n) {
		rows = [...rows];
		while (rows.length < n) rows.push({ date: '', amount: '' });
	} else if (rows.length > n) {
		rows = rows.slice(0, n);
	}
	const principal = effectivePrincipal(d);
	if (principal.trim() && rows.length === n && rows.every((r) => !r.amount.trim())) {
		try {
			const total = toCents(principal);
			const base = Math.floor(total / n);
			const lastAmt = total - base * (n - 1);
			rows = rows.map((r, i) => ({
				...r,
				amount: fromCents(i === n - 1 ? lastAmt : base)
			}));
		} catch {
			/* keep */
		}
	}
	if (rows === d.scheduleRows) return d;
	return { ...d, scheduleRows: rows };
}

export async function refreshCreditCreateSchedule(
	tz: string,
	explicitOverride: string | null | undefined = undefined
) {
	const d0 = get(creditCreateDraft);
	if (!d0 || isManualInterval(d0)) return;
	const principal = effectivePrincipal(d0);
	if (!principal.trim() || !d0.termMonths) return;

	const base = baseScheduleParamsKey(d0);
	let draft = d0;
	if (d0.lastBaseScheduleKey && base !== d0.lastBaseScheduleKey) {
		draft = { ...d0, paymentOverride: null, lastScheduleKey: '', lastBaseScheduleKey: base };
		creditCreateDraft.set(draft);
	} else if (!d0.lastBaseScheduleKey) {
		draft = { ...d0, lastBaseScheduleKey: base };
		creditCreateDraft.set(draft);
	}

	const expectedKey = scheduleParamsKey(draft);
	if (expectedKey === draft.lastScheduleKey && explicitOverride === undefined) return;

	patchCreditCreate({ scheduleLoading: true, scheduleError: '' });
	const overrideForRequest =
		explicitOverride !== undefined
			? explicitOverride
			: draft.paymentOverride
				? draft.paymentOverride
				: null;
	try {
		const res = await previewCreditSchedule({
			principal: toAPIAmount(principal),
			term: Number(draft.termMonths),
			interest_rate: draft.productType === 'installment' ? 0 : Number(draft.interestRate) || 0,
			payment_interval: draft.interval,
			issue_date: fromDatetimeLocalValue(draft.issueDateLocal, tz),
			credit_kind: draft.productType === 'mortgage' ? 'mortgage' : 'consumer',
			monthly_payment: overrideForRequest ? toAPIAmount(overrideForRequest) : null,
			first_payment_today: supportsFirstPaymentToday(draft) ? draft.firstPaymentToday : false
		});
		const current = get(creditCreateDraft);
		if (!current || scheduleParamsKey(current) !== expectedKey) return;
		let paymentOverride = current.paymentOverride;
		if (explicitOverride !== undefined) {
			paymentOverride = explicitOverride ? formatMoneyInput(explicitOverride) : null;
		} else if (res.user_set_monthly_payment && res.effective_monthly_payment_display) {
			paymentOverride = res.effective_monthly_payment_display;
		} else if (!overrideForRequest) {
			paymentOverride = null;
		}
		patchCreditCreate({
			scheduleRows: (res.schedule_preview ?? []).map((row) => ({
				date: dateOnlyLocalValue(toDatetimeLocalValue(row.payment_date, tz)),
				amount: formatMoneyForInput(row.amount_display ?? fromCents(row.amount))
			})),
			calculatedPayment: res.calculated_monthly_payment_display,
			paymentOverride,
			lastScheduleKey: expectedKey,
			schedulePage: 1,
			scheduleError: ''
		});
	} catch (e) {
		const current = get(creditCreateDraft);
		if (!current || scheduleParamsKey(current) !== expectedKey) return;
		patchCreditCreate({
			scheduleError: e instanceof ApiError ? e.message : 'Не удалось рассчитать график',
			lastScheduleKey: ''
		});
	} finally {
		patchCreditCreate({ scheduleLoading: false });
	}
}

export async function applyPaymentOverride(draftAmount: string, tz: string): Promise<boolean> {
	const draft = draftAmount.trim();
	let nextOverride: string | null = null;
	if (draft) {
		try {
			const normalized = formatMoneyInput(draft);
			if (normalized) nextOverride = normalized;
		} catch {
			return false;
		}
	}
	patchCreditCreate({ paymentOverride: nextOverride, lastScheduleKey: '' });
	await refreshCreditCreateSchedule(tz, nextOverride);
	return true;
}

export async function submitCreditCreate(tz: string): Promise<Credit> {
	const d = get(creditCreateDraft);
	if (!d) throw new Error('no draft');
	const err = validateReadyToSave(d);
	if (err) {
		const e = new Error(err);
		(e as Error & { i18nKey?: string }).i18nKey = err;
		throw e;
	}
	patchCreditCreate({ saving: true });
	try {
		const created = await createCredit(buildCreatePayload(d, tz));
		clearCreditCreate();
		return created;
	} finally {
		const cur = get(creditCreateDraft);
		if (cur) patchCreditCreate({ saving: false });
	}
}
