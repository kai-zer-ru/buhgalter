import { getServerUrl } from '$lib/platform/server-url';
import {
	secureGet,
	secureRemove,
	secureSet,
	initStorageNamespace
} from '$lib/platform/secure-store';

const TOKEN_KEY = 'buhgalter.auth_token';
const ORIGIN_KEY = 'buhgalter.auth_server_origin';
const KIND_KEY = 'buhgalter.auth_kind';

export type AuthTokenKind = 'session' | 'api_token';

let memoryToken = '';
let memoryKind: AuthTokenKind = 'api_token';
let memoryAuthOrigin = '';
let initPromise: Promise<void> | null = null;

function readLegacyLocalStorage(): string {
	if (typeof localStorage === 'undefined') return '';
	try {
		return localStorage.getItem(TOKEN_KEY) ?? '';
	} catch {
		return '';
	}
}

function removeLegacyLocalStorage(): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.removeItem(TOKEN_KEY);
	} catch {
		// ignore
	}
}

function readKind(): AuthTokenKind {
	if (typeof localStorage === 'undefined') return 'api_token';
	try {
		const v = localStorage.getItem(KIND_KEY);
		return v === 'session' ? 'session' : 'api_token';
	} catch {
		return 'api_token';
	}
}

function writeKind(kind: AuthTokenKind): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(KIND_KEY, kind);
	} catch {
		// ignore
	}
}

function removeKind(): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.removeItem(KIND_KEY);
	} catch {
		// ignore
	}
}

async function loadToken(): Promise<void> {
	await initStorageNamespace();
	const secure = await secureGet(TOKEN_KEY);
	const origin = await secureGet(ORIGIN_KEY);
	if (origin) memoryAuthOrigin = origin;
	if (secure) {
		memoryToken = secure;
		memoryKind = readKind();
		removeLegacyLocalStorage();
		return;
	}
	const legacy = readLegacyLocalStorage().trim();
	if (legacy) {
		memoryToken = legacy;
		memoryKind = readKind();
		writeKind(memoryKind);
		await secureSet(TOKEN_KEY, legacy);
		removeLegacyLocalStorage();
	}
}

/** Load token from secure storage (migrates legacy localStorage once). Call before auth bootstrap. */
export function initAuthToken(): Promise<void> {
	if (!initPromise) {
		initPromise = loadToken();
	}
	return initPromise;
}

export function getAuthToken(): string {
	return memoryToken;
}

export function getAuthTokenKind(): AuthTokenKind {
	return memoryKind;
}

/** Server origin (LAN or remote URL) this token was issued for. */
export function getAuthServerOrigin(): string {
	return memoryAuthOrigin;
}

export async function setAuthToken(
	token: string,
	kind: AuthTokenKind = 'api_token'
): Promise<void> {
	const trimmed = token.trim();
	memoryToken = trimmed;
	memoryKind = kind;
	const origin = getServerUrl();
	memoryAuthOrigin = origin;
	if (trimmed) {
		await secureSet(TOKEN_KEY, trimmed);
		if (origin) await secureSet(ORIGIN_KEY, origin);
		writeKind(kind);
	} else {
		await secureRemove(TOKEN_KEY);
		await secureRemove(ORIGIN_KEY);
		memoryAuthOrigin = '';
		removeKind();
	}
	removeLegacyLocalStorage();
}

export async function clearAuthToken(): Promise<void> {
	memoryToken = '';
	memoryKind = 'api_token';
	memoryAuthOrigin = '';
	await secureRemove(TOKEN_KEY);
	await secureRemove(ORIGIN_KEY);
	removeKind();
	removeLegacyLocalStorage();
}

export function authHeaders(): Record<string, string> {
	const token = getAuthToken();
	if (!token) return {};
	return { Authorization: `Bearer ${token}` };
}

export function resetAuthTokenForTests(): void {
	memoryToken = '';
	memoryKind = 'api_token';
	memoryAuthOrigin = '';
	initPromise = null;
	removeLegacyLocalStorage();
	removeKind();
}
