import { describe, expect, it } from 'vitest';
import type { Transaction } from '$lib/api/client';
import {
	canDeleteTransaction,
	canEditTransaction,
	canRepeatTransaction,
	transactionCategoryLabel
} from './transaction-display';

function tx(overrides: Partial<Transaction> = {}): Transaction {
	return {
		id: 'tx-1',
		account_id: 'acc-1',
		type: 'expense',
		amount: 1000,
		amount_display: '10.00',
		kind: 'manual',
		description: null,
		category_id: null,
		subcategory_id: null,
		transaction_date: '2026-01-01T12:00:00Z',
		created_at: '2026-01-01T12:00:00Z',
		updated_at: '2026-01-01T12:00:00Z',
		...overrides
	};
}

describe('canRepeatTransaction', () => {
	it('allows regular income and expense', () => {
		expect(canRepeatTransaction(tx({ type: 'expense' }))).toBe(true);
		expect(canRepeatTransaction(tx({ type: 'income' }))).toBe(true);
	});

	it('allows transfers', () => {
		expect(
			canRepeatTransaction(
				tx({ type: 'transfer', transfer_group_id: 'grp-1', transfer_is_out: true })
			)
		).toBe(true);
	});

	it('blocks credit-linked payments like edit', () => {
		expect(canRepeatTransaction(tx({ credit_payment_linked: true }))).toBe(false);
		expect(canEditTransaction(tx({ credit_payment_linked: true }))).toBe(false);
	});

	it('blocks income and expense with system category', () => {
		expect(canRepeatTransaction(tx({ type: 'expense', category_is_system: true }))).toBe(false);
		expect(canRepeatTransaction(tx({ type: 'income', category_is_system: true }))).toBe(false);
	});
});

describe('canDeleteTransaction', () => {
	it('allows delete when deletable is omitted or true', () => {
		expect(canDeleteTransaction(tx())).toBe(true);
		expect(canDeleteTransaction(tx({ deletable: true }))).toBe(true);
	});

	it('blocks delete when deletable is false', () => {
		expect(canDeleteTransaction(tx({ deletable: false }))).toBe(false);
	});
});

describe('transactionCategoryLabel', () => {
	const t = (key: string) => (key === 'transactions.transfer' ? 'Transfer' : key);

	it('uses i18n for transfers regardless of server category name', () => {
		expect(transactionCategoryLabel({ type: 'transfer', category_name: 'transfer' }, t)).toBe(
			'Transfer'
		);
		expect(transactionCategoryLabel({ type: 'transfer', category_name: 'Перевод' }, t)).toBe(
			'Transfer'
		);
	});

	it('uses category name for income and expense', () => {
		expect(transactionCategoryLabel({ type: 'expense', category_name: 'Food' }, t)).toBe('Food');
		expect(transactionCategoryLabel({ type: 'income', category_name: null }, t)).toBe('income');
	});
});
