import { describe, expect, it, vi } from 'vitest';
import { nativeTimePicker, openNativePicker } from './native-time-picker';

function fakeInput(overrides: Partial<{ disabled: boolean; readOnly: boolean }> = {}) {
	const listeners = new Map<string, EventListener>();
	const input = {
		disabled: false,
		readOnly: false,
		...overrides,
		showPicker: vi.fn() as (() => void) | undefined,
		addEventListener: vi.fn((type: string, fn: EventListener) => listeners.set(type, fn)),
		removeEventListener: vi.fn((type: string, fn: EventListener) => {
			if (listeners.get(type) === fn) listeners.delete(type);
		})
	};
	const click = () => listeners.get('click')?.(new Event('click'));
	return { input, listeners, click };
}

describe('openNativePicker', () => {
	it('calls showPicker', () => {
		const { input } = fakeInput();
		expect(openNativePicker(input)).toBe(true);
		expect(input.showPicker).toHaveBeenCalledOnce();
	});

	it('does nothing without showPicker', () => {
		const { input } = fakeInput();
		input.showPicker = undefined;
		expect(openNativePicker(input)).toBe(false);
	});

	it('swallows showPicker errors', () => {
		const { input } = fakeInput();
		input.showPicker = vi.fn(() => {
			throw new DOMException('no gesture', 'NotAllowedError');
		});
		expect(() => openNativePicker(input)).not.toThrow();
		expect(openNativePicker(input)).toBe(false);
	});

	it('skips disabled and readOnly fields', () => {
		for (const flags of [{ disabled: true }, { readOnly: true }]) {
			const { input } = fakeInput(flags);
			expect(openNativePicker(input)).toBe(false);
			expect(input.showPicker).not.toHaveBeenCalled();
		}
	});
});

describe('nativeTimePicker action', () => {
	it('opens the picker on click', () => {
		const { input, click } = fakeInput();
		nativeTimePicker(input as unknown as HTMLInputElement);
		click();
		expect(input.showPicker).toHaveBeenCalledOnce();
	});

	it('removes the click listener on destroy', () => {
		const { input, listeners, click } = fakeInput();
		const action = nativeTimePicker(input as unknown as HTMLInputElement);
		action.destroy();
		expect(listeners.has('click')).toBe(false);
		click();
		expect(input.showPicker).not.toHaveBeenCalled();
	});
});
