/** Custom URL scheme for Android shortcuts / intents (matches AndroidManifest). */
export const APP_URL_SCHEME = 'ru.kai_zer.buhgalter';

/** Parse `ru.kai_zer.buhgalter://transactions/new?type=expense` → app route path. */
export function parseAppDeepLink(raw: string): string | null {
	const trimmed = raw.trim();
	if (!trimmed) return null;
	const prefix = `${APP_URL_SCHEME}://`;
	if (!trimmed.startsWith(prefix)) return null;
	const rest = trimmed.slice(prefix.length);
	if (!rest || rest === '/') return '/';
	const route = rest.startsWith('/') ? rest : `/${rest}`;
	if (!route.startsWith('/')) return null;
	return route;
}

/** Path and query of an in-app href. The hash is not part of the identity. */
function appPathAndQuery(href: string): string {
	const trimmed = href.trim();
	if (!trimmed) return '/';
	try {
		const url = new URL(trimmed, 'https://localhost');
		return url.pathname + url.search;
	} catch {
		const noHash = trimmed.split('#')[0] ?? trimmed;
		return noHash.startsWith('/') ? noHash : `/${noHash}`;
	}
}

/** How an external link joins the open history entry. Replace is not an option. */
export type DeepLinkHistoryAction = 'skip' | 'push';

/**
 * Skip when the open page already has this path and query; otherwise push.
 * There is no replace action: replaceState would erase the cold-start `/`,
 * and system Back would exit instead of opening home.
 */
export function deepLinkHistoryAction(current: string, target: string): DeepLinkHistoryAction {
	if (appPathAndQuery(current) === appPathAndQuery(target)) return 'skip';
	return 'push';
}

/**
 * Apply the skip/push decision. Same path+query does not call `navigate`.
 * A push passes the original route and `{ replaceState: false }` so the open
 * entry (cold start: `/`) stays underneath.
 */
export function followDeepLinkHistory(
	current: string,
	route: string,
	navigate: (route: string, opts: { replaceState: false }) => void | Promise<void>
): void | Promise<void> {
	if (deepLinkHistoryAction(current, route) === 'skip') return;
	return navigate(route, { replaceState: false });
}

export type DeepLinkListener = (route: string) => void;

/** Subscribe to cold-start and warm deep links (`@capacitor/app`). */
export async function initDeepLinkListener(onRoute: DeepLinkListener): Promise<() => void> {
	const { App } = await import('@capacitor/app');

	const handle = (url: string | undefined) => {
		if (!url) return;
		const route = parseAppDeepLink(url);
		if (route) onRoute(route);
	};

	const launch = await App.getLaunchUrl();
	handle(launch?.url);

	const sub = await App.addListener('appUrlOpen', (event) => {
		handle(event.url);
	});

	return () => void sub.remove();
}
