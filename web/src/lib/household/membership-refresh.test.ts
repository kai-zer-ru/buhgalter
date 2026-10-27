import { get } from 'svelte/store';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const getHouseholdMe = vi.fn();

vi.mock('$lib/api/client', () => ({
	getHouseholdMe: () => getHouseholdMe()
}));

const store = new Map<string, string>();
vi.stubGlobal('localStorage', {
	getItem: (k: string) => store.get(k) ?? null,
	setItem: (k: string, v: string) => void store.set(k, v),
	removeItem: (k: string) => void store.delete(k)
});

const { householdMeLoaded, refreshHouseholdMe } = await import('./membership');
const { readStoredCatalogScope, writeStoredCatalogScope } = await import('./catalog-scope');

describe('refreshHouseholdMe catalog scope fallback', () => {
	beforeEach(() => {
		store.clear();
		getHouseholdMe.mockReset();
		householdMeLoaded.set(false);
	});

	it('kicked / left: resets a stored family tab to all', async () => {
		writeStoredCatalogScope('household');
		getHouseholdMe.mockResolvedValue({ household: null });
		await refreshHouseholdMe();
		expect(readStoredCatalogScope()).toBe('all');
		expect(get(householdMeLoaded)).toBe(true);
	});

	it('still a member: keeps the stored tab', async () => {
		writeStoredCatalogScope('household');
		getHouseholdMe.mockResolvedValue({
			household: { id: 'h', name: 'A', kind: 'family', created_at: '' }
		});
		await refreshHouseholdMe();
		expect(readStoredCatalogScope()).toBe('household');
	});

	it('network error: keeps the tab and is not a confirmed membership', async () => {
		writeStoredCatalogScope('household');
		householdMeLoaded.set(true);
		getHouseholdMe.mockRejectedValue(new Error('offline'));
		await expect(refreshHouseholdMe()).rejects.toThrow('offline');
		expect(readStoredCatalogScope()).toBe('household');
		expect(get(householdMeLoaded)).toBe(false);
	});
});
