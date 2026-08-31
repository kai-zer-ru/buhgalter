type WithIdUpdated = { id?: string; updated_at?: string };

function recordsSame(prev: WithIdUpdated[], next: WithIdUpdated[]): boolean {
	if (prev.length !== next.length) return false;
	for (let i = 0; i < prev.length; i++) {
		const a = prev[i];
		const b = next[i];
		if (a?.id !== b?.id) return false;
		if (a?.updated_at !== b?.updated_at) return false;
	}
	return true;
}

/** Drop presentation-only fields (`*_display`) so equality ignores formatting noise. */
export function stripDisplayFields(value: unknown): unknown {
	if (value === null || typeof value !== 'object') return value;
	if (Array.isArray(value)) return value.map(stripDisplayFields);
	const out: Record<string, unknown> = {};
	for (const [key, child] of Object.entries(value as Record<string, unknown>)) {
		if (key.endsWith('_display')) continue;
		out[key] = stripDisplayFields(child);
	}
	return out;
}

export function stableEqual(a: unknown, b: unknown): boolean {
	return JSON.stringify(stripDisplayFields(a)) === JSON.stringify(stripDisplayFields(b));
}

/** Heuristic: dashboard payload from GET /api/v1/dashboard. */
export function isDashboardShape(value: unknown): boolean {
	if (!value || typeof value !== 'object') return false;
	const o = value as Record<string, unknown>;
	return 'total_balance' in o && 'accounts' in o && 'debts_summary' in o;
}

export function isDashboardRefPath(path: string): boolean {
	return (path.split('?')[0] ?? path) === '/api/v1/dashboard';
}

/** Skip $state assignment when value is unchanged — fewer reactive runs on background refresh. */
export function assignIfChanged<T>(prev: T, next: T): T {
	if (prev === next) return prev;
	if (isDashboardShape(prev) && isDashboardShape(next)) {
		return stableEqual(prev, next) ? prev : next;
	}
	if (Array.isArray(prev) && Array.isArray(next)) {
		if (recordsSame(prev as WithIdUpdated[], next as WithIdUpdated[])) return prev;
	}
	if (JSON.stringify(prev) === JSON.stringify(next)) return prev;
	return next;
}

/** IDs present in next but not in prev (for row enter animations). */
export function diffNewIds<T extends { id: string }>(prev: T[], next: T[]): Set<string> {
	const prevIds = new Set(prev.map((item) => item.id));
	const added = new Set<string>();
	for (const item of next) {
		if (!prevIds.has(item.id)) added.add(item.id);
	}
	return added;
}
