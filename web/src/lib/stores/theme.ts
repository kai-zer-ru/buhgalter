import { applyTheme } from '$lib/theme';

export function initTheme() {
	const stored = localStorage.getItem('theme');
	const theme = stored === 'dark' || stored === 'light' ? stored : 'light';
	applyTheme(theme);
	return theme;
}

export function syncThemeFromUser(theme: string) {
	if (theme === 'dark' || theme === 'light') {
		localStorage.setItem('theme', theme);
		applyTheme(theme);
	}
}
