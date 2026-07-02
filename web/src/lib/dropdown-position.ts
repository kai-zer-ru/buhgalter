const DROPDOWN_MARGIN = 8;

function shouldOpenUp(spaceBelow: number, spaceAbove: number, listHeight: number): boolean {
	if (spaceBelow >= listHeight + DROPDOWN_MARGIN) return false;
	if (spaceAbove >= listHeight + DROPDOWN_MARGIN) return true;
	return spaceAbove > spaceBelow;
}

export function actionMenuStyle(
	trigger: HTMLElement,
	menuHeight: number,
	align: 'start' | 'end' = 'end',
	menuWidth?: number
): string {
	const rect = trigger.getBoundingClientRect();
	const margin = DROPDOWN_MARGIN;
	const viewportWidth = window.innerWidth;
	const width = Math.min(menuWidth ?? 176, viewportWidth - margin * 2);

	const spaceBelow = window.innerHeight - rect.bottom;
	const spaceAbove = rect.top;
	const openUp = shouldOpenUp(spaceBelow, spaceAbove, menuHeight);

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

export function dropdownListStyle(
	trigger: HTMLElement,
	listHeight: number,
	usePortal: boolean
): string {
	const rect = trigger.getBoundingClientRect();
	const spaceBelow = window.innerHeight - rect.bottom;
	const spaceAbove = rect.top;
	const openUp = shouldOpenUp(spaceBelow, spaceAbove, listHeight);

	if (usePortal) {
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
