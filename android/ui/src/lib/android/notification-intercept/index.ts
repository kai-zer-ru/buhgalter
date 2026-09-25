import { get } from 'svelte/store';
import { _ } from 'svelte-i18n';
import { isNativeApp } from '$lib/platform/native';
import { toast } from '$lib/toast';
import { user } from '$lib/stores/auth';
import {
	addDraftsChangedListener,
	addPendingAvailableListener,
	finishQuietWake,
	isQuietWake
} from './plugin';
import {
	processPendingBankNotificationsDetailed,
	syncDraftNotifyFromNative,
	syncInterceptNativeFromSettings
} from './process';

export * from './types';
export * from './banks';
export * from './settings';
export * from './drafts';
export * from './draft-transfer';
export * from './draft-notify';
export * from './prefill';
export * from './plugin';
export * from './process';
export * from './history-local';
export * from './history-create';
export { parseBankNotification } from './parsers';
export { resolveAccountId } from './account-resolve';
export { matchMerchant } from './merchant-match';
export { pickCategorySuggestion, suggestCategoryFromMerchant } from './category-suggest';

/**
 * Wire native pending queue → drafts. Call once from root layout.
 * Returns cleanup.
 */
export async function initNotificationIntercept(): Promise<() => void> {
	if (!isNativeApp()) {
		return () => undefined;
	}

	const run = () => {
		if (!get(user)?.id) return;
		void (async () => {
			const quiet = await isQuietWake();
			await syncDraftNotifyFromNative();
			await syncInterceptNativeFromSettings();
			const { added, cancelled } = await processPendingBankNotificationsDetailed();
			if (!quiet) {
				if (added > 0) {
					toast(get(_)('bankNotifications.drafts.toastNew', { values: { n: added } }));
				}
				if (cancelled > 0) {
					toast(get(_)('bankNotifications.drafts.toastCancelled', { values: { n: cancelled } }));
				}
			}
			if (quiet) {
				await finishQuietWake();
			}
		})();
	};

	run();

	const removePending = await addPendingAvailableListener(run);
	const removeDrafts = await addDraftsChangedListener(run);

	let removeResume: (() => void) | undefined;
	try {
		const { App } = await import('@capacitor/app');
		const handle = await App.addListener('appStateChange', ({ isActive }) => {
			if (isActive) run();
		});
		removeResume = () => void handle.remove();
	} catch {
		removeResume = undefined;
	}

	const unsubUser = user.subscribe(() => {
		run();
	});

	return () => {
		removePending();
		removeDrafts();
		removeResume?.();
		unsubUser();
	};
}
