import iconsData from '../../../data/category_icons.json';

export type CategoryKind = 'expense' | 'income';
export type IconScope = 'expense' | 'income' | 'both';

export type CategoryIconDef = {
	id: string;
	emoji?: string;
	brand?: { bg: string; fg: string; label: string; size?: number };
	kind: IconScope;
	name: string;
	tags: string[];
};

export const CATEGORY_ICONS = iconsData.icons as CategoryIconDef[];

export const CATEGORY_ICON_IDS = new Set(CATEGORY_ICONS.map((i) => i.id));

const iconById = new Map(CATEGORY_ICONS.map((i) => [i.id, i]));

export function getCategoryIconDef(id: string): CategoryIconDef | undefined {
	return iconById.get(id);
}

export function isKnownCategoryIcon(id: string): boolean {
	return CATEGORY_ICON_IDS.has(id);
}

export function iconMatchesKind(iconId: string, kind: CategoryKind): boolean {
	const def = getCategoryIconDef(iconId);
	if (!def) return false;
	return def.kind === kind || def.kind === 'both';
}

export function iconsForKind(kind: CategoryKind): CategoryIconDef[] {
	return CATEGORY_ICONS.filter((icon) => icon.kind === kind || icon.kind === 'both');
}

export function quickIconsForKind(kind: CategoryKind): readonly string[] {
	return iconsData.quick[kind] as readonly string[];
}

/** Быстрый ряд: иконка из «Ещё» (или текущая, если её нет в quick) — на 1 месте, последняя из дефолтного ряда скрывается. */
export function quickIconsDisplay(
	kind: CategoryKind,
	options?: { pinned?: string | null; selected?: string }
): string[] {
	const base = [...quickIconsForKind(kind)];
	const pin =
		options?.pinned ??
		(options?.selected &&
		!base.includes(options.selected) &&
		iconMatchesKind(options.selected, kind)
			? options.selected
			: null);

	if (!pin) return base;

	const rest = base.slice(0, -1).filter((id) => id !== pin);
	return [pin, ...rest];
}

export function defaultIconForKind(kind: CategoryKind): string {
	const quick = quickIconsForKind(kind);
	return quick[quick.length - 1] ?? 'default';
}

export function defaultCategoryNameForIcon(id: string, kind: CategoryKind): string {
	if (id === 'default') {
		return kind === 'income' ? 'Прочие доходы' : 'Разное';
	}
	return getCategoryIconDef(id)?.name ?? '';
}

export function normalizeIconForKind(iconId: string, kind: CategoryKind): string {
	if (iconMatchesKind(iconId, kind)) return iconId;
	return defaultIconForKind(kind);
}

export function searchCategoryIcons(query: string, kind: CategoryKind): CategoryIconDef[] {
	const base = iconsForKind(kind);
	const q = query.trim().toLowerCase();
	if (!q) return base;
	return base.filter(
		(icon) => icon.id.includes(q) || icon.tags.some((tag) => tag.toLowerCase().includes(q))
	);
}
