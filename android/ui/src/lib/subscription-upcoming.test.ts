import { describe, expect, it } from 'vitest';
import {
	seedUpcomingLocalDates,
	upcomingLocalToAPI,
	upcomingToLocalDates
} from './subscription-upcoming';

describe('subscription-upcoming', () => {
	it('seeds three local dates from period', () => {
		expect(seedUpcomingLocalDates('2026-03-01', 'month')).toEqual([
			'2026-03-01',
			'2026-04-01',
			'2026-05-01'
		]);
	});

	it('maps upcoming for offline local entity', () => {
		const local = upcomingToLocalDates(
			['2026-03-01T05:00:00Z', '2026-03-29T05:00:00Z', '2026-04-26T05:00:00Z'],
			'UTC'
		);
		const api = upcomingLocalToAPI(local, '08:00', 'UTC');
		expect(api[0]).toBeTruthy();
		expect(api).toHaveLength(3);
	});
});
