import { createTransactionTemplate, type Transaction } from '$lib/api/client';
import { promptText } from '$lib/prompt';
import {
	defaultTemplateName,
	templateUpsertFromTransaction
} from '$lib/transaction-template-prefill';
import { toast } from '$lib/toast';
import { tr } from '$lib/i18n';

export async function saveTransactionAsTemplate(tx: Transaction): Promise<boolean> {
	const name = await promptText({
		title: tr('templates.saveAs'),
		message: tr('templates.saveAs.name'),
		defaultValue: defaultTemplateName(tx),
		maxLength: 80
	});
	if (!name) return false;
	try {
		await createTransactionTemplate(templateUpsertFromTransaction(tx, name));
		toast(tr('templates.saveAs.saved'));
		return true;
	} catch (err) {
		toast.fromError(err);
		return false;
	}
}
