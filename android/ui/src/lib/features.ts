import { get, writable } from 'svelte/store';
import { getFeatures, type FeatureFlagsSnapshot } from '$lib/api/client';

export const featureFlags = writable<FeatureFlagsSnapshot | null>(null);

const PATH_FEATURE_GATES: Array<{ prefix: string; feature: string }> = [
	{ prefix: '/budget', feature: 'budget' },
	{ prefix: '/stats', feature: 'stats' },
	{ prefix: '/debts', feature: 'debts' },
	{ prefix: '/debtors', feature: 'debts' },
	{ prefix: '/credits', feature: 'credits' },
	{ prefix: '/merchants', feature: 'merchants_tags' },
	{ prefix: '/tags', feature: 'merchants_tags' },
	{ prefix: '/recurring-operations', feature: 'recurring' },
	{ prefix: '/subscriptions', feature: 'subscriptions' },
	{ prefix: '/settings/notifications', feature: 'notifications' },
	{ prefix: '/settings/import', feature: 'import_export' },
	{ prefix: '/transaction-templates', feature: 'transaction_templates' }
];

/** Pass `$featureFlags` from Svelte templates/derived so the UI re-runs when flags load or change. */
export function isFeatureEnabled(
	key: string,
	snap: FeatureFlagsSnapshot | null = get(featureFlags)
): boolean {
	if (!snap) return true;
	if (!(key in snap)) return true;
	return snap[key] === true;
}

export async function loadFeatureFlags(): Promise<void> {
	try {
		const snap = await getFeatures();
		featureFlags.set(snap);
	} catch {
		// Keep previous snapshot offline; only clear when never loaded.
		if (get(featureFlags) === null) {
			featureFlags.set(null);
		}
	}
}

export function clearFeatureFlags() {
	featureFlags.set(null);
}

export function featureRequiredForPath(pathname: string): string | null {
	if (/^\/accounts\/[^/]+\/auto-topup(?:\/|$)/.test(pathname)) {
		return 'balance_maintenance';
	}
	for (const gate of PATH_FEATURE_GATES) {
		if (pathname === gate.prefix || pathname.startsWith(gate.prefix + '/')) {
			return gate.feature;
		}
	}
	return null;
}
