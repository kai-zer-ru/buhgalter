import { writable } from 'svelte/store';
import { tr } from '$lib/i18n';
import { secureGet, secureRemove, secureSet } from '$lib/platform/secure-store';

export const PIN_LENGTH = 4;

/** Preset timeouts for Настройки → Безопасность. */
export const BACKGROUND_LOCK_OPTIONS_MS = [
	30_000,
	60_000,
	5 * 60_000,
	10 * 60_000,
	15 * 60_000,
	30 * 60_000,
	60 * 60_000
] as const;

export type BackgroundLockMs = (typeof BACKGROUND_LOCK_OPTIONS_MS)[number];

/** Default background idle before lock (also legacy constant name). */
export const BACKGROUND_LOCK_MS: BackgroundLockMs = 60_000;

const KEY_ENABLED = 'app_lock.enabled';
const KEY_BIOMETRIC = 'app_lock.biometric_enabled';
const KEY_PIN = 'app_lock.pin_credentials';
const KEY_BACKGROUND_MS = 'app_lock.background_ms';
/** Legacy keys from short-lived trigger toggles — cleared on logout. */
const KEY_LOCK_ON_LAUNCH = 'app_lock.lock_on_launch';
const KEY_LOCK_ON_BACKGROUND = 'app_lock.lock_on_background';

export type AppLockConfig = {
	enabled: boolean;
	biometricEnabled: boolean;
	/** Idle in background before lock when returning to the app. */
	backgroundLockMs: BackgroundLockMs;
};

const DEFAULT_CONFIG: AppLockConfig = {
	enabled: false,
	biometricEnabled: false,
	backgroundLockMs: BACKGROUND_LOCK_MS
};

export type PinCredentials = {
	hash: string;
	salt: string;
};

let configCache: AppLockConfig | null = null;
let sessionUnlocked = false;
let backgroundAt: number | null = null;
let failedAttempts = 0;
let blockedUntil = 0;

export const appLockVisible = writable(false);

function syncLockScreenVisible(): void {
	appLockVisible.set(shouldShowLockScreen());
}

function bufferToBase64(buffer: ArrayBuffer | Uint8Array): string {
	const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
	let binary = '';
	for (const byte of bytes) binary += String.fromCharCode(byte);
	return btoa(binary);
}

function base64ToBytes(value: string): Uint8Array {
	const binary = atob(value);
	const bytes = new Uint8Array(binary.length);
	for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
	return bytes;
}

export function isValidPin(pin: string): boolean {
	return /^\d{4}$/.test(pin);
}

/** Common / trivial PINs rejected on setup and change (not on unlock). */
const BLOCKED_PINS = new Set([
	'0000',
	'1111',
	'2222',
	'3333',
	'4444',
	'5555',
	'6666',
	'7777',
	'8888',
	'9999',
	'1234',
	'4321',
	'0123',
	'3210',
	'1212',
	'2121',
	'1010',
	'0101',
	'1122',
	'2211',
	'6969'
]);

function isRepeatedPin(pin: string): boolean {
	return /^(\d)\1{3}$/.test(pin);
}

function isSequentialPin(pin: string): boolean {
	const digits = pin.split('').map((d) => Number(d));
	let ascending = true;
	let descending = true;
	for (let i = 1; i < digits.length; i++) {
		if (digits[i] !== (digits[i - 1] + 1) % 10) ascending = false;
		if (digits[i] !== (digits[i - 1] + 9) % 10) descending = false;
	}
	return ascending || descending;
}

export function isWeakPin(pin: string): boolean {
	if (!isValidPin(pin)) return false;
	return isRepeatedPin(pin) || isSequentialPin(pin) || BLOCKED_PINS.has(pin);
}

export type NewPinValidation = 'ok' | 'format' | 'weak';

export function validateNewPin(pin: string): NewPinValidation {
	if (!isValidPin(pin)) return 'format';
	if (isWeakPin(pin)) return 'weak';
	return 'ok';
}

export async function hashPin(pin: string, saltBytes?: Uint8Array): Promise<PinCredentials> {
	const salt = saltBytes ?? crypto.getRandomValues(new Uint8Array(16));
	const enc = new TextEncoder();
	const keyMaterial = await crypto.subtle.importKey('raw', enc.encode(pin), 'PBKDF2', false, [
		'deriveBits'
	]);
	const saltBuffer = salt.buffer.slice(
		salt.byteOffset,
		salt.byteOffset + salt.byteLength
	) as ArrayBuffer;
	const derived = await crypto.subtle.deriveBits(
		{
			name: 'PBKDF2',
			salt: saltBuffer,
			iterations: 100_000,
			hash: 'SHA-256'
		},
		keyMaterial,
		256
	);
	return {
		hash: bufferToBase64(derived),
		salt: bufferToBase64(salt)
	};
}

async function readPinCredentials(): Promise<PinCredentials | null> {
	const raw = await secureGet(KEY_PIN);
	if (!raw) return null;
	try {
		const parsed = JSON.parse(raw) as PinCredentials;
		if (typeof parsed.hash === 'string' && typeof parsed.salt === 'string') return parsed;
	} catch {
		// ignore
	}
	return null;
}

function parseBackgroundLockMs(raw: string | null): BackgroundLockMs {
	if (raw === null || raw === '') return BACKGROUND_LOCK_MS;
	const n = Number(raw);
	if (!Number.isFinite(n) || n < 0) return BACKGROUND_LOCK_MS;
	const exact = BACKGROUND_LOCK_OPTIONS_MS.find((ms) => ms === n);
	if (exact !== undefined) return exact;
	// Snap legacy / unknown values to the nearest preset.
	let best: BackgroundLockMs = BACKGROUND_LOCK_MS;
	let bestDelta = Number.POSITIVE_INFINITY;
	for (const ms of BACKGROUND_LOCK_OPTIONS_MS) {
		const delta = Math.abs(ms - n);
		if (delta < bestDelta) {
			best = ms;
			bestDelta = delta;
		}
	}
	return best;
}

export async function refreshAppLockConfig(): Promise<AppLockConfig> {
	const [enabledRaw, biometricRaw, timeoutRaw] = await Promise.all([
		secureGet(KEY_ENABLED),
		secureGet(KEY_BIOMETRIC),
		secureGet(KEY_BACKGROUND_MS)
	]);
	configCache = {
		enabled: enabledRaw === '1',
		biometricEnabled: biometricRaw === '1',
		backgroundLockMs: parseBackgroundLockMs(timeoutRaw)
	};
	syncLockScreenVisible();
	return configCache;
}

export function getAppLockConfig(): AppLockConfig {
	return configCache ?? { ...DEFAULT_CONFIG };
}

export async function isAppLockEnabled(): Promise<boolean> {
	if (!configCache) await refreshAppLockConfig();
	return configCache?.enabled ?? false;
}

export function isSessionUnlocked(): boolean {
	return sessionUnlocked;
}

export function shouldShowLockScreen(): boolean {
	const config = getAppLockConfig();
	return config.enabled && !sessionUnlocked;
}

export function unlockSession(): void {
	sessionUnlocked = true;
	failedAttempts = 0;
	blockedUntil = 0;
	backgroundAt = null;
	syncLockScreenVisible();
}

export function lockSession(): void {
	sessionUnlocked = false;
	syncLockScreenVisible();
}

export function retryBlockedMs(): number {
	return Math.max(0, blockedUntil - Date.now());
}

export async function verifyPin(pin: string): Promise<{ ok: boolean; retryAfterMs?: number }> {
	const waitMs = retryBlockedMs();
	if (waitMs > 0) return { ok: false, retryAfterMs: waitMs };

	if (!isValidPin(pin)) return { ok: false };

	const stored = await readPinCredentials();
	if (!stored) return { ok: false };

	const candidate = await hashPin(pin, base64ToBytes(stored.salt));
	if (candidate.hash !== stored.hash) {
		failedAttempts += 1;
		if (failedAttempts >= 3) {
			const exponent = Math.min(failedAttempts - 2, 5);
			const delayMs = Math.min(30_000, 1000 * 2 ** (exponent - 1));
			blockedUntil = Date.now() + delayMs;
			return { ok: false, retryAfterMs: delayMs };
		}
		return { ok: false };
	}

	unlockSession();
	return { ok: true };
}

export async function setPin(pin: string): Promise<void> {
	const validation = validateNewPin(pin);
	if (validation === 'format') throw new Error('INVALID_PIN');
	if (validation === 'weak') throw new Error('WEAK_PIN');
	const credentials = await hashPin(pin);
	await secureSet(KEY_PIN, JSON.stringify(credentials));
}

export async function enableAppLock(pin: string): Promise<void> {
	await setPin(pin);
	await secureSet(KEY_ENABLED, '1');
	await refreshAppLockConfig();
	lockSession();
	const { setWidgetLockEnabled } = await import('$lib/widgets/bridge');
	await setWidgetLockEnabled(true);
}

export async function disableAppLock(pin: string): Promise<boolean> {
	const verified = await verifyPin(pin);
	if (!verified.ok) return false;
	await clearAppLock();
	return true;
}

/** Removes PIN and biometric lock without verification — e.g. on logout. */
export async function clearAppLock(): Promise<void> {
	await Promise.all([
		secureRemove(KEY_ENABLED),
		secureRemove(KEY_BIOMETRIC),
		secureRemove(KEY_PIN),
		secureRemove(KEY_BACKGROUND_MS),
		secureRemove(KEY_LOCK_ON_LAUNCH),
		secureRemove(KEY_LOCK_ON_BACKGROUND)
	]);
	configCache = { ...DEFAULT_CONFIG };
	sessionUnlocked = false;
	failedAttempts = 0;
	blockedUntil = 0;
	backgroundAt = null;
	syncLockScreenVisible();
	const { setWidgetLockEnabled } = await import('$lib/widgets/bridge');
	await setWidgetLockEnabled(false);
}

export async function setBackgroundLockMs(ms: BackgroundLockMs): Promise<void> {
	await secureSet(KEY_BACKGROUND_MS, String(ms));
	await refreshAppLockConfig();
}

export async function changePin(currentPin: string, nextPin: string): Promise<boolean> {
	const verified = await verifyPin(currentPin);
	if (!verified.ok) return false;
	if (validateNewPin(nextPin) !== 'ok') return false;
	await setPin(nextPin);
	lockSession();
	return true;
}

export async function setBiometricEnabled(enabled: boolean): Promise<void> {
	if (enabled) {
		await secureSet(KEY_BIOMETRIC, '1');
	} else {
		await secureRemove(KEY_BIOMETRIC);
	}
	await refreshAppLockConfig();
}

export async function isBiometricAvailable(): Promise<boolean> {
	try {
		const { BiometricAuth } = await import('@aparajita/capacitor-biometric-auth');
		const result = await BiometricAuth.checkBiometry();
		return Boolean(result.isAvailable);
	} catch {
		return false;
	}
}

export async function verifyBiometric(
	reason: string,
	cancelTitle?: string,
	androidTitle?: string
): Promise<boolean> {
	try {
		const { BiometricAuth } = await import('@aparajita/capacitor-biometric-auth');
		await BiometricAuth.authenticate({
			reason,
			cancelTitle: cancelTitle ?? tr('common.cancel'),
			androidTitle: androidTitle ?? tr('appLock.biometricTitle'),
			allowDeviceCredential: true
		});
		unlockSession();
		return true;
	} catch {
		return false;
	}
}

export function noteAppBackground(): void {
	backgroundAt = Date.now();
}

export function noteAppForeground(): void {
	const config = getAppLockConfig();
	if (!config.enabled) return;
	if (!sessionUnlocked) return;
	if (backgroundAt === null) return;
	if (Date.now() - backgroundAt >= config.backgroundLockMs) {
		lockSession();
	}
	backgroundAt = null;
}

export function initAppLockListener(): () => void {
	let remove: (() => void) | undefined;

	void import('@capacitor/app').then(({ App }) => {
		void App.addListener('appStateChange', ({ isActive }) => {
			if (isActive) noteAppForeground();
			else noteAppBackground();
		}).then((handle) => {
			remove = () => void handle.remove();
		});
	});

	return () => remove?.();
}

export function resetAppLockForTests(): void {
	configCache = null;
	sessionUnlocked = false;
	backgroundAt = null;
	failedAttempts = 0;
	blockedUntil = 0;
	appLockVisible.set(false);
}

export function setAppLockConfigForTests(config: Partial<AppLockConfig>): void {
	configCache = { ...DEFAULT_CONFIG, ...config };
	syncLockScreenVisible();
}
