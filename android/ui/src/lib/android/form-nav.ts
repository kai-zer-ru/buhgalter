import { goto } from '$app/navigation';
import { resolve } from '$app/paths';

/**
 * Resolve a dynamically-built app path for Kit typed routes.
 * Prefer this (or `form-routes` helpers) over `resolve(plainString)`.
 */
export function resolveAppPath(path: string): ReturnType<typeof resolve> {
	// Single literal member of Pathname — union Pathname breaks resolve()'s generic.
	return resolve(path as '/');
}

/**
 * Replace the current history entry.
 * Use for wizard steps and when the form itself was opened with replaceState
 * (credit create). Do **not** use after a form opened with push from `returnTo` —
 * that duplicates `returnTo` and makes the first Back appear to do nothing; use
 * {@link leaveForm} instead.
 */
export function gotoReplace(path: string): ReturnType<typeof goto> {
	// resolve() must appear at the goto() call site (svelte/no-navigation-without-resolve).
	return goto(resolve(path as '/'), { replaceState: true });
}

/**
 * Leave a create/edit form that was opened with push.
 * Pops the form via `history.back()` so the stack is […, returnTo] again.
 * Falls back to {@link gotoReplace} when back did not leave the form (deep link).
 */
export function leaveForm(returnTo: string): ReturnType<typeof goto> {
	if (typeof window === 'undefined') {
		return gotoReplace(returnTo);
	}
	const formUrl = window.location.pathname + window.location.search;
	window.history.back();
	return new Promise((resolvePromise) => {
		const done = () => resolvePromise(undefined);
		// If back is a no-op (no prior entry), replace with returnTo.
		window.requestAnimationFrame(() => {
			window.requestAnimationFrame(() => {
				const stillOnForm = window.location.pathname + window.location.search === formUrl;
				if (stillOnForm) {
					void gotoReplace(returnTo).then(done);
				} else {
					done();
				}
			});
		});
	});
}
