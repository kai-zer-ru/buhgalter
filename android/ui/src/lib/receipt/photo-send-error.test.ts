import { describe, expect, it } from 'vitest';
import { ReceiptPhotoError } from './compress-photo';
import { receiptPhotoSendErrorKey } from './photo-send-error';

describe('receiptPhotoSendErrorKey', () => {
	it('does not treat a local upload failure as a dropped session', () => {
		expect(receiptPhotoSendErrorKey({ code: 'UNREACHABLE', message: 'Request timed out' })).toBe(
			'receipt.photoNotSent'
		);
	});

	it('names a photo that could not be prepared', () => {
		expect(receiptPhotoSendErrorKey(new ReceiptPhotoError())).toBe('receipt.photoTooLarge');
	});

	it('leaves a receipt API error to the usual formatter', () => {
		expect(
			receiptPhotoSendErrorKey({ code: 'ERR_RECEIPT_QR', message: 'bad qr', status: 400 })
		).toBeNull();
	});
});
