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
