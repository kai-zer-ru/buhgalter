import { describe, expect, it } from 'vitest';
import { refCachePathMatches } from './ref-cache-watch';

describe('refCachePathMatches', () => {
	it('matches exact path with query', () => {
		const path = '/api/v1/transactions?kind=manual&page=1';
		expect(refCachePathMatches(path, path)).toBe(true);
	});

	it('matches same base path', () => {
		expect(
			refCachePathMatches('/api/v1/transactions?kind=future', '/api/v1/transactions?kind=manual')
		).toBe(true);
	});

	it('does not match unrelated path', () => {
		expect(refCachePathMatches('/api/v1/accounts', '/api/v1/dashboard')).toBe(false);
	});

	it('matches any path in array', () => {
		expect(refCachePathMatches('/api/v1/banks', ['/api/v1/dashboard', '/api/v1/banks'])).toBe(true);
	});
});
