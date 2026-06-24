export function dropdownListStyle(
	trigger: HTMLElement,
	listHeight: number,
	usePortal: boolean
): string {
	const rect = trigger.getBoundingClientRect();
	const spaceBelow = window.innerHeight - rect.bottom;
	const spaceAbove = rect.top;
	const openUp = spaceBelow < listHeight + 8 && spaceAbove > spaceBelow;

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
