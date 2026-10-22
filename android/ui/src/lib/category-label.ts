import { tr } from '$lib/i18n';

type CategoryType = 'income' | 'expense';

type NamedCategory = { name: string; type?: CategoryType };

/** Names that appear more than once (e.g. system «Долги» for income and expense). */
export function duplicateCategoryNames(items: NamedCategory[]): Set<string> {
	const counts = new Map<string, number>();
	for (const item of items) {
		counts.set(item.name, (counts.get(item.name) ?? 0) + 1);
	}
	const dup = new Set<string>();
	for (const [name, count] of counts) {
		if (count > 1) dup.add(name);
	}
	return dup;
}

function typeSuffix(type: CategoryType): string {
	return type === 'income' ? tr('transactions.type.income') : tr('transactions.type.expense');
}

/** Label for selects and tables; adds «(Доход)» / «(Расход)» when names collide. */
export function categoryDisplayLabel(
	name: string,
	type: CategoryType,
	duplicates: Set<string>
): string {
	if (!duplicates.has(name)) return name;
	return `${name} (${typeSuffix(type)})`;
}

export function categorySelectLabel(
	cat: NamedCategory & { type: CategoryType },
	all: NamedCategory[]
): string {
	return categoryDisplayLabel(cat.name, cat.type, duplicateCategoryNames(all));
}
