import { afterEach, describe, expect, it, vi } from 'vitest';
import { clearMoneyKeypadInset, setMoneyKeypadInset } from './money-keypad-inset';

describe('money-keypad-inset', () => {
	const props = new Map<string, string>();

	afterEach(() => {
		props.clear();
		vi.unstubAllGlobals();
	});

	it('sets and clears the CSS variable on documentElement', () => {
		vi.stubGlobal('document', {
			documentElement: {
				style: {
					setProperty: (key: string, value: string) => {
						props.set(key, value);
					},
					getPropertyValue: (key: string) => props.get(key) ?? ''
				}
			}
		});

		setMoneyKeypadInset(240.4);
		expect(props.get('--money-keypad-inset')).toBe('240px');
		clearMoneyKeypadInset();
		expect(props.get('--money-keypad-inset')).toBe('0px');
	});
});
