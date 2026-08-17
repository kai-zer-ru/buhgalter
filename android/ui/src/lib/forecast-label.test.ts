import { describe, expect, it } from 'vitest';
import {
	aggregateForecastLabelFlags,
	forecastWithLabelKey
} from './forecast-label';

describe('forecastWithLabelKey', () => {
	it('planned only', () => {
		expect(forecastWithLabelKey({ has_planned_this_month: true })).toBe(
			'dashboard.withPlans'
		);
	});

	it('subscriptions only', () => {
		expect(forecastWithLabelKey({ has_subscriptions_this_month: true })).toBe(
			'dashboard.withSubscriptions'
		);
	});

	it('both', () => {
		expect(
			forecastWithLabelKey({
				has_planned_this_month: true,
				has_subscriptions_this_month: true
			})
		).toBe('dashboard.withPlansAndSubscriptions');
	});

	it('neither falls back to planned', () => {
		expect(forecastWithLabelKey({})).toBe('dashboard.withPlans');
	});
});

describe('aggregateForecastLabelFlags', () => {
	it('ORs flags across accounts', () => {
		expect(
			aggregateForecastLabelFlags([
				{ has_planned_this_month: true },
				{ has_subscriptions_this_month: true }
			])
		).toEqual({
			has_planned_this_month: true,
			has_subscriptions_this_month: true
		});
	});
});
