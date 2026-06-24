import { browser } from '$app/environment';
import { get } from 'svelte/store';
import { _, init, register, locale } from 'svelte-i18n';

register('ru', () => import('./ru.json'));
register('en', () => import('./en.json'));

function getInitialLocale(): string {
	if (browser) {
		const stored = localStorage.getItem('locale');
		if (stored === 'en' || stored === 'ru') return stored;
		const nav = navigator.language ?? '';
		return nav.startsWith('en') ? 'en' : 'ru';
	}
	return 'ru';
}

init({
	fallbackLocale: 'ru',
	initialLocale: getInitialLocale()
});

export function setLocale(lang: string) {
	const code = lang === 'en' ? 'en' : 'ru';
	if (browser) {
		localStorage.setItem('locale', code);
	}
	locale.set(code);
}

type TrValues = Record<string, string | number | boolean | Date | null | undefined>;

/** Sync translate for script / $derived.by. In templates use $_(key). */
export function tr(key: string, options?: { values?: TrValues }): string {
	const format = get(_);
	return typeof format === 'function' ? (format(key, options) as string) : key;
}
