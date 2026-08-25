/** Flags that drive the «С учётом …» label on forecast balance. */
export type ForecastLabelFlags = {
	has_planned_this_month?: boolean;
	has_subscriptions_this_month?: boolean;
};

/**
 * i18n key for the forecast row:
 * - only planned (future and/or recurring) → withPlans
 * - only subscriptions → withSubscriptions
 * - both → withPlansAndSubscriptions
 */
export function forecastWithLabelKey(flags: ForecastLabelFlags): string {
	const planned = !!flags.has_planned_this_month;
	const subs = !!flags.has_subscriptions_this_month;
	if (planned && subs) return 'dashboard.withPlansAndSubscriptions';
	if (subs) return 'dashboard.withSubscriptions';
	return 'dashboard.withPlans';
}

export function aggregateForecastLabelFlags(accounts: ForecastLabelFlags[]): ForecastLabelFlags {
	let hasPlanned = false;
	let hasSubs = false;
	for (const a of accounts) {
		if (a.has_planned_this_month) hasPlanned = true;
		if (a.has_subscriptions_this_month) hasSubs = true;
		if (hasPlanned && hasSubs) break;
	}
	return {
		has_planned_this_month: hasPlanned,
		has_subscriptions_this_month: hasSubs
	};
}
