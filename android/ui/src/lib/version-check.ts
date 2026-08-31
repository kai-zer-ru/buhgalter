import { getVersionCheck, type VersionCheckResult } from '$lib/api/client';
import { writable } from 'svelte/store';

const LAST_CHECK_KEY = 'buhgalter.versionCheckLastAt';
const LAST_RESULT_KEY = 'buhgalter.versionCheckLastResult.v1';
const DISMISSED_VERSION_KEY = 'buhgalter.versionCheckDismissedVersion';
/** Network version check at most once per day (app vs server). */
export const VERSION_CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000;
const GITHUB_REPO = 'kai-zer-ru/buhgalter';

export type PendingVersionUpdate = VersionCheckResult;

/** Version info for the app vs connected server instance. */
export type AppVersionInfo = {
	appVersion: string;
	serverVersion: string | null;
	releaseUrl: string | null;
	/** App is older than server (`app < server`), any semver part. */
	versionMismatch: boolean;
	/** Full-screen block: major or minor behind; patch-only behind does not block. */
	versionBlocked: boolean;
};

/** Full-screen block when app version is behind server. */
export const versionBlockInfo = writable<AppVersionInfo | null>(null);

/** @deprecated Use AppVersionInfo */
export type PendingAppUpdate = AppVersionInfo & { updateNeeded: boolean };

let memoryResult: AppVersionInfo | null = null;
let memoryCheckedAt = 0;

function shouldCheckNow(): boolean {
	if (memoryCheckedAt > 0 && Date.now() - memoryCheckedAt < VERSION_CHECK_INTERVAL_MS) {
		return false;
	}
	if (typeof localStorage === 'undefined') return true;
	const raw = localStorage.getItem(LAST_CHECK_KEY);
	if (!raw) return true;
	const lastCheck = Number(raw);
	if (!Number.isFinite(lastCheck)) return true;
	if (Date.now() - lastCheck < VERSION_CHECK_INTERVAL_MS) {
		memoryCheckedAt = lastCheck;
		return false;
	}
	return true;
}

function markCheckedNow(): void {
	memoryCheckedAt = Date.now();
	if (typeof localStorage === 'undefined') return;
	localStorage.setItem(LAST_CHECK_KEY, String(memoryCheckedAt));
}

function readCachedResult(app: string): AppVersionInfo | null {
	if (memoryResult && memoryResult.appVersion === app) return memoryResult;
	if (typeof localStorage === 'undefined') return null;
	try {
		const raw = localStorage.getItem(LAST_RESULT_KEY);
		if (!raw) return null;
		const parsed = JSON.parse(raw) as AppVersionInfo;
		if (!parsed || typeof parsed.appVersion !== 'string') return null;
		if (parsed.appVersion !== app) return null;
		memoryResult = parsed;
		return parsed;
	} catch {
		return null;
	}
}

function writeCachedResult(info: AppVersionInfo): void {
	memoryResult = info;
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(LAST_RESULT_KEY, JSON.stringify(info));
	} catch {
		// quota
	}
}

export function dismissVersionUpdate(version: string): void {
	localStorage.setItem(DISMISSED_VERSION_KEY, version);
}

export function normalizeVersion(v: string): string {
	return v.trim().replace(/^v/i, '');
}

export function versionParts(v: string): number[] {
	return normalizeVersion(v)
		.split('.')
		.map((part) => {
			const core = part.split('-')[0]?.split('+')[0] ?? '';
			const n = Number.parseInt(core, 10);
			return Number.isFinite(n) ? n : 0;
		});
}

/** Semver-style compare (major.minor.patch parts). Returns -1, 0, or 1. */
export function compareVersions(a: string, b: string): number {
	const aParts = versionParts(a);
	const bParts = versionParts(b);
	const maxLen = Math.max(aParts.length, bParts.length);
	for (let i = 0; i < maxLen; i++) {
		const ap = aParts[i] ?? 0;
		const bp = bParts[i] ?? 0;
		if (ap < bp) return -1;
		if (ap > bp) return 1;
	}
	return 0;
}

export function releaseUrlForVersion(version: string): string {
	const v = normalizeVersion(version);
	return `https://github.com/${GITHUB_REPO}/releases/tag/v${v}`;
}

/** True when app is behind at major or minor level (patch-only behind does not block). */
export function versionBehindBlocks(appVersion: string, serverVersion: string): boolean {
	if (compareVersions(appVersion, serverVersion) >= 0) return false;
	const [appMajor, appMinor] = versionParts(appVersion);
	const [serverMajor, serverMinor] = versionParts(serverVersion);
	if (appMajor < serverMajor) return true;
	return appMajor === serverMajor && appMinor < serverMinor;
}

/** True when app should be fully blocked (major or minor behind). */
export function isBlockingVersionMismatch(info: AppVersionInfo): boolean {
	if (!info.serverVersion) return false;
	return info.versionBlocked;
}

export function applyVersionBlock(info: AppVersionInfo): void {
	versionBlockInfo.set(isBlockingVersionMismatch(info) ? info : null);
}

export function clearVersionBlock(): void {
	versionBlockInfo.set(null);
}

function buildAppVersionInfo(app: string, server: string | null): AppVersionInfo {
	const mismatch = server ? versionsMismatch(app, server) : false;
	const blocked = server ? versionBehindBlocks(app, server) : false;
	return {
		appVersion: app,
		serverVersion: server,
		releaseUrl: server ? releaseUrlForVersion(server) : null,
		versionMismatch: mismatch,
		versionBlocked: blocked
	};
}

/** True when app version is older than server (`app < server`). */
export function versionsMismatch(appVersion: string, serverVersion: string): boolean {
	return compareVersions(appVersion, serverVersion) < 0;
}

export type FetchAppVersionOptions = {
	/** Bypass 24h throttle (e.g. explicit user action). */
	force?: boolean;
};

/**
 * Compare APK vs server version. Network at most once per 24h unless `force`.
 * Within the interval returns last successful result (or offline shape).
 */
export async function fetchAppVersionInfo(
	appVersion: string,
	opts: FetchAppVersionOptions = {}
): Promise<AppVersionInfo> {
	const app = normalizeVersion(appVersion);
	if (!opts.force && !shouldCheckNow()) {
		return readCachedResult(app) ?? buildAppVersionInfo(app, null);
	}

	try {
		const result = await getVersionCheck();
		markCheckedNow();
		const rawServer = result.current_version?.trim();
		const info = !rawServer
			? buildAppVersionInfo(app, null)
			: buildAppVersionInfo(app, normalizeVersion(rawServer));
		writeCachedResult(info);
		return info;
	} catch {
		// Do not mark success timestamp on failure — retry next launch.
		return readCachedResult(app) ?? buildAppVersionInfo(app, null);
	}
}

/** @deprecated Use fetchAppVersionInfo */
export async function checkAppBehindServer(appVersion: string): Promise<AppVersionInfo> {
	return fetchAppVersionInfo(appVersion);
}

export async function checkForVersionUpdate(): Promise<PendingVersionUpdate | null> {
	if (!shouldCheckNow()) {
		return null;
	}

	try {
		const result = await getVersionCheck();
		markCheckedNow();
		if (!result.update_available || !result.latest_version) {
			return null;
		}
		const dismissed = localStorage.getItem(DISMISSED_VERSION_KEY);
		if (dismissed === result.latest_version) {
			return null;
		}
		return result;
	} catch {
		return null;
	}
}

export function resetVersionCheckForTests(): void {
	memoryResult = null;
	memoryCheckedAt = 0;
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.removeItem(LAST_CHECK_KEY);
		localStorage.removeItem(LAST_RESULT_KEY);
		localStorage.removeItem(DISMISSED_VERSION_KEY);
	} catch {
		// ignore
	}
}
