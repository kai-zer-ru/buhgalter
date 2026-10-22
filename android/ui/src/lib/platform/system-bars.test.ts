import { beforeEach, describe, expect, it, vi } from 'vitest';

const setStyle = vi.fn(() => Promise.resolve());
const isNativeApp = vi.fn(() => false);

vi.mock('@capacitor/core', () => ({
	SystemBars: { setStyle },
	SystemBarsStyle: { Dark: 'DARK', Light: 'LIGHT', Default: 'DEFAULT' }
}));

vi.mock('$lib/platform/native', () => ({
	isNativeApp: () => isNativeApp()
}));

describe('syncSystemBars', () => {
	beforeEach(() => {
		setStyle.mockClear();
		isNativeApp.mockReset();
		isNativeApp.mockReturnValue(false);
	});

	it('no-ops when not native', async () => {
		const { syncSystemBars } = await import('./system-bars');
		await syncSystemBars('dark');
		expect(setStyle).not.toHaveBeenCalled();
	});

	it('uses Dark style for dark theme on native', async () => {
		isNativeApp.mockReturnValue(true);
		const { syncSystemBars } = await import('./system-bars');
		await syncSystemBars('dark');
		expect(setStyle).toHaveBeenCalledWith({ style: 'DARK' });
	});

	it('uses Light style for light theme on native', async () => {
		isNativeApp.mockReturnValue(true);
		const { syncSystemBars } = await import('./system-bars');
		await syncSystemBars('light');
		expect(setStyle).toHaveBeenCalledWith({ style: 'LIGHT' });
	});

	it('swallows plugin errors', async () => {
		isNativeApp.mockReturnValue(true);
		setStyle.mockRejectedValueOnce(new Error('unavailable'));
		const { syncSystemBars } = await import('./system-bars');
		await expect(syncSystemBars('dark')).resolves.toBeUndefined();
	});
});
