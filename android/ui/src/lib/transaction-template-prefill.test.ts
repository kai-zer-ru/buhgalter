import { describe, expect, it } from 'vitest';
import type { Transaction, TransactionTemplate } from '$lib/api/client';
import {
	defaultTemplateName,
	templateToRepeatFrom,
	templateUpsertFromTransaction
} from './transaction-template-prefill';

const ctx = {
	activeAccountIds: new Set(['a1', 'a2']),
	categoryIds: new Set(['c1']),
	merchantIds: new Set(['m1']),
	tagIds: new Set(['t1', 't2'])
};

function tpl(partial: Partial<TransactionTemplate>): TransactionTemplate {
	return {
		id: 'tpl1',
		name: 'Обед',
		type: 'expense',
		sort_order: 1,
		tags: [{ id: 't1', name: 'еда' }],
		created_at: '',
		updated_at: '',
		...partial
	};
}

describe('templateToRepeatFrom', () => {
	it('maps expense with amount and clears dead refs', () => {
		const { tx, warnings } = templateToRepeatFrom(
			tpl({
				account_id: 'gone',
				category_id: 'c1',
				merchant_id: 'gone',
				amount: 25000,
				tags: [
					{ id: 't1', name: 'еда' },
					{ id: 'dead', name: 'x' }
				]
			}),
			ctx
		);
		expect(tx.account_id).toBe('');
		expect(tx.category_id).toBe('c1');
		expect(tx.merchant_id).toBeNull();
		expect(tx.amount_display).toBe('250.00');
		expect(tx.tags?.map((t) => t.id)).toEqual(['t1']);
		expect(warnings.accountMissing).toBe(true);
		expect(warnings.merchantMissing).toBe(true);
	});

	it('leaves amount empty when null', () => {
		const { tx } = templateToRepeatFrom(tpl({ account_id: 'a1', amount: undefined }), ctx);
		expect(tx.amount_display).toBe('');
		expect(tx.amount).toBe(0);
	});

	it('maps transfer with synthetic group', () => {
		const { tx } = templateToRepeatFrom(
			tpl({
				type: 'transfer',
				account_id: 'a1',
				to_account_id: 'a2',
				amount: 10000,
				tags: []
			}),
			ctx
		);
		expect(tx.type).toBe('transfer');
		expect(tx.transfer_group_id).toBe('template:tpl1');
		expect(tx.transfer_account_id).toBe('a2');
		expect(tx.transfer_is_out).toBe(true);
	});
});

describe('templateUpsertFromTransaction', () => {
	it('builds expense payload', () => {
		const tx = {
			id: '1',
			account_id: 'a1',
			type: 'expense',
			kind: 'manual',
			amount: 100,
			amount_display: '1.00',
			description: 'x',
			category_id: 'c1',
			subcategory_id: null,
			merchant_id: 'm1',
			tags: [{ id: 't1', name: 'еда' }],
			transaction_date: '',
			created_at: '',
			updated_at: ''
		} satisfies Transaction;
		expect(templateUpsertFromTransaction(tx, 'Обед')).toEqual({
			name: 'Обед',
			type: 'expense',
			account_id: 'a1',
			category_id: 'c1',
			subcategory_id: null,
			amount: 100,
			description: 'x',
			merchant_id: 'm1',
			tag_ids: ['t1']
		});
	});
});

describe('defaultTemplateName', () => {
	it('prefers description', () => {
		expect(
			defaultTemplateName({
				id: '1',
				account_id: 'a',
				type: 'expense',
				kind: 'manual',
				amount: 1,
				amount_display: '0.01',
				description: 'Ланч',
				category_id: null,
				subcategory_id: null,
				transaction_date: '',
				created_at: '',
				updated_at: ''
			})
		).toBe('Ланч');
	});
});
