import { formatApiError } from '$lib/api/errors';
import { toast } from '$lib/toast';

export function capturePageLoadError(err: unknown): string {
	return formatApiError(err, 'common.loadFailed');
}

export type PageLoadFailOpts = {
	background?: boolean;
	silent?: boolean;
	hasData?: boolean;
};

/**
 * Returns an error message for inline page UI, or null if the failure was reported via toast only
 * (background/silent refresh while cached data is already shown).
 */
export function reportPageLoadFailure(err: unknown, opts: PageLoadFailOpts = {}): string | null {
	const soft = opts.background || opts.silent;
	if (soft && opts.hasData) {
		toast.fromError(err);
		return null;
	}
	return capturePageLoadError(err);
}
