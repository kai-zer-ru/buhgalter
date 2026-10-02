import { registerPlugin } from '@capacitor/core';

type AppInstancePlugin = {
	getStorageNamespace(): Promise<{ namespace: string }>;
	hideSoftKeyboard(): Promise<void>;
	setPinLockEnabled?(options: { enabled: boolean }): Promise<void>;
};

const plugin = registerPlugin<AppInstancePlugin>('AppInstance', {
	web: () => import('$lib/platform/app-instance.web').then((m) => new m.AppInstanceWeb())
});

/** Unique per Android install (main app vs OEM clone). Empty in browser. */
export async function getAppStorageNamespace(): Promise<string> {
	try {
		const { namespace } = await plugin.getStorageNamespace();
		return typeof namespace === 'string' ? namespace.trim() : '';
	} catch {
		return '';
	}
}

/** Hide the system soft keyboard without blurring the focused field. */
export async function hideSoftKeyboard(): Promise<void> {
	try {
		await plugin.hideSoftKeyboard();
	} catch {
		// no-op in browser / older APK
	}
}

/** Native PIN-on flag for intercept quiet-wake (must not use widget hide-amounts). */
export async function setNativePinLockEnabled(enabled: boolean): Promise<void> {
	try {
		await plugin.setPinLockEnabled?.({ enabled });
	} catch {
		// older APK / web
	}
}
