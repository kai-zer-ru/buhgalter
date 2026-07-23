import { registerPlugin } from '@capacitor/core';

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
		allowUntrusted?: boolean;
	}): Promise<NativeHttpResult>;
}

export const SslTrust = registerPlugin<SslTrustPlugin>('SslTrust');
