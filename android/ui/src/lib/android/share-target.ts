import { registerPlugin } from '@capacitor/core';
import { isNativeApp } from '$lib/platform/native';

export type ShareTargetPayload = {
	text?: string;
	subject?: string;
	streamUri?: string;
	mimeType?: string;
};

interface ShareTargetPlugin {
	consumePending(): Promise<ShareTargetPayload>;
	addListener(
		eventName: 'shareReceived',
		listenerFunc: (payload: ShareTargetPayload) => void
	): Promise<{ remove: () => void }>;
}

const ShareTarget = registerPlugin<ShareTargetPlugin>('ShareTarget');

let pendingPrefill: { description: string } | null = null;

/** Queue description for the next create-transaction form (share / deep link). */
export function setSharePrefill(description: string): void {
	const trimmed = description.trim();
	pendingPrefill = trimmed ? { description: trimmed.slice(0, 2000) } : null;
}

/** Take and clear share prefill (once). */
export function takeSharePrefill(): { description: string } | null {
	const v = pendingPrefill;
	pendingPrefill = null;
	return v;
}

export function resetSharePrefillForTests(): void {
	pendingPrefill = null;
}

export function hasShareContent(payload: ShareTargetPayload): boolean {
	return Boolean(
		(payload.text && payload.text.trim()) ||
		(payload.subject && payload.subject.trim()) ||
		payload.streamUri
	);
}

/** Build description for expense form from a share payload (no OCR). */
export function descriptionFromShare(payload: ShareTargetPayload, imageFallback: string): string {
	const parts: string[] = [];
	if (payload.subject?.trim()) parts.push(payload.subject.trim());
	if (payload.text?.trim()) parts.push(payload.text.trim());
	if (parts.length) return parts.join('\n').slice(0, 2000);
	if (payload.streamUri) return imageFallback;
	return '';
}

export const SHARE_EXPENSE_ROUTE = '/transactions/new?type=expense';

/**
 * Cold-start + warm share targets → route to expense form.
 * Returns cleanup for the Cap listener.
 */
export async function initShareTargetListener(
	onRoute: (route: string) => void,
	imageFallback: string | (() => string)
): Promise<() => void> {
	if (!isNativeApp()) {
		return () => undefined;
	}

	const fallback = () => (typeof imageFallback === 'function' ? imageFallback() : imageFallback);

	const handle = (payload: ShareTargetPayload) => {
		if (!hasShareContent(payload)) return;
		const description = descriptionFromShare(payload, fallback());
		if (description) setSharePrefill(description);
		onRoute(SHARE_EXPENSE_ROUTE);
	};

	try {
		const pending = await ShareTarget.consumePending();
		handle(pending);
	} catch {
		// plugin unavailable in browser / tests
	}

	let remove: (() => void) | undefined;
	try {
		const handleRef = await ShareTarget.addListener('shareReceived', handle);
		remove = () => void handleRef.remove();
	} catch {
		remove = undefined;
	}

	return () => {
		remove?.();
	};
}
