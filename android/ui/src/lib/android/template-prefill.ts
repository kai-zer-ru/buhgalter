import {
	getTransactionTemplate,
	listAccounts,
	listCategories,
	listMerchants,
	listTags,
	type Transaction
} from '$lib/api/client';
import {
	hasTemplatePrefillWarnings,
	templateToRepeatFrom,
	type TemplatePrefillWarnings
} from '$lib/transaction-template-prefill';
import { readRefCache } from '$lib/offline/ref-cache';
import type { TransactionTemplate } from '$lib/api/client';

export async function loadTemplateRepeatFrom(
	templateId: string
): Promise<{ tx: Transaction; warnings: TemplatePrefillWarnings } | null> {
	let tpl =
		readRefCache<TransactionTemplate[]>('/api/v1/transaction-templates')?.find(
			(t) => t.id === templateId
		) ?? null;
	if (!tpl) {
		try {
			tpl = await getTransactionTemplate(templateId);
		} catch {
			return null;
		}
	}
	const [accounts, expenseCats, incomeCats, merchants, tags] = await Promise.all([
		listAccounts('active').catch(() => []),
		listCategories('expense').catch(() => []),
		listCategories('income').catch(() => []),
		listMerchants().catch(() => []),
		listTags().catch(() => [])
	]);
	return templateToRepeatFrom(tpl, {
		activeAccountIds: new Set(accounts.map((a) => a.id)),
		categoryIds: new Set([...expenseCats, ...incomeCats].map((c) => c.id)),
		merchantIds: new Set(merchants.map((m) => m.id)),
		tagIds: new Set(tags.map((t) => t.id))
	});
}

export { hasTemplatePrefillWarnings };
