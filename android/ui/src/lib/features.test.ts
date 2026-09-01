import { beforeEach, describe, expect, it } from 'vitest';
import { featureFlags, featureRequiredForPath, isFeatureEnabled } from './features';
import { androidMainNavItems } from './android/nav-items';

describe('android feature flags', () => {
	beforeEach(() => {
		featureFlags.set(null);
	});

	it('hides budget nav when disabled', () => {
		featureFlags.set({ budget: false, debts: true });
		const keys = androidMainNavItems().map((i) => i.labelKey);
		expect(keys).not.toContain('nav.budget');
		expect(keys).toContain('nav.debts');
		expect(isFeatureEnabled('budget')).toBe(false);
	});

	it('maps paths to features', () => {
		expect(featureRequiredForPath('/budget')).toBe('budget');
		expect(featureRequiredForPath('/settings/import')).toBe('import_export');
		expect(featureRequiredForPath('/transaction-templates')).toBe('transaction_templates');
		expect(featureRequiredForPath('/accounts/1/auto-topup')).toBe('balance_maintenance');
	});
});
