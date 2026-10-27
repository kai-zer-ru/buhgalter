import { describe, expect, it } from 'vitest';
import { catalogScopeQuery, dashboardRefPath } from './catalog-scope';

describe('dashboardRefPath', () => {
	it('keeps the unscoped dashboard URL for all', () => {
		expect(dashboardRefPath('all')).toBe('/api/v1/dashboard');
		expect(dashboardRefPath()).toBe('/api/v1/dashboard');
		expect(catalogScopeQuery('all')).toBe('');
	});

	it('uses a distinct cache key per catalog tab', () => {
		expect(dashboardRefPath('personal')).toBe('/api/v1/dashboard?scope=personal');
		expect(dashboardRefPath('household')).toBe('/api/v1/dashboard?scope=household');
		expect(dashboardRefPath('personal')).not.toBe(dashboardRefPath('household'));
		expect(dashboardRefPath('personal')).not.toBe(dashboardRefPath('all'));
	});
});
