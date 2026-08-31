import { WebPlugin } from '@capacitor/core';

export class AppInstanceWeb extends WebPlugin {
	async getStorageNamespace(): Promise<{ namespace: string }> {
		return { namespace: '' };
	}
}
