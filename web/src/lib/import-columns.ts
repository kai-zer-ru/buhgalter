/** Target fields for custom import column_map (matches backend importexport constants). */
export type ImportColumnField =
	| 'type'
	| 'date'
	| 'debit_amount'
	| 'debit_account'
	| 'credit_amount'
	| 'credit_account'
	| 'category'
	| 'subcategory'
	| 'description'
	| 'project'
	| 'user'
	| 'debit_currency'
	| 'credit_currency';

export type ImportColumnFieldMeta = {
	id: ImportColumnField;
	required?: boolean;
};

export const IMPORT_COLUMN_FIELDS: ImportColumnFieldMeta[] = [
	{ id: 'type', required: true },
	{ id: 'date', required: true },
	{ id: 'debit_amount' },
	{ id: 'debit_account' },
	{ id: 'credit_amount' },
	{ id: 'credit_account' },
	{ id: 'category' },
	{ id: 'subcategory' },
	{ id: 'description' },
	{ id: 'project' },
	{ id: 'user' },
	{ id: 'debit_currency' },
	{ id: 'credit_currency' }
];

const GUESS_ALIASES: Record<ImportColumnField, string[]> = {
	type: ['тип', 'type'],
	date: ['дата', 'date'],
	debit_amount: ['сумма списания', 'debit', 'amount out', 'расход'],
	credit_amount: ['сумма пополнения', 'credit', 'amount in', 'доход'],
	debit_account: ['счет списания', 'счёт списания', 'from account', 'from'],
	credit_account: ['счет пополнения', 'счёт пополнения', 'to account', 'to'],
	category: ['категория', 'category'],
	subcategory: ['subcategory', 'подкатегория'],
	description: ['описание', 'description', 'memo', 'note'],
	project: ['проект', 'project'],
	user: ['пользователь', 'user'],
	debit_currency: ['валюта списания', 'debit currency'],
	credit_currency: ['валюта назначения', 'credit currency']
};

export function guessColumnMap(headers: string[]): Record<string, string> {
	const out: Record<string, string> = {};
	const used = new Set<string>();
	for (const field of IMPORT_COLUMN_FIELDS) {
		const aliases = GUESS_ALIASES[field.id];
		for (const header of headers) {
			if (used.has(header)) continue;
			const h = header.trim().toLowerCase();
			for (const alias of aliases) {
				if (h === alias || h.includes(alias)) {
					out[field.id] = header;
					used.add(header);
					break;
				}
			}
			if (out[field.id]) break;
		}
	}
	return out;
}

export function isColumnMapValid(map: Record<string, string>): boolean {
	return Boolean(map.type?.trim() && map.date?.trim());
}

/** Suggested account type when auto-creating from import file. */
export function accountTypeLabel(type: string, t: (key: string) => string): string {
	if (type === 'bank') return t('accounts.type.bank');
	return t('accounts.type.cash');
}
