const DROPDOWN_MARGIN = 8;

export type DropdownPlacement = 'up' | 'down';

/** Space taken by fixed UI at the bottom (e.g. money keypad). */
function bottomObstructionPx(): number {
	if (typeof document === 'undefined') return 0;
	const raw = getComputedStyle(document.documentElement)
		.getPropertyValue('--money-keypad-inset')
		.trim();
	if (!raw) return 0;
	const px = Number.parseFloat(raw);
	return Number.isFinite(px) ? Math.max(0, px) : 0;
}

/**
 * Prefer opening downward. Open upward only when the list cannot fit below
 * (near the bottom / above a fixed bottom obstruction).
 */
export function decideDropdownPlacement(
	spaceBelow: number,
	spaceAbove: number,
	listHeight: number
): DropdownPlacement {
	if (spaceBelow >= listHeight + DROPDOWN_MARGIN) return 'down';
	if (spaceAbove >= listHeight + DROPDOWN_MARGIN) return 'up';
	return spaceAbove > spaceBelow ? 'up' : 'down';
}

function placementForTrigger(
	trigger: HTMLElement,
	listHeight: number,
	locked?: DropdownPlacement
): DropdownPlacement {
	if (locked) return locked;
	const rect = trigger.getBoundingClientRect();
	const spaceBelow = window.innerHeight - bottomObstructionPx() - rect.bottom;
	const spaceAbove = rect.top;
	return decideDropdownPlacement(spaceBelow, spaceAbove, listHeight);
}

export function actionMenuStyle(
	trigger: HTMLElement,
	menuHeight: number,
	align: 'start' | 'end' = 'end',
	menuWidth?: number,
	placement?: DropdownPlacement
): string {
	const rect = trigger.getBoundingClientRect();
	const margin = DROPDOWN_MARGIN;
	const viewportWidth = window.innerWidth;
	const width = Math.min(menuWidth ?? 176, viewportWidth - margin * 2);
	const openUp = placementForTrigger(trigger, menuHeight, placement) === 'up';

	let left = align === 'end' ? rect.right - width : rect.left;
	left = Math.max(margin, Math.min(left, viewportWidth - width - margin));

	return [
		'position:fixed',
		`left:${left}px`,
		`max-width:${viewportWidth - margin * 2}px`,
		openUp ? `bottom:${window.innerHeight - rect.top + 4}px` : `top:${rect.bottom + 4}px`,
		'z-index:70'
	].join(';');
}

/**
 * Styles for a dropdown list.
 * Non-portal (`top/bottom:100%`) assumes the list's positioned ancestor wraps only the
 * trigger — not the field label or hint — otherwise the panel looks detached.
 * Pass `placement` after the first open to keep direction stable across scroll/resize.
 */
export function dropdownListStyle(
	trigger: HTMLElement,
	listHeight: number,
	usePortal: boolean,
	placement?: DropdownPlacement
): string {
	const openUp = placementForTrigger(trigger, listHeight, placement) === 'up';

	if (usePortal) {
		const rect = trigger.getBoundingClientRect();
		return [
			'position:fixed',
			`left:${rect.left}px`,
			`width:${rect.width}px`,
			openUp ? `bottom:${window.innerHeight - rect.top + 4}px` : `top:${rect.bottom + 4}px`,
			'z-index:70'
		].join(';');
	}

	return openUp ? 'bottom:100%;margin-bottom:4px;top:auto;' : 'top:100%;margin-top:4px;';
}

/** Decide placement once (accounts for money-keypad inset). */
export function dropdownPlacementFor(trigger: HTMLElement, listHeight: number): DropdownPlacement {
	return placementForTrigger(trigger, listHeight);
}
