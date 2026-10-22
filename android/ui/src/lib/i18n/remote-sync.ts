import { browser } from '$app/environment';
import { addMessages, waitLocale } from 'svelte-i18n';
import { getUIi18n } from '$lib/api/client';
import { versionsMismatch } from '$lib/version-check';
import bundledRu from './ru.json';

const CACHE_PREFIX = 'buhgalter.remote_i18n.v2.';

/** Reject truncated server catalogs (e.g. web-only export) — keep bundled Android strings. */
const MIN_REMOTE_I18N_KEYS = Math.floor(Object.keys(bundledRu).length * 0.85);

function remoteCatalogUsable(messages: Record<string, string>): boolean {
	return Object.keys(messages).length >= MIN_REMOTE_I18N_KEYS;
}

function normalizeLang(lang: string): 'ru' | 'en' {
	return lang === 'en' ? 'en' : 'ru';
}

function cacheKey(serverVersion: string, lang: string): string {
	return `${CACHE_PREFIX}${serverVersion}.${lang}`;
}

function readCachedMessages(serverVersion: string, lang: string): Record<string, string> | null {
	if (!browser) return null;
	try {
		const raw = localStorage.getItem(cacheKey(serverVersion, lang));
		if (!raw) return null;
		const parsed = JSON.parse(raw) as unknown;
		if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null;
		return parsed as Record<string, string>;
	} catch {
		return null;
	}
}

function writeCachedMessages(
	serverVersion: string,
	lang: string,
	messages: Record<string, string>
): void {
	if (!browser) return;
	try {
		localStorage.setItem(cacheKey(serverVersion, lang), JSON.stringify(messages));
	} catch {
		// quota / private mode — ignore
	}
}

/**
 * When the APK is older than the server, fetch the server UI catalog and merge
 * over bundled svelte-i18n messages (new/changed keys for older builds).
 * Soft-fails offline. Cache keyed by server version + language.
 */
export async function syncRemoteI18nOnMismatch(
	appVersion: string,
	serverVersion: string | null,
	lang: string
): Promise<boolean> {
	if (!browser || !serverVersion || !versionsMismatch(appVersion, serverVersion)) {
		return false;
	}

	const code = normalizeLang(lang);
	await waitLocale(code);

	const cached = readCachedMessages(serverVersion, code);
	if (cached) {
		if (!remoteCatalogUsable(cached)) {
			if (browser) {
				try {
					localStorage.removeItem(cacheKey(serverVersion, code));
				} catch {
					// ignore
				}
			}
		} else {
			addMessages(code, cached);
			return true;
		}
	}

	try {
		const res = await getUIi18n(code);
		const messages = res.messages as Record<string, string>;
		if (!messages || typeof messages !== 'object' || !remoteCatalogUsable(messages)) {
			return false;
		}
		addMessages(code, messages);
		writeCachedMessages(serverVersion, code, messages);
		return true;
	} catch {
		return false;
	}
}
