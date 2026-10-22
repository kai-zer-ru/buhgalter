import type {
	TagRef,
	Transaction,
	TransactionTemplate,
	TransactionTemplateUpsert
} from '$lib/api/client';
import { fromCents } from '$lib/money';
import { transferAccountIds } from '$lib/transaction-display';

export type TemplatePrefillWarnings = {
	accountMissing?: boolean;
	toAccountMissing?: boolean;
	categoryMissing?: boolean;
	merchantMissing?: boolean;
};

export type TemplatePrefill = {
	tx: Transaction;
	warnings: TemplatePrefillWarnings;
};

type PrefillContext = {
	activeAccountIds: Set<string>;
	categoryIds: Set<string>;
	merchantIds: Set<string>;
	tagIds: Set<string>;
};

/** Map template → Transaction-shaped object for TransactionForm / TransferForm `repeatFrom`. */
export function templateToRepeatFrom(
	tpl: TransactionTemplate,
	ctx: PrefillContext
): TemplatePrefill {
	const warnings: TemplatePrefillWarnings = {};
	let accountId = tpl.account_id ?? '';
	if (accountId && !ctx.activeAccountIds.has(accountId)) {
		accountId = '';
		warnings.accountMissing = true;
	}
	let toAccountId = tpl.to_account_id ?? '';
	if (toAccountId && !ctx.activeAccountIds.has(toAccountId)) {
		toAccountId = '';
		warnings.toAccountMissing = true;
	}
	let categoryId = tpl.category_id ?? null;
	if (categoryId && !ctx.categoryIds.has(categoryId)) {
		categoryId = null;
		warnings.categoryMissing = true;
	}
	let merchantId = tpl.merchant_id ?? null;
	if (merchantId && !ctx.merchantIds.has(merchantId)) {
		merchantId = null;
		warnings.merchantMissing = true;
	}
	const tags = (tpl.tags ?? []).filter((t) => ctx.tagIds.has(t.id));

	const amount = tpl.amount ?? 0;
	const amountDisplay = tpl.amount != null ? fromCents(tpl.amount) : '';

	const tx: Transaction = {
		id: `template:${tpl.id}`,
		account_id: accountId,
		type: tpl.type,
		kind: 'manual',
		amount,
		amount_display: amountDisplay,
		description: tpl.description ?? null,
		merchant_id: merchantId,
		tags,
		category_id: categoryId,
		subcategory_id: tpl.subcategory_id ?? null,
		transfer_group_id: tpl.type === 'transfer' ? `template:${tpl.id}` : null,
		transfer_account_id: tpl.type === 'transfer' ? toAccountId : null,
		transfer_is_out: tpl.type === 'transfer' ? true : undefined,
		transaction_date: '',
		created_at: '',
		updated_at: ''
	};
	return { tx, warnings };
}

/** Build create payload from a journal transaction («Сохранить как шаблон»). */
export function templateUpsertFromTransaction(
	tx: Transaction,
	name: string
): TransactionTemplateUpsert {
	const tagIds = (tx.tags ?? []).map((t: TagRef) => t.id);
	if (tx.type === 'transfer') {
		const { fromAccountId, toAccountId } = transferAccountIds(tx);
		return {
			name,
			type: 'transfer',
			account_id: fromAccountId || null,
			to_account_id: toAccountId || null,
			amount: tx.amount,
			description: tx.description
		};
	}
	return {
		name,
		type: tx.type === 'income' ? 'income' : 'expense',
		account_id: tx.account_id || null,
		category_id: tx.category_id,
		subcategory_id: tx.subcategory_id,
		amount: tx.amount,
		description: tx.description,
		merchant_id: tx.merchant_id ?? null,
		tag_ids: tagIds
	};
}

export function defaultTemplateName(tx: Transaction): string {
	const desc = tx.description?.trim();
	if (desc) return desc.slice(0, 80);
	if (tx.type === 'transfer') {
		return (
			[tx.account_name, tx.transfer_account_name].filter(Boolean).join(' → ').slice(0, 80) ||
			'Перевод'
		);
	}
	const parts = [tx.category_name, tx.subcategory_name].filter(Boolean);
	if (parts.length) return parts.join(' · ').slice(0, 80);
	return tx.type === 'income' ? 'Доход' : 'Расход';
}

export function hasTemplatePrefillWarnings(w: TemplatePrefillWarnings): boolean {
	return Boolean(w.accountMissing || w.toAccountMissing || w.categoryMissing || w.merchantMissing);
}
