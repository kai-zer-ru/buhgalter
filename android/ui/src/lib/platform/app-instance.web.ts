import { WebPlugin } from '@capacitor/core';

export class AppInstanceWeb extends WebPlugin {
	async getStorageNamespace(): Promise<{ namespace: string }> {
		return { namespace: '' };
	}

	async hideSoftKeyboard(): Promise<void> {
		// Browser uses the OS keyboard; MoneyInput is Android-only keypad UX.
	}
}
