import { describe, expect, it } from 'vitest';
import { withTimeout } from './plugin-timeout';

describe('withTimeout', () => {
	it('resolves the original value when it finishes in time', async () => {
		await expect(withTimeout(Promise.resolve('ok'), 50, 'fallback')).resolves.toBe('ok');
	});

	it('returns fallback when the promise never settles', async () => {
		const hung = new Promise<string>(() => {
			/* never */
		});
		await expect(withTimeout(hung, 20, 'fallback')).resolves.toBe('fallback');
	});

	it('rejects when the original promise rejects before the timeout', async () => {
		await expect(withTimeout(Promise.reject(new Error('boom')), 50, 'fallback')).rejects.toThrow(
			'boom'
		);
	});
});
