import type { FeatureFlagsSnapshot } from '$lib/api/client';
import { isFeatureEnabled } from '$lib/features';

/** True when the server exposes receipt_api_token_set and it is set (pre-v1.10: field absent → configured). */
export function isReceiptApiTokenConfigured(snap: FeatureFlagsSnapshot | null): boolean {
	if (!snap) return false;
	if (!('receipt_api_token_set' in snap)) return true;
	return snap.receipt_api_token_set === true;
}

export function isReceiptScannerAvailable(snap: FeatureFlagsSnapshot | null): boolean {
	return isFeatureEnabled('receipt_scanner', snap) && isReceiptApiTokenConfigured(snap);
}
