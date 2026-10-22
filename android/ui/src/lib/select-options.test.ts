import { describe, expect, it } from 'vitest';
import type { Account, Bank, Category, Subcategory } from '$lib/api/client';
import {
	accountFromUIMetaRef,
	accountRefSelectIcon,
	accountRefSelectOption,
	accountSelectIcon,
	accountSelectOption,
	categorySelectIcon,
	subcategorySelectIcon
} from './select-options';

const bank: Bank = {
	id: 'bank-1',
	name: 'Yandex',
	bic: null,
	icon_path: '/icons/yandex.png',
	sort_order: 0
};

function account(overrides: Partial<Account> = {}): Account {
	return {
		id: 'acc-1',
		name: 'Wallet',
		type: 'cash',
		bank_id: null,
		initial_balance: 0,
		balance: 0,
		balance_display: '0.00',
		status: 'active',
		is_primary: false,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z',
		...overrides
	};
}

describe('accountFromUIMetaRef', () => {
	it('resolves bank icon from meta banks', () => {
		expect(
			accountFromUIMetaRef(
				{ id: 'a', name: 'Yandex', type: 'bank', status: 'active', bank_id: 'bank-1' },
				[bank]
			)
		).toEqual({
			id: 'a',
			name: 'Yandex',
			type: 'bank',
			bank_id: 'bank-1',
			bank_icon: '/icons/yandex.png'
		});
	});
});

describe('accountSelectIcon', () => {
	it('maps cash account without bank icon', () => {
		expect(accountSelectIcon(account())).toEqual({
			type: 'account',
			accountType: 'cash',
			bankIcon: undefined
		});
	});

	it('maps bank account with bank icon', () => {
		expect(accountSelectIcon(account({ type: 'bank', bank_icon: '/icons/yandex.png' }))).toEqual({
			type: 'account',
			accountType: 'bank',
			bankIcon: '/icons/yandex.png'
		});
	});
});

describe('accountRefSelectIcon', () => {
	it('resolves bank icon from meta banks', () => {
		expect(accountRefSelectIcon({ type: 'bank', bank_id: 'bank-1' }, [bank])).toEqual({
			type: 'account',
			accountType: 'bank',
			bankIcon: '/icons/yandex.png'
		});
	});
});

describe('accountSelectOption', () => {
	it('includes value, label and icon', () => {
		expect(accountSelectOption(account({ id: 'a', name: 'Cash' }))).toEqual({
			value: 'a',
			label: 'Cash',
			icon: { type: 'account', accountType: 'cash', bankIcon: undefined }
		});
	});
});

describe('accountRefSelectOption', () => {
	it('builds option from ui meta account ref', () => {
		expect(
			accountRefSelectOption(
				{ id: 'a', name: 'Yandex', type: 'bank', status: 'active', bank_id: 'bank-1' },
				[bank]
			)
		).toEqual({
			value: 'a',
			label: 'Yandex',
			icon: { type: 'account', accountType: 'bank', bankIcon: '/icons/yandex.png' }
		});
	});
});

describe('categorySelectIcon', () => {
	it('maps category icon path', () => {
		const cat: Category = {
			id: 'c1',
			name: 'Food',
			type: 'expense',
			icon: 'food',
			sort_order: 0,
			is_primary: false,
			is_system: false,
			subcategory_count: 0,
			created_at: '2026-01-01T00:00:00Z'
		};
		expect(categorySelectIcon(cat)).toEqual({ type: 'category', icon: 'food' });
	});
});

describe('subcategorySelectIcon', () => {
	it('maps subcategory icon path', () => {
		const sub: Subcategory = {
			id: 's1',
			category_id: 'c1',
			name: 'Groceries',
			icon: 'groceries',
			sort_order: 0,
			created_at: '2026-01-01T00:00:00Z'
		};
		expect(subcategorySelectIcon(sub)).toEqual({ type: 'category', icon: 'groceries' });
	});
});
