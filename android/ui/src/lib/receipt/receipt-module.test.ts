import { describe, expect, it } from 'vitest';
import { isReceiptApiTokenConfigured, isReceiptScannerEnabled } from './receipt-module';

describe('receipt-module', () => {
	it('treats missing receipt_api_token_set as configured', () => {
		expect(isReceiptApiTokenConfigured({ receipt_scanner: true })).toBe(true);
	});

	it('requires receipt_api_token_set when present', () => {
		expect(isReceiptApiTokenConfigured({ receipt_api_token_set: false })).toBe(false);
		expect(isReceiptApiTokenConfigured({ receipt_api_token_set: true })).toBe(true);
	});

	it('combines feature flag and token', () => {
		expect(isReceiptScannerEnabled({ receipt_scanner: false, receipt_api_token_set: true })).toBe(
			false
		);
		expect(isReceiptScannerEnabled({ receipt_scanner: true, receipt_api_token_set: true })).toBe(
			true
		);
	});
});
