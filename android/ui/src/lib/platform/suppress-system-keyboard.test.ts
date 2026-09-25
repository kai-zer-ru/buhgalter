import { afterEach, describe, expect, it, vi } from 'vitest';

const hideSoftKeyboard = vi.fn(async () => {});
const isNativeApp = vi.fn(() => false);

vi.mock('$lib/platform/app-instance', () => ({
	hideSoftKeyboard: () => hideSoftKeyboard()
}));

vi.mock('$lib/platform/native', () => ({
	isNativeApp: () => isNativeApp()
}));

vi.mock('@capacitor/app', () => ({
	App: {
		addListener: vi.fn(async () => ({ remove: vi.fn() }))
	}
}));

describe('suppressSystemKeyboard', () => {
	afterEach(() => {
		hideSoftKeyboard.mockClear();
		isNativeApp.mockReset();
		isNativeApp.mockReturnValue(false);
		vi.resetModules();
	});

	it('sets inputmode none and virtualkeyboardpolicy on the input', async () => {
		const { suppressSystemKeyboard } = await import('./suppress-system-keyboard');
		const attrs = new Map<string, string>();
		const input = {
			setAttribute: (k: string, v: string) => attrs.set(k, v)
		} as HTMLInputElement;

		suppressSystemKeyboard(input);

		expect(attrs.get('inputmode')).toBe('none');
		expect(attrs.get('virtualkeyboardpolicy')).toBe('manual');
		expect(hideSoftKeyboard).not.toHaveBeenCalled();
	});

	it('hides soft keyboard on native', async () => {
		isNativeApp.mockReturnValue(true);
		const { suppressSystemKeyboard } = await import('./suppress-system-keyboard');

		suppressSystemKeyboard();

		expect(hideSoftKeyboard).toHaveBeenCalledOnce();
	});
});
