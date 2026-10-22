import { APP_VERSION } from '$lib/platform/app-version';
import { getServerUrl } from '$lib/platform/server-url';
import { failedOutboxCount, getOutboxEntries, pendingOutboxCount } from '$lib/offline/store';

export type OutboxExportPayload = {
	exportedAt: string;
	appVersion: string;
	serverUrl: string | null;
	pending: number;
	failed: number;
	entries: ReturnType<typeof getOutboxEntries>;
};

export function buildOutboxExportPayload(): OutboxExportPayload {
	return {
		exportedAt: new Date().toISOString(),
		appVersion: APP_VERSION,
		serverUrl: getServerUrl(),
		pending: pendingOutboxCount(),
		failed: failedOutboxCount(),
		entries: getOutboxEntries()
	};
}

export function exportOutboxForSupport(): string {
	return JSON.stringify(buildOutboxExportPayload(), null, 2);
}

export async function copyOutboxExportToClipboard(): Promise<void> {
	const text = exportOutboxForSupport();
	if (!navigator.clipboard?.writeText) {
		throw new Error('clipboard_unavailable');
	}
	await navigator.clipboard.writeText(text);
}
