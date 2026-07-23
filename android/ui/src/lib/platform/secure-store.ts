import { isNativeApp } from '$lib/platform/native';

const memory = new Map<string, string>();
/** Persist across reloads in browser / Playwright (Capacitor SecureStorage is native-only). */
const WEB_PREFIX = 'buhgalter.secure.';

function webRead(key: string): string | null {
	const mem = memory.get(key);
	if (mem !== undefined) return mem;
	if (typeof localStorage === 'undefined') return null;
	try {
		return localStorage.getItem(WEB_PREFIX + key);
	} catch {
		return null;
	}
}

function webWrite(key: string, value: string): void {
	memory.set(key, value);
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(WEB_PREFIX + key, value);
	} catch {
		// ignore quota / private mode
	}
}

function webDelete(key: string): void {
	memory.delete(key);
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.removeItem(WEB_PREFIX + key);
	} catch {
		// ignore
	}
}

/**
 * Native secure storage; browser / e2e use memory + localStorage (never call Cap plugins —
 * unimplemented bridge calls can hang and leave the SPA on «Загрузка…»).
 */
export async function secureGet(key: string): Promise<string | null> {
	if (!isNativeApp()) {
		return webRead(key);
	}
	try {
		const { SecureStorage } = await import('@aparajita/capacitor-secure-storage');
		const value = await SecureStorage.get(key);
		return typeof value === 'string' ? value : null;
	} catch {
		return memory.get(key) ?? null;
	}
}

export async function secureSet(key: string, value: string): Promise<void> {
	if (!isNativeApp()) {
		webWrite(key, value);
		return;
	}
	try {
		const { SecureStorage } = await import('@aparajita/capacitor-secure-storage');
		await SecureStorage.set(key, value);
	} catch {
		memory.set(key, value);
	}
}

export async function secureRemove(key: string): Promise<void> {
	if (!isNativeApp()) {
		webDelete(key);
		return;
	}
	try {
		const { SecureStorage } = await import('@aparajita/capacitor-secure-storage');
		await SecureStorage.remove(key);
	} catch {
		memory.delete(key);
	}
}

export function resetSecureStoreForTests(): void {
	memory.clear();
	if (typeof localStorage === 'undefined') return;
	try {
		const keys: string[] = [];
		for (let i = 0; i < localStorage.length; i++) {
			const k = localStorage.key(i);
			if (k?.startsWith(WEB_PREFIX)) keys.push(k);
		}
		for (const k of keys) localStorage.removeItem(k);
	} catch {
		// ignore
	}
}
