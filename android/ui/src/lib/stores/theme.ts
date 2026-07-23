import { applyThemePreference, isThemePreference, type ThemePreference } from '$lib/theme';

export function initTheme(): ThemePreference {
	const stored = localStorage.getItem('theme');
	const preference = stored && isThemePreference(stored) ? stored : 'system';
	applyThemePreference(preference);
	return preference;
}

export function syncThemeFromUser(theme: string) {
	if (!isThemePreference(theme)) {
		return;
	}
	localStorage.setItem('theme', theme);
	applyThemePreference(theme);
}
