const CACHE = 'buhgalter-static-v2';

const STATIC_PATH = /^\/(_app\/|icon-|manifest\.json|service-worker\.js)/;
const STATIC_EXT = /\.(js|css|woff2?|png|svg|ico|json)$/;
const DEV_PATH = /^\/(@fs\/|@vite\/|node_modules\/)/;

function isStaticAsset(url) {
	if (DEV_PATH.test(url.pathname)) return false;
	return STATIC_PATH.test(url.pathname) || STATIC_EXT.test(url.pathname);
}

self.addEventListener('install', (event) => {
	event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) => Promise.all(keys.filter((key) => key !== CACHE).map((key) => caches.delete(key))))
			.then(() => self.clients.claim())
	);
});

self.addEventListener('fetch', (event) => {
	const request = event.request;
	if (request.method !== 'GET') return;

	const url = new URL(request.url);
	if (url.origin !== self.location.origin) return;
	if (!isStaticAsset(url)) return;

	event.respondWith(
		caches.open(CACHE).then(async (cache) => {
			const cached = await cache.match(request);
			if (cached) return cached;

			const response = await fetch(request);
			if (response.ok) {
				cache.put(request, response.clone());
			}
			return response;
		})
	);
});
