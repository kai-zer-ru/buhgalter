import { getVersionCheck, type VersionCheckResult } from '$lib/api/client';
import { writable } from 'svelte/store';

const LAST_CHECK_KEY = 'buhgalter.versionCheckLastAt';
const DISMISSED_VERSION_KEY = 'buhgalter.versionCheckDismissedVersion';
const CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000;
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

function shouldCheckNow(): boolean {
	const raw = localStorage.getItem(LAST_CHECK_KEY);
	if (!raw) return true;
	const lastCheck = Number(raw);
	if (!Number.isFinite(lastCheck)) return true;
	return Date.now() - lastCheck >= CHECK_INTERVAL_MS;
}

function markCheckedNow(): void {
	localStorage.setItem(LAST_CHECK_KEY, String(Date.now()));
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

export async function fetchAppVersionInfo(appVersion: string): Promise<AppVersionInfo> {
	const app = normalizeVersion(appVersion);
	try {
		const result = await getVersionCheck();
		const rawServer = result.current_version?.trim();
		if (!rawServer) {
			return buildAppVersionInfo(app, null);
		}
		const server = normalizeVersion(rawServer);
		return buildAppVersionInfo(app, server);
	} catch {
		return buildAppVersionInfo(app, null);
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
		markCheckedNow();
		return null;
	}
}
