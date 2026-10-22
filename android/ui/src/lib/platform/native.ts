type CapacitorBridge = {
	isNativePlatform?: () => boolean;
};

function capacitor(): CapacitorBridge | undefined {
	if (typeof window === 'undefined') return undefined;
	return (window as Window & { Capacitor?: CapacitorBridge }).Capacitor;
}

/**
 * True when running inside a Capacitor native shell (APK / device).
 * Browser / Vite preview / Playwright see no Capacitor → false (fetch + loopback e2e OK).
 */
export function isNativeApp(): boolean {
	const cap = capacitor();
	if (!cap) return false;
	if (typeof cap.isNativePlatform === 'function') return cap.isNativePlatform();
	return true;
}

export function isMobileApp(): boolean {
	return isNativeApp();
}
