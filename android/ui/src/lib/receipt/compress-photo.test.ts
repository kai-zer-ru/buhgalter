import { describe, expect, it } from 'vitest';
import { compressReceiptPhoto, ReceiptPhotoError } from './compress-photo';

describe('compressReceiptPhoto', () => {
	it('returns non-images unchanged', async () => {
		const file = new File(['x'], 'a.txt', { type: 'text/plain' });
		expect(await compressReceiptPhoto(file)).toBe(file);
	});

	it('returns small images unchanged', async () => {
		const file = new File([new Uint8Array(100)], 'a.jpg', { type: 'image/jpeg' });
		expect(await compressReceiptPhoto(file)).toBe(file);
	});

	it('leaves a large non-image unchanged', async () => {
		const file = new File([new Uint8Array(800_000)], 'notes.txt', { type: 'text/plain' });
		expect(await compressReceiptPhoto(file)).toBe(file);
	});

	it('rejects a photo that cannot be shrunk under the bridge limit', async () => {
		const file = new File([new Uint8Array(800_000)], 'shot.jpg', { type: 'image/jpeg' });
		await expect(compressReceiptPhoto(file)).rejects.toBeInstanceOf(ReceiptPhotoError);
	});
});
