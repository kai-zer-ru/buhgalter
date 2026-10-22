export function registerServiceWorker() {
	if (!('serviceWorker' in navigator)) return;
	if (import.meta.env.DEV) {
		void navigator.serviceWorker.getRegistrations().then((regs) => {
			for (const reg of regs) void reg.unregister();
		});
		return;
	}

	void navigator.serviceWorker.register('/service-worker.js').catch(() => {
		// Offline install is optional; ignore registration errors in dev or unsupported contexts.
	});
}
