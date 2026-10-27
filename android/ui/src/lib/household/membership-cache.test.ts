import { beforeEach, describe, expect, it, vi } from 'vitest';

const { clearRefCache, resetLedgerLocalState, scheduleWidgetSnapshotPublish, dataRefreshTick } =
	vi.hoisted(() => ({
		clearRefCache: vi.fn(),
		resetLedgerLocalState: vi.fn(),
		scheduleWidgetSnapshotPublish: vi.fn(),
		dataRefreshTick: { update: vi.fn((fn: (n: number) => number) => fn(0)) }
	}));

vi.mock('$lib/offline/ref-cache', () => ({
	clearRefCache
}));

vi.mock('$lib/offline/local-state', () => ({
	resetLedgerLocalState
}));

vi.mock('$lib/offline/sync', () => ({
	dataRefreshTick
}));

vi.mock('$lib/widgets/publish', () => ({
	scheduleWidgetSnapshotPublish
}));

import {
	applyHouseholdMembershipSnapshot,
	membershipFingerprint,
	resetCachesAfterMembershipChange,
	resetMembershipCacheForTests
} from './membership-cache';

describe('membershipFingerprint', () => {
	it('is empty without a household and includes id, role and members', () => {
		expect(membershipFingerprint({ household: null })).toBe('');
		expect(
			membershipFingerprint({
				household: { id: 'hh1', name: 'Fam', kind: 'family', created_at: '' },
				role: 'owner',
				members: [
					{ user_id: 'b', display_name: 'B', login: 'b', role: 'member', joined_at: '' },
					{ user_id: 'a', display_name: 'A', login: 'a', role: 'owner', joined_at: '' }
				]
			})
		).toBe('hh1:owner:a,b');
	});
});

describe('applyHouseholdMembershipSnapshot', () => {
	beforeEach(() => {
		resetMembershipCacheForTests();
		clearRefCache.mockClear();
		resetLedgerLocalState.mockClear();
		dataRefreshTick.update.mockClear();
	});

	it('does not reset on the first snapshot', () => {
		applyHouseholdMembershipSnapshot({
			household: { id: 'hh1', name: 'Fam', kind: 'family', created_at: '' },
			role: 'owner',
			members: []
		});
		expect(clearRefCache).not.toHaveBeenCalled();
	});

	it('resets caches when household id or members change', () => {
		applyHouseholdMembershipSnapshot({
			household: { id: 'hh1', name: 'Fam', kind: 'family', created_at: '' },
			role: 'owner',
			members: [{ user_id: 'a', display_name: 'A', login: 'a', role: 'owner', joined_at: '' }]
		});
		applyHouseholdMembershipSnapshot({ household: null });
		expect(clearRefCache).toHaveBeenCalled();
		expect(resetLedgerLocalState).toHaveBeenCalled();
	});
});

describe('resetCachesAfterMembershipChange', () => {
	it('clears ref-cache, ledger and schedules widgets', async () => {
		resetCachesAfterMembershipChange();
		expect(clearRefCache).toHaveBeenCalled();
		expect(resetLedgerLocalState).toHaveBeenCalled();
		await Promise.resolve();
		expect(scheduleWidgetSnapshotPublish).toHaveBeenCalled();
	});
});
