import { isNativeApp } from '$lib/platform/native';

const memory = new Map<string, string>();
/** Persist across reloads in browser / Playwright (Capacitor SecureStorage is native-only). */
const WEB_PREFIX = 'buhgalter.secure.';

/** e.g. `u10123.` — separates main app vs OEM dual-app clone in shared keystore. */
let namespacePrefix = '';
let namespaceReady: Promise<void> | null = null;

function webStorageKey(physicalKey: string): string {
	return WEB_PREFIX + physicalKey;
}

/** Call before first secure read/write in native APK (initAuthToken does this). */
export function initStorageNamespace(): Promise<void> {
	if (!namespaceReady) {
		namespaceReady = (async () => {
			if (!isNativeApp()) return;
			try {
				const { getAppStorageNamespace } = await import('$lib/platform/app-instance');
				const ns = await getAppStorageNamespace();
				namespacePrefix = ns ? `${ns}.` : '';
			} catch {
				namespacePrefix = '';
			}
		})();
	}
	return namespaceReady;
}

export function resetStorageNamespaceForTests(): void {
	namespacePrefix = '';
	namespaceReady = null;
}

function physicalKey(logicalKey: string): string {
	return namespacePrefix ? `${namespacePrefix}${logicalKey}` : logicalKey;
}

function webRead(physicalKey: string): string | null {
	const mem = memory.get(physicalKey);
	if (mem !== undefined) return mem;
	if (typeof localStorage === 'undefined') return null;
	try {
		return localStorage.getItem(webStorageKey(physicalKey));
	} catch {
		return null;
	}
}

function webWrite(physicalKey: string, value: string): void {
	memory.set(physicalKey, value);
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(webStorageKey(physicalKey), value);
	} catch {
		// ignore quota / private mode
	}
}

function webDelete(physicalKey: string): void {
	memory.delete(physicalKey);
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.removeItem(webStorageKey(physicalKey));
	} catch {
		// ignore
	}
}

async function nativeGet(key: string): Promise<string | null> {
	try {
		const { SecureStorage } = await import('@aparajita/capacitor-secure-storage');
		const value = await SecureStorage.get(key);
		return typeof value === 'string' ? value : null;
	} catch {
		return memory.get(key) ?? null;
	}
}

async function nativeSet(key: string, value: string): Promise<void> {
	try {
		const { SecureStorage } = await import('@aparajita/capacitor-secure-storage');
		await SecureStorage.set(key, value);
	} catch {
		memory.set(key, value);
	}
}

async function nativeRemove(key: string): Promise<void> {
	try {
		const { SecureStorage } = await import('@aparajita/capacitor-secure-storage');
		await SecureStorage.remove(key);
	} catch {
		memory.delete(key);
	}
}

async function readPhysical(physicalKey: string, legacyKey: string): Promise<string | null> {
	if (!isNativeApp()) {
		return webRead(physicalKey) ?? (legacyKey !== physicalKey ? webRead(legacyKey) : null);
	}
	const direct = await nativeGet(physicalKey);
	if (direct !== null) return direct;
	if (legacyKey === physicalKey) return null;
	const legacy = await nativeGet(legacyKey);
	if (legacy === null) return null;
	await nativeSet(physicalKey, legacy);
	await nativeRemove(legacyKey);
	return legacy;
}

async function writePhysical(physicalKey: string, legacyKey: string, value: string): Promise<void> {
	if (!isNativeApp()) {
		webWrite(physicalKey, value);
		if (legacyKey !== physicalKey) webDelete(legacyKey);
		return;
	}
	await nativeSet(physicalKey, value);
	if (legacyKey !== physicalKey) await nativeRemove(legacyKey);
}

async function removePhysical(physicalKey: string, legacyKey: string): Promise<void> {
	if (!isNativeApp()) {
		webDelete(physicalKey);
		if (legacyKey !== physicalKey) webDelete(legacyKey);
		return;
	}
	await nativeRemove(physicalKey);
	if (legacyKey !== physicalKey) await nativeRemove(legacyKey);
}

/**
 * Native secure storage; browser / e2e use memory + localStorage (never call Cap plugins —
 * unimplemented bridge calls can hang and leave the SPA on «Загрузка…»).
 */
export async function secureGet(key: string): Promise<string | null> {
	await initStorageNamespace();
	const pk = physicalKey(key);
	return readPhysical(pk, key);
}

export async function secureSet(key: string, value: string): Promise<void> {
	await initStorageNamespace();
	const pk = physicalKey(key);
	await writePhysical(pk, key, value);
}

export async function secureRemove(key: string): Promise<void> {
	await initStorageNamespace();
	const pk = physicalKey(key);
	await removePhysical(pk, key);
}

export function resetSecureStoreForTests(): void {
	memory.clear();
	resetStorageNamespaceForTests();
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
