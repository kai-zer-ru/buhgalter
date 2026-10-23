import { registerPlugin } from '@capacitor/core';
import type { NativeMultipartPart } from '$lib/platform/form-data-native';

export type NativeHttpErrorCode = 'SSL_CERTIFICATE' | 'UNREACHABLE';

export type NativeHttpResult = {
	status?: number;
	body?: string;
	ok?: boolean;
	errorCode?: NativeHttpErrorCode;
	message?: string;
};

export interface SslTrustPlugin {
	setTrustedOrigins(options: { origins: string[] }): Promise<void>;
	request(options: {
		url: string;
		method?: string;
		headers?: Record<string, string>;
		body?: string;
		multipart?: NativeMultipartPart[];
		/** Single file upload. Avoids a nested multipart array, which Android often drops. */
		imageBase64?: string;
		imageName?: string;
		imageMime?: string;
		allowUntrusted?: boolean;
	}): Promise<NativeHttpResult>;
}

export const SslTrust = registerPlugin<SslTrustPlugin>('SslTrust');
