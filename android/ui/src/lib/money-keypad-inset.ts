const CSS_VAR = '--money-keypad-inset';

/** Reserve space above the fixed money keypad so forms stay scrollable. */
export function setMoneyKeypadInset(px: number): void {
	if (typeof document === 'undefined') return;
	document.documentElement.style.setProperty(CSS_VAR, `${Math.max(0, Math.round(px))}px`);
}

export function clearMoneyKeypadInset(): void {
	if (typeof document === 'undefined') return;
	document.documentElement.style.setProperty(CSS_VAR, '0px');
}
