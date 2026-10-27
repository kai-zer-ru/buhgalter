import { describe, expect, it } from 'vitest';
import { matchRefByName, refsForAccount, remapRefIds } from './refs-for-account';

describe('refsForAccount', () => {
	it('keeps personal refs for a personal account and missing scope', () => {
		const refs = [
			{ id: 'p', name: 'Food', scope: 'personal' as const },
			{ id: 'h', name: 'Food', scope: 'household' as const },
			{ id: 'old', name: 'Legacy' }
		];
		expect(refsForAccount({ scope: 'personal' }, refs).map((r) => r.id)).toEqual(['p', 'old']);
		expect(refsForAccount({}, refs).map((r) => r.id)).toEqual(['p', 'old']);
	});

	it('keeps household refs for a household account', () => {
		const refs = [
			{ id: 'p', name: 'Food', scope: 'personal' as const },
			{ id: 'h', name: 'Groceries', scope: 'household' as const }
		];
		expect(refsForAccount({ scope: 'household' }, refs).map((r) => r.id)).toEqual(['h']);
	});
});

describe('matchRefByName', () => {
	it('prefers the same id, then the same name in the new catalog', () => {
		const current = { id: 'old', name: 'Food' };
		expect(matchRefByName(current, [{ id: 'old', name: 'Food' }])?.id).toBe('old');
		expect(matchRefByName(current, [{ id: 'new', name: 'Food' }])?.id).toBe('new');
		expect(matchRefByName(current, [{ id: 'x', name: 'Other' }])).toBeUndefined();
	});
});

describe('remapRefIds', () => {
	it('drops refs that have no name match', () => {
		const { next, dropped } = remapRefIds(
			[
				{ id: 'a', name: 'Alpha' },
				{ id: 'b', name: 'Beta' }
			],
			[{ id: 'a2', name: 'Alpha' }]
		);
		expect(next.map((r) => r.id)).toEqual(['a2']);
		expect(dropped).toBe(true);
	});
});
