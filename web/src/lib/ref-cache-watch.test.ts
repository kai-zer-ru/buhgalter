import { describe, expect, it } from 'vitest';
import { refCachePathMatches } from './ref-cache-watch';

describe('refCachePathMatches', () => {
	it('matches exact path with query', () => {
		const path = '/api/v1/transactions?kind=manual&page=1';
		expect(refCachePathMatches(path, path)).toBe(true);
	});

	it('does not match unrelated path', () => {
		expect(refCachePathMatches('/api/v1/accounts', '/api/v1/dashboard')).toBe(false);
	});

	it('matches wildcard realtime invalidate', () => {
		expect(refCachePathMatches('*', '/api/v1/dashboard')).toBe(true);
		expect(refCachePathMatches('*', ['/api/v1/accounts', '/api/v1/budget'])).toBe(true);
	});
});
