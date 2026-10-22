import { describe, expect, it } from 'vitest';
import { refCachePathMatches, refCacheUpdateMatches } from './ref-cache-watch';

describe('refCachePathMatches', () => {
	it('matches exact path', () => {
		const path = '/api/v1/dashboard';
		expect(refCachePathMatches(path, path)).toBe(true);
	});

	it('does not match unrelated path', () => {
		expect(refCachePathMatches('/api/v1/accounts', '/api/v1/dashboard')).toBe(false);
	});

	it('matches wildcard realtime invalidate', () => {
		expect(refCachePathMatches('*', '/api/v1/dashboard')).toBe(true);
		expect(refCachePathMatches('*', ['/api/v1/accounts', '/api/v1/budget'])).toBe(true);
	});

	it('matches account list hint to account detail watch', () => {
		expect(refCachePathMatches('/api/v1/accounts', '/api/v1/accounts/acc-1')).toBe(true);
	});

	it('matches account detail hint to accounts list watch', () => {
		expect(refCachePathMatches('/api/v1/accounts/acc-1', '/api/v1/accounts')).toBe(true);
	});

	it('matches path ignoring query', () => {
		expect(
			refCachePathMatches('/api/v1/transactions', '/api/v1/transactions?kind=manual&page=1')
		).toBe(true);
	});
});

describe('refCacheUpdateMatches', () => {
	it('matches any of multiple hint paths', () => {
		const update = {
			path: '/api/v1/transactions',
			paths: ['/api/v1/transactions', '/api/v1/dashboard', '/api/v1/accounts']
		};
		expect(refCacheUpdateMatches(update, '/api/v1/dashboard')).toBe(true);
		expect(refCacheUpdateMatches(update, '/api/v1/transaction-templates')).toBe(false);
	});

	it('falls back to single path when paths omitted', () => {
		expect(refCacheUpdateMatches({ path: '/api/v1/budgets' }, '/api/v1/budgets/summary')).toBe(
			true
		);
	});
});
