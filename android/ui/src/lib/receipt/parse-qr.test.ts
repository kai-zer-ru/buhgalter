import { describe, expect, it } from 'vitest';
import { decodeQrFromImageFile } from './parse-qr';

describe('decodeQrFromImageFile', () => {
	it('returns null when the picture has no readable QR', async () => {
		const file = new File([new Uint8Array([1, 2, 3])], 'blank.png', { type: 'image/png' });
		expect(await decodeQrFromImageFile(file)).toBeNull();
	});
});
