import { get } from 'svelte/store';
import {
	getNotificationInterceptBindings,
	putNotificationInterceptBindings,
	type NotificationInterceptBindings
} from '$lib/api/client';
import { user } from '$lib/stores/auth';
import type { BankBinding, InterceptSettings } from './types';

const STORAGE_PREFIX = 'buhgalter.notification_intercept.settings.v1:';

const DEFAULT_SETTINGS: InterceptSettings = {
	enabled: false,
	shadeNotifications: true,
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
		shadeNotifications: true,
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
			// Default on for existing installs (field missing → true).
			shadeNotifications: parsed.shadeNotifications !== false,
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

function bindingsPayload(settings: InterceptSettings): NotificationInterceptBindings {
	return {
		bank_bindings: settings.bankBindings.map((b) => ({
			package_name: b.packageName,
			bank_id: b.bankId,
			account_id: b.accountId
		})),
		card_bindings: settings.cardBindings.map((c) => ({
			bank_id: c.bankId,
			last4: c.last4,
			account_id: c.accountId
		}))
	};
}

function bindingsFromServer(
	remote: NotificationInterceptBindings
): Pick<InterceptSettings, 'bankBindings' | 'cardBindings'> {
	return {
		bankBindings: (remote.bank_bindings ?? []).map((b) => ({
			packageName: b.package_name,
			bankId: b.bank_id,
			accountId: b.account_id
		})),
		cardBindings: (remote.card_bindings ?? []).map((c) => ({
			bankId: c.bank_id,
			last4: normalizeLast4(c.last4),
			accountId: c.account_id
		}))
	};
}

function hasBindings(settings: Pick<InterceptSettings, 'bankBindings' | 'cardBindings'>): boolean {
	return settings.bankBindings.length > 0 || settings.cardBindings.length > 0;
}

function bankBindingKey(b: BankBinding): string {
	return b.accountId;
}

function cardBindingKey(c: { bankId: string; last4: string }): string {
	return `${c.bankId}:${c.last4}`;
}

/** Union of local and remote; same account / same card → local wins (active device). */
export function mergeInterceptBindings(
	local: Pick<InterceptSettings, 'bankBindings' | 'cardBindings'>,
	remote: Pick<InterceptSettings, 'bankBindings' | 'cardBindings'>
): Pick<InterceptSettings, 'bankBindings' | 'cardBindings'> {
	const bankMap = new Map<string, BankBinding>();
	for (const b of remote.bankBindings) bankMap.set(bankBindingKey(b), b);
	for (const b of local.bankBindings) bankMap.set(bankBindingKey(b), b);
	const cardMap = new Map<string, InterceptSettings['cardBindings'][number]>();
	for (const c of remote.cardBindings) cardMap.set(cardBindingKey(c), c);
	for (const c of local.cardBindings) cardMap.set(cardBindingKey(c), c);
	return {
		bankBindings: [...bankMap.values()],
		cardBindings: [...cardMap.values()]
	};
}

function bindingsPayloadEqual(
	a: NotificationInterceptBindings,
	b: NotificationInterceptBindings
): boolean {
	const norm = (p: NotificationInterceptBindings) =>
		JSON.stringify({
			bank_bindings: [...p.bank_bindings].sort((x, y) => x.account_id.localeCompare(y.account_id)),
			card_bindings: [...p.card_bindings].sort((x, y) =>
				`${x.bank_id}:${x.last4}`.localeCompare(`${y.bank_id}:${y.last4}`)
			)
		});
	return norm(a) === norm(b);
}

let syncFromServerInflight: Promise<void> | null = null;

/**
 * Two-way sync of bank/card bindings (not enabled/shade — device-only).
 * Empty server never wipes local; reinstall (empty local) pulls from server; both sides → merge + push if needed.
 */
export function syncInterceptSettingsFromServer(userId: string): Promise<void> {
	if (syncFromServerInflight) return syncFromServerInflight;
	syncFromServerInflight = (async () => {
		try {
			const remote = await getNotificationInterceptBindings();
			const remoteBindings = bindingsFromServer(remote);
			const local = loadInterceptSettings(userId);
			const localBindings = {
				bankBindings: local.bankBindings,
				cardBindings: local.cardBindings
			};
			const remoteHas = hasBindings(remoteBindings);
			const localHas = hasBindings(localBindings);

			if (!remoteHas && !localHas) return;

			if (!remoteHas && localHas) {
				await putNotificationInterceptBindings(bindingsPayload(local));
				return;
			}

			if (remoteHas && !localHas) {
				saveInterceptSettings(userId, { ...local, ...remoteBindings }, { skipServer: true });
				return;
			}

			const merged = mergeInterceptBindings(localBindings, remoteBindings);
			const mergedSettings: InterceptSettings = { ...local, ...merged };
			saveInterceptSettings(userId, mergedSettings, { skipServer: true });
			const mergedPayload = bindingsPayload(mergedSettings);
			if (!bindingsPayloadEqual(mergedPayload, remote)) {
				await putNotificationInterceptBindings(mergedPayload);
			}
		} catch {
			// offline or old server — keep local bindings
		}
	})().finally(() => {
		syncFromServerInflight = null;
	});
	return syncFromServerInflight;
}

export function saveInterceptSettings(
	userId: string,
	settings: InterceptSettings,
	opts?: { skipServer?: boolean }
): void {
	const normalized: InterceptSettings = {
		enabled: Boolean(settings.enabled),
		shadeNotifications: settings.shadeNotifications !== false,
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
	if (!opts?.skipServer) {
		void putNotificationInterceptBindings(bindingsPayload(normalized)).catch(() => undefined);
	}
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
