import { syncSystemBars } from '$lib/platform/system-bars';

export type ThemePreference = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

export type Theme = ResolvedTheme;

export const THEME_COLORS: Record<ResolvedTheme, string> = {
	light: '#f8fafc',
	dark: '#0f172a'
};

export function isThemePreference(value: string): value is ThemePreference {
	return value === 'light' || value === 'dark' || value === 'system';
}

export function resolveTheme(preference: ThemePreference): ResolvedTheme {
	if (preference === 'light' || preference === 'dark') {
		return preference;
	}
	if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches) {
		return 'dark';
	}
	return 'light';
}

function syncThemeColor(theme: ResolvedTheme) {
	const meta = document.querySelector('meta[name="theme-color"]');
	if (meta) {
		meta.setAttribute('content', THEME_COLORS[theme]);
	}
}

export function applyTheme(theme: ResolvedTheme) {
	const root = document.documentElement;
	root.classList.toggle('dark', theme === 'dark');
	root.dataset.theme = theme;
	syncThemeColor(theme);
	void syncSystemBars(theme);
}

let mediaQuery: MediaQueryList | null = null;
let mediaListener: ((event: MediaQueryListEvent) => void) | null = null;

function clearSystemListener() {
	if (mediaQuery && mediaListener) {
		mediaQuery.removeEventListener('change', mediaListener);
	}
	mediaQuery = null;
	mediaListener = null;
}

export function applyThemePreference(preference: ThemePreference) {
	clearSystemListener();
	applyTheme(resolveTheme(preference));
	if (preference !== 'system' || typeof window === 'undefined') {
		return;
	}
	mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
	mediaListener = () => {
		applyTheme(resolveTheme('system'));
	};
	mediaQuery.addEventListener('change', mediaListener);
}
