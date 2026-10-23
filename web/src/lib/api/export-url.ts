export type ExportCSVParams = {
	from?: string;
	to?: string;
	account_id?: string;
	category_id?: string;
	format?: 'buhgalter' | 'cubux';
};

/** Native Buhgalter dump is always the full ledger; date/account/category filters are Cubux-only. */
export function exportCSVUrl(params: ExportCSVParams): string {
	const q = new URLSearchParams();
	const format = params.format ?? 'buhgalter';
	const scoped = format === 'cubux';
	if (scoped && params.from) q.set('from', params.from);
	if (scoped && params.to) q.set('to', params.to);
	if (scoped && params.account_id) q.set('account_id', params.account_id);
	if (scoped && params.category_id) q.set('category_id', params.category_id);
	q.set('format', format);
	const qs = q.toString();
	return `/api/v1/export${qs ? `?${qs}` : ''}`;
}
