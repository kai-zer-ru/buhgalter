import type { HouseholdMe } from '$lib/api/client';
import { clearRefCache } from '$lib/offline/ref-cache';
import { resetLedgerLocalState } from '$lib/offline/local-state';
import { dataRefreshTick } from '$lib/offline/sync';

const FINGERPRINT_KEY = 'buhgalter.household.fingerprint';

let lastFingerprint: string | null | undefined;

export function membershipFingerprint(me: HouseholdMe | null | undefined): string {
	if (!me?.household) return '';
	const members = [...(me.members ?? [])]
		.map((m) => m.user_id)
		.sort()
		.join(',');
	return `${me.household.id}:${me.role ?? ''}:${members}`;
}

function readStoredFingerprint(): string | null {
	if (lastFingerprint !== undefined) return lastFingerprint;
	try {
		lastFingerprint = localStorage.getItem(FINGERPRINT_KEY);
	} catch {
		lastFingerprint = null;
	}
	return lastFingerprint;
}

function writeStoredFingerprint(value: string): void {
	lastFingerprint = value;
	try {
		localStorage.setItem(FINGERPRINT_KEY, value);
	} catch {
		/* ignore quota / private mode */
	}
}

/** Drop dictionaries after create/accept/leave/kick/transfer/dissolve/share/unshare. */
export function resetCachesAfterMembershipChange(): void {
	clearRefCache();
	resetLedgerLocalState();
	dataRefreshTick.update((n) => n + 1);
	void import('$lib/widgets/publish').then((m) => m.scheduleWidgetSnapshotPublish());
}

/** Compare GET /households/me; reset caches when the group or members changed. */
export function applyHouseholdMembershipSnapshot(me: HouseholdMe): void {
	const next = membershipFingerprint(me);
	const prev = readStoredFingerprint();
	if (prev !== null && prev !== next) {
		resetCachesAfterMembershipChange();
	}
	writeStoredFingerprint(next);
}

export function clearHouseholdMembershipFingerprint(): void {
	lastFingerprint = null;
	try {
		localStorage.removeItem(FINGERPRINT_KEY);
	} catch {
		/* ignore */
	}
}

export function resetMembershipCacheForTests(): void {
	lastFingerprint = undefined;
	try {
		localStorage.removeItem(FINGERPRINT_KEY);
	} catch {
		/* ignore */
	}
}
