import type { Account, Bank, Category, Subcategory, UIMetaAccountRef } from '$lib/api/client';

export type SelectOptionIcon =
	| { type: 'account'; accountType: Account['type']; bankIcon?: string | null }
	| { type: 'category'; icon: string };

export type SelectOption = {
	value: string;
	label: string;
	disabled?: boolean;
	icon?: SelectOptionIcon;
};

export function accountFromUIMetaRef(
	acc: UIMetaAccountRef,
	banks: readonly Bank[]
): Pick<Account, 'id' | 'name' | 'type' | 'bank_id' | 'bank_icon'> {
	return {
		id: acc.id,
		name: acc.name,
		type: acc.type,
		bank_id: acc.bank_id ?? null,
		bank_icon: acc.bank_id ? (banks.find((b) => b.id === acc.bank_id)?.icon_path ?? null) : null
	};
}

export function accountsFromUIMeta(
	refs: readonly UIMetaAccountRef[],
	banks: readonly Bank[]
): Pick<Account, 'id' | 'name' | 'type' | 'bank_id' | 'bank_icon'>[] {
	return refs.map((acc) => accountFromUIMetaRef(acc, banks));
}

export function accountSelectIcon(acc: Pick<Account, 'type' | 'bank_icon'>): SelectOptionIcon {
	return { type: 'account', accountType: acc.type, bankIcon: acc.bank_icon };
}

export function accountRefSelectIcon(
	acc: Pick<UIMetaAccountRef, 'type' | 'bank_id'>,
	banks: readonly Bank[]
): SelectOptionIcon {
	const bankIcon = acc.bank_id
		? (banks.find((b) => b.id === acc.bank_id)?.icon_path ?? null)
		: null;
	return { type: 'account', accountType: acc.type, bankIcon };
}

export function categorySelectIcon(cat: Pick<Category, 'icon'>): SelectOptionIcon {
	return { type: 'category', icon: cat.icon };
}

export function subcategorySelectIcon(sub: Pick<Subcategory, 'icon'>): SelectOptionIcon {
	return { type: 'category', icon: sub.icon };
}

export function accountSelectOption(acc: Account, label?: string): SelectOption {
	return { value: acc.id, label: label ?? acc.name, icon: accountSelectIcon(acc) };
}

export function accountSelectOptions(
	accounts: readonly Account[],
	label?: (acc: Account) => string
): SelectOption[] {
	return accounts.map((acc) => accountSelectOption(acc, label?.(acc)));
}

export function accountRefSelectOption(
	acc: UIMetaAccountRef,
	banks: readonly Bank[],
	label?: string
): SelectOption {
	return {
		value: acc.id,
		label: label ?? acc.name,
		icon: accountRefSelectIcon(acc, banks)
	};
}

export function categorySelectOption(cat: Category, label?: string): SelectOption {
	return { value: cat.id, label: label ?? cat.name, icon: categorySelectIcon(cat) };
}

export function categorySelectOptions(
	categories: readonly Category[],
	label?: (cat: Category) => string
): SelectOption[] {
	return categories.map((cat) => categorySelectOption(cat, label?.(cat)));
}

export function subcategorySelectOption(sub: Subcategory, label?: string): SelectOption {
	return { value: sub.id, label: label ?? sub.name, icon: subcategorySelectIcon(sub) };
}

export function subcategorySelectOptions(
	subs: readonly Subcategory[],
	label?: (sub: Subcategory) => string
): SelectOption[] {
	return subs.map((sub) => subcategorySelectOption(sub, label?.(sub)));
}
