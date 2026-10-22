import { get } from 'svelte/store';
import { _ } from 'svelte-i18n';
import { toast } from '$lib/toast';
import { isServerOfflineMode } from '$lib/offline/server-connectivity';

/**
 * Guard for actions that are intentionally online-only (not in outbox).
 * Shows a toast and returns false when the app is in offline mode.
 */
export function requireOnline(i18nKey = 'offline.onlineOnly'): boolean {
	if (!isServerOfflineMode()) return true;
	toast.error(get(_)(i18nKey));
	return false;
}

export class OnlineOnlyError extends Error {
	i18nKey: string;
	constructor(i18nKey = 'offline.onlineOnly') {
		super(i18nKey);
		this.name = 'OnlineOnlyError';
		this.i18nKey = i18nKey;
	}
}

/** Throw from non-UI paths (draft submit) so callers can toast via i18nKey. */
export function assertOnline(i18nKey = 'offline.onlineOnly'): void {
	if (!isServerOfflineMode()) return;
	throw new OnlineOnlyError(i18nKey);
}
