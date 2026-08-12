import { beforeEach, describe, expect, it } from 'vitest';
import { featureFlags, featureRequiredForPath, isFeatureEnabled } from './features';

describe('isFeatureEnabled', () => {
	beforeEach(() => {
		featureFlags.set(null);
	});

	it('defaults to true when snapshot not loaded', () => {
		expect(isFeatureEnabled('budget')).toBe(true);
	});

	it('reads snapshot values', () => {
		featureFlags.set({ budget: false, debts: true });
		expect(isFeatureEnabled('budget')).toBe(false);
		expect(isFeatureEnabled('debts')).toBe(true);
	});
});

describe('featureRequiredForPath', () => {
	it('maps known module paths', () => {
		expect(featureRequiredForPath('/budget')).toBe('budget');
		expect(featureRequiredForPath('/debtors/1')).toBe('debts');
		expect(featureRequiredForPath('/settings/notifications')).toBe('notifications');
		expect(featureRequiredForPath('/accounts/1/auto-topup')).toBe('balance_maintenance');
		expect(featureRequiredForPath('/accounts')).toBeNull();
	});

	it('accepts an explicit snapshot', () => {
		expect(isFeatureEnabled('budget', { budget: false })).toBe(false);
		expect(isFeatureEnabled('budget', { budget: true })).toBe(true);
	});
});
