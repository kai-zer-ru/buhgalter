import { writable } from 'svelte/store';
import { getHouseholdMe, type HouseholdMe } from '$lib/api/client';
import { writeStoredCatalogScope } from './catalog-scope';

export const householdMe = writable<HouseholdMe>({ household: null });

/** True once the server confirmed membership (not the boot placeholder / an offline error). */
export const householdMeLoaded = writable(false);

export async function refreshHouseholdMe(): Promise<HouseholdMe> {
	try {
		const me = await getHouseholdMe();
		// Left / kicked: a stored family tab would hide personal accounts with no tabs to switch back.
		if (!me.household) writeStoredCatalogScope('all');
		householdMe.set(me);
		householdMeLoaded.set(true);
		return me;
	} catch (err) {
		householdMe.set({ household: null });
		householdMeLoaded.set(false);
		throw err;
	}
}

export function clearHouseholdMe() {
	householdMeLoaded.set(false);
	householdMe.set({ household: null });
}

export function inHousehold(me: HouseholdMe | null | undefined): boolean {
	return Boolean(me?.household);
}

export function householdNavLabelKey(me: HouseholdMe | null | undefined): string {
	if (me?.household?.kind === 'team') return 'household.kind.team';
	if (me?.household?.kind === 'family') return 'household.kind.family';
	return 'settings.tab.household';
}

/** i18n key for the current group kind: `<base>.team` for teams, the family wording otherwise. */
export function householdKindKey(base: string, kind: string | null | undefined): string {
	return kind === 'team' ? `${base}.team` : base;
}
