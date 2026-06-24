export type Theme = 'light' | 'dark';

export const THEME_COLORS: Record<Theme, string> = {
	light: '#f8fafc',
	dark: '#0f172a'
};

function syncThemeColor(theme: Theme) {
	const meta = document.querySelector('meta[name="theme-color"]');
	if (meta) {
		meta.setAttribute('content', THEME_COLORS[theme]);
	}
}

export function applyTheme(theme: Theme) {
	const root = document.documentElement;
	root.classList.toggle('dark', theme === 'dark');
	root.dataset.theme = theme;
	syncThemeColor(theme);
}
