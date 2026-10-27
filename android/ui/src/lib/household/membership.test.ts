import { describe, expect, it } from 'vitest';
import { householdKindKey, householdNavLabelKey } from './membership';

describe('householdNavLabelKey', () => {
	it('falls back to the combined menu title without a group', () => {
		expect(householdNavLabelKey({ household: null })).toBe('settings.tab.household');
		expect(householdNavLabelKey(null)).toBe('settings.tab.household');
	});

	it('uses family or team once the group exists', () => {
		expect(
			householdNavLabelKey({
				household: { id: 'h', name: 'A', kind: 'family', created_at: '' }
			})
		).toBe('household.kind.family');
		expect(
			householdNavLabelKey({
				household: { id: 'h', name: 'B', kind: 'team', created_at: '' }
			})
		).toBe('household.kind.team');
	});
});

describe('householdKindKey', () => {
	it('uses the team variant only for teams', () => {
		expect(householdKindKey('accounts.shared.toggle', 'team')).toBe('accounts.shared.toggle.team');
		expect(householdKindKey('accounts.shared.toggle', 'family')).toBe('accounts.shared.toggle');
		expect(householdKindKey('household.leave', undefined)).toBe('household.leave');
	});
});
