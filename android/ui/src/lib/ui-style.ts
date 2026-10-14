export type UiStyle = 'classic' | 'material';

export const UI_STYLE_STORAGE_KEY = 'ui_style';

export function isUiStyle(value: string): value is UiStyle {
	return value === 'classic' || value === 'material';
}

export function applyUiStyle(style: UiStyle) {
	if (typeof document === 'undefined') return;
	document.documentElement.dataset.uiStyle = style;
}

export function readUiStyle(): UiStyle {
	if (typeof localStorage === 'undefined') return 'classic';
	const stored = localStorage.getItem(UI_STYLE_STORAGE_KEY);
	return stored && isUiStyle(stored) ? stored : 'classic';
}

/** Persist and apply. Orthogonal to light|dark|system theme. */
export function setUiStyle(style: UiStyle) {
	if (typeof localStorage !== 'undefined') {
		localStorage.setItem(UI_STYLE_STORAGE_KEY, style);
	}
	applyUiStyle(style);
}

export function initUiStyle(): UiStyle {
	const style = readUiStyle();
	applyUiStyle(style);
	return style;
}
