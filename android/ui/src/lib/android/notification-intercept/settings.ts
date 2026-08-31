import { get } from 'svelte/store';
import { user } from '$lib/stores/auth';
import type { BankBinding, InterceptSettings } from './types';

const STORAGE_PREFIX = 'buhgalter.notification_intercept.settings.v1:';

const DEFAULT_SETTINGS: InterceptSettings = {
	enabled: false,
	bankBindings: [],
	cardBindings: []
};

/**
 * Bind account → bank. Several accounts may share the same bank
 * (disambiguation for drafts is last4 → account, else first bank match).
 */
export function setAccountBankBinding(
	bankBindings: BankBinding[],
	accountId: string,
	opts: { bankId: string; packageName: string } | null
): BankBinding[] {
	const withoutAccount = bankBindings.filter((b) => b.accountId !== accountId);
	if (!opts) return withoutAccount;
	return [...withoutAccount, { bankId: opts.bankId, packageName: opts.packageName, accountId }];
}

function storageKey(userId: string): string {
	return `${STORAGE_PREFIX}${userId}`;
}

function normalizeLast4(value: string): string {
	return value.replace(/\D/g, '').slice(-4);
}

export function emptyInterceptSettings(): InterceptSettings {
	return {
		enabled: false,
		bankBindings: [],
		cardBindings: []
	};
}

const memorySettings = new Map<string, string>();

function readStorage(key: string): string | null {
	try {
		if (typeof localStorage !== 'undefined') {
			return localStorage.getItem(key);
		}
	} catch {
		// ignore
	}
	return memorySettings.get(key) ?? null;
}

function writeStorage(key: string, value: string): void {
	memorySettings.set(key, value);
	try {
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem(key, value);
		}
	} catch {
		// ignore
	}
}

function removeStorage(key: string): void {
	memorySettings.delete(key);
	try {
		if (typeof localStorage !== 'undefined') {
			localStorage.removeItem(key);
		}
	} catch {
		// ignore
	}
}

export function loadInterceptSettings(userId: string | null | undefined): InterceptSettings {
	if (!userId) return emptyInterceptSettings();
	try {
		const raw = readStorage(storageKey(userId));
		if (!raw) return emptyInterceptSettings();
		const parsed = JSON.parse(raw) as Partial<InterceptSettings>;
		return {
			enabled: Boolean(parsed.enabled),
			bankBindings: Array.isArray(parsed.bankBindings) ? parsed.bankBindings : [],
			cardBindings: Array.isArray(parsed.cardBindings)
				? parsed.cardBindings.map((c) => ({
						...c,
						last4: normalizeLast4(c.last4)
					}))
				: []
		};
	} catch {
		return emptyInterceptSettings();
	}
}

export function saveInterceptSettings(userId: string, settings: InterceptSettings): void {
	const normalized: InterceptSettings = {
		enabled: Boolean(settings.enabled),
		bankBindings: settings.bankBindings.map((b) => ({
			packageName: b.packageName,
			bankId: b.bankId,
			accountId: b.accountId
		})),
		cardBindings: settings.cardBindings
			.map((c) => ({
				bankId: c.bankId,
				last4: normalizeLast4(c.last4),
				accountId: c.accountId
			}))
			.filter((c) => c.last4.length === 4 && c.accountId)
	};
	writeStorage(storageKey(userId), JSON.stringify(normalized));
}

/** Test helper. */
export function clearInterceptSettingsForTests(userId?: string): void {
	if (userId) {
		removeStorage(storageKey(userId));
		return;
	}
	memorySettings.clear();
}

/** Settings for the currently logged-in user. */
export function getCurrentInterceptSettings(): InterceptSettings {
	const u = get(user);
	return loadInterceptSettings(u?.id);
}

export function updateCurrentInterceptSettings(
	patch: Partial<InterceptSettings> | ((prev: InterceptSettings) => InterceptSettings)
): InterceptSettings | null {
	const u = get(user);
	if (!u?.id) return null;
	const prev = loadInterceptSettings(u.id);
	const next = typeof patch === 'function' ? patch(prev) : { ...prev, ...patch };
	saveInterceptSettings(u.id, next);
	return next;
}

export { DEFAULT_SETTINGS, normalizeLast4 };
