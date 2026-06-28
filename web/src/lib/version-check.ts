import { getVersionCheck, type VersionCheckResult } from '$lib/api/client';

const LAST_CHECK_KEY = 'buhgalter.versionCheckLastAt';
const DISMISSED_VERSION_KEY = 'buhgalter.versionCheckDismissedVersion';
const CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000;

export type PendingVersionUpdate = VersionCheckResult;

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
