import type { Dashboard } from '$lib/api/client';
import { readRefCache } from '$lib/ref-cache';

export const DASHBOARD_REF_PATH = '/api/v1/dashboard';

/** Inbox queue size from last cached dashboard snapshot. */
export function readInboxCount(): number {
	const dash = readRefCache<Dashboard>(DASHBOARD_REF_PATH);
	const n = dash?.inbox_count;
	if (typeof n !== 'number' || n <= 0) return 0;
	return n;
}
