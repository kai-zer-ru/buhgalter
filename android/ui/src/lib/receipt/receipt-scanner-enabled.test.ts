import { describe, expect, it } from 'vitest';
import { isReceiptApiTokenConfigured, isReceiptScannerAvailable } from './receipt-scanner-enabled';

describe('receipt-scanner-enabled', () => {
	it('treats missing receipt_api_token_set as configured', () => {
		expect(isReceiptApiTokenConfigured({ receipt_scanner: true })).toBe(true);
	});

	it('requires receipt_api_token_set when present', () => {
		expect(isReceiptApiTokenConfigured({ receipt_api_token_set: false })).toBe(false);
		expect(isReceiptApiTokenConfigured({ receipt_api_token_set: true })).toBe(true);
	});

	it('combines flag and token', () => {
		expect(isReceiptScannerAvailable({ receipt_scanner: false, receipt_api_token_set: true })).toBe(
			false
		);
		expect(isReceiptScannerAvailable({ receipt_scanner: true, receipt_api_token_set: true })).toBe(
			true
		);
	});
});
