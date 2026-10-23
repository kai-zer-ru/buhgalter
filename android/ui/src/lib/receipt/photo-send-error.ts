import { isServerUnavailableForUser } from '$lib/offline/server-connectivity';
import { ReceiptPhotoError } from './compress-photo';

/**
 * Photo upload / poll died on the device (bridge, timeout, no route to the server).
 * That is not a dropped session — `common.server_unavailable` must not be shown here.
 * A real receipt API error returns null so the caller uses `formatApiError`.
 */
export function receiptPhotoSendErrorKey(err: unknown): string | null {
	if (err instanceof ReceiptPhotoError) return 'receipt.photoTooLarge';
	if (isServerUnavailableForUser(err)) return 'receipt.photoNotSent';
	return null;
}
