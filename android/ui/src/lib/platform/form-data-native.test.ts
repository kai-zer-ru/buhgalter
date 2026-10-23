import { describe, expect, it } from 'vitest';
import { formDataToNativeMultipart } from './form-data-native';

describe('formDataToNativeMultipart', () => {
	it('encodes text fields and files', async () => {
		const form = new FormData();
		form.append('confirm', 'RESTORE');
		form.append(
			'image',
			new File([new Uint8Array([1, 2, 3])], 'receipt.jpg', { type: 'image/jpeg' })
		);

		const parts = await formDataToNativeMultipart(form);
		expect(parts).toHaveLength(2);
		expect(parts[0]).toEqual({ name: 'confirm', value: 'RESTORE' });
		expect(parts[1]).toMatchObject({
			name: 'image',
			filename: 'receipt.jpg',
			mime: 'image/jpeg',
			base64: 'AQID'
		});
	});

	it('encodes a payload larger than one chunk', async () => {
		const bytes = new Uint8Array(9000);
		for (let i = 0; i < bytes.length; i++) bytes[i] = i % 256;
		const form = new FormData();
		form.append('image', new File([bytes], 'big.jpg', { type: 'image/jpeg' }));
		const parts = await formDataToNativeMultipart(form);
		expect(parts[0]).toMatchObject({
			name: 'image',
			base64: Buffer.from(bytes).toString('base64')
		});
	});
});
