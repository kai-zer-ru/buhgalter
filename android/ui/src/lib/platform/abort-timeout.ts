/** AbortSignal.timeout is missing on older Android WebViews (< Chromium 124). */
export function abortTimeout(ms: number): AbortSignal {
	if (typeof AbortSignal !== 'undefined' && typeof AbortSignal.timeout === 'function') {
		return AbortSignal.timeout(ms);
	}
	const controller = new AbortController();
	setTimeout(() => controller.abort(), ms);
	return controller.signal;
}
