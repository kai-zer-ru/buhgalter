import { SystemBars, SystemBarsStyle } from '@capacitor/core';
import { isNativeApp } from '$lib/platform/native';

/**
 * Sync status/navigation bar icon contrast with the resolved SPA theme.
 * Dark theme → light glyphs (SystemBarsStyle.Dark); light → dark glyphs.
 */
export async function syncSystemBars(theme: 'light' | 'dark'): Promise<void> {
	if (!isNativeApp()) return;
	const style = theme === 'dark' ? SystemBarsStyle.Dark : SystemBarsStyle.Light;
	try {
		await SystemBars.setStyle({ style });
	} catch {
		// Soft-fail: plugin missing / old WebView — UI still fine.
	}
}
