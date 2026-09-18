import { describe, expect, it } from 'vitest';
import { exportCSVUrl } from './export-url';

describe('exportCSVUrl', () => {
	it('omits date and entity filters for native buhgalter format', () => {
		expect(
			exportCSVUrl({
				from: '2025-01-01',
				to: '2026-09-18',
				account_id: 'acc-1',
				category_id: 'cat-1',
				format: 'buhgalter'
			})
		).toBe('/api/v1/export?format=buhgalter');
	});

	it('keeps Cubux range filters', () => {
		expect(
			exportCSVUrl({
				from: '2025-01-01',
				to: '2025-12-31',
				format: 'cubux'
			})
		).toBe('/api/v1/export?from=2025-01-01&to=2025-12-31&format=cubux');
	});
});
