import { describe, expect, it } from 'vitest';
import {
	canPairDraftsAsTransfer,
	findUniqueTransferComplement,
	listUniqueTransferPairs,
	unpairedInterceptDrafts,
	countVisibleInterceptItems,
	transferPrefillFromDrafts,
	transferPrefillFromSelection,
	TRANSFER_PAIR_WINDOW_MS
} from './draft-transfer';
import type { InterceptDraft, ParsedPurchase } from './types';

function draft(
	id: string,
	kind: 'purchase' | 'income',
	extra: Partial<ParsedPurchase> & { accountId?: string; createdAt?: string } = {}
): InterceptDraft {
	const { accountId, createdAt, ...parsed } = extra;
	return {
		id,
		createdAt: createdAt ?? '2026-09-22T10:00:00.000Z',
		accountId,
		parsed: {
			bankId: parsed.bankId ?? 'tinkoff',
			packageName: parsed.packageName ?? 'com.idamob.tinkoff.android',
			amount: parsed.amount ?? '1500.00',
			occurredAt: parsed.occurredAt ?? '2026-09-22T10:00:00.000Z',
			merchantText: parsed.merchantText ?? (kind === 'income' ? 'Пополнение' : 'Перевод'),
			rawHash: parsed.rawHash ?? id,
			kind,
			last4: parsed.last4
		}
	};
}

describe('draft transfer pairing', () => {
	const expense = draft('e1', 'purchase', { bankId: 'tinkoff', accountId: 'acc-t' });
	const income = draft('i1', 'income', {
		bankId: 'sberbank',
		packageName: 'ru.sberbankmobile',
		accountId: 'acc-s'
	});

	it('pairs expense and income with same amount and different accounts', () => {
		expect(canPairDraftsAsTransfer(expense, income)).toBe(true);
	});

	it('rejects same type', () => {
		const otherExpense = draft('e2', 'purchase', { bankId: 'sberbank', accountId: 'acc-s' });
		expect(canPairDraftsAsTransfer(expense, otherExpense)).toBe(false);
	});

	it('rejects different amounts', () => {
		const other = draft('i2', 'income', {
			bankId: 'sberbank',
			accountId: 'acc-s',
			amount: '1500.01'
		});
		expect(canPairDraftsAsTransfer(expense, other)).toBe(false);
	});

	it('rejects the same account', () => {
		const sameAcc = draft('i2', 'income', { bankId: 'sberbank', accountId: 'acc-t' });
		expect(canPairDraftsAsTransfer(expense, sameAcc)).toBe(false);
	});

	it('rejects same bank when accounts are unknown', () => {
		const a = draft('e2', 'purchase', { bankId: 'tinkoff' });
		const b = draft('i2', 'income', { bankId: 'tinkoff' });
		expect(canPairDraftsAsTransfer(a, b)).toBe(false);
	});

	it('allows same bank when accounts differ', () => {
		const a = draft('e2', 'purchase', { bankId: 'tinkoff', accountId: 'card' });
		const b = draft('i2', 'income', { bankId: 'tinkoff', accountId: 'save' });
		expect(canPairDraftsAsTransfer(a, b)).toBe(true);
	});

	it('allows different banks when accounts are unknown', () => {
		const a = draft('e2', 'purchase', { bankId: 'tinkoff' });
		const b = draft('i2', 'income', { bankId: 'sberbank' });
		expect(canPairDraftsAsTransfer(a, b)).toBe(true);
	});

	it('rejects drafts outside the time window', () => {
		const late = draft('i2', 'income', {
			bankId: 'sberbank',
			accountId: 'acc-s',
			occurredAt: new Date(
				Date.parse(expense.parsed.occurredAt) + TRANSFER_PAIR_WINDOW_MS + 1000
			).toISOString()
		});
		expect(canPairDraftsAsTransfer(expense, late)).toBe(false);
	});

	it('finds a unique complement', () => {
		expect(findUniqueTransferComplement(expense, [expense, income])?.id).toBe('i1');
	});

	it('does not pair when several complements match', () => {
		const income2 = draft('i2', 'income', {
			bankId: 'alfabank',
			accountId: 'acc-a'
		});
		expect(findUniqueTransferComplement(expense, [expense, income, income2])).toBeNull();
	});

	it('lists unique pairs only', () => {
		const noise = draft('e3', 'purchase', {
			bankId: 'yandex',
			accountId: 'acc-y',
			amount: '80.00'
		});
		expect(listUniqueTransferPairs([expense, income, noise])).toEqual([
			{ from: expense, to: income }
		]);
	});

	it('hides unique pair members from the unpaired list', () => {
		const noise = draft('e3', 'purchase', {
			bankId: 'yandex',
			accountId: 'acc-y',
			amount: '80.00'
		});
		expect(unpairedInterceptDrafts([expense, income, noise])).toEqual([noise]);
		expect(unpairedInterceptDrafts([expense, income])).toEqual([]);
		expect(countVisibleInterceptItems([expense, income])).toBe(1);
		expect(countVisibleInterceptItems([expense, income, noise])).toBe(2);
	});

	it('builds transfer prefill from a pair', () => {
		const prefill = transferPrefillFromDrafts(income, expense);
		expect(prefill).toEqual({
			fromAccountId: 'acc-t',
			toAccountId: 'acc-s',
			amount: '1500.00',
			occurredAt: '2026-09-22T10:00:00.000Z',
			draftIds: ['e1', 'i1']
		});
	});

	it('builds one-sided prefill from an expense draft', () => {
		expect(transferPrefillFromDrafts(expense)).toMatchObject({
			fromAccountId: 'acc-t',
			toAccountId: undefined,
			amount: '1500.00',
			draftIds: ['e1']
		});
	});

	it('builds one-sided prefill from an income draft', () => {
		expect(transferPrefillFromDrafts(income)).toMatchObject({
			fromAccountId: undefined,
			toAccountId: 'acc-s',
			draftIds: ['i1']
		});
	});

	it('uses the earlier occurredAt for a pair', () => {
		const laterIncome = draft('i3', 'income', {
			bankId: 'sberbank',
			accountId: 'acc-s',
			occurredAt: '2026-09-22T10:05:00.000Z'
		});
		expect(transferPrefillFromDrafts(expense, laterIncome)?.occurredAt).toBe(
			'2026-09-22T10:00:00.000Z'
		);
	});

	it('returns null prefill for an invalid two-draft selection', () => {
		const otherExpense = draft('e2', 'purchase', { bankId: 'sberbank', accountId: 'acc-s' });
		expect(transferPrefillFromSelection([expense, otherExpense])).toBeNull();
	});

	it('accepts a single selected draft', () => {
		expect(transferPrefillFromSelection([income])?.toAccountId).toBe('acc-s');
	});
});
