import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { dropdownListStyle } from './dropdown-position';

function mockTrigger(rect: { top: number; bottom: number; left?: number; width?: number }) {
	const el = {
		getBoundingClientRect: () => ({
			top: rect.top,
			bottom: rect.bottom,
			left: rect.left ?? 0,
			width: rect.width ?? 200,
			right: (rect.left ?? 0) + (rect.width ?? 200),
			height: rect.bottom - rect.top,
			x: rect.left ?? 0,
			y: rect.top,
			toJSON: () => ({})
		})
	} as HTMLElement;

	return el;
}

describe('dropdownListStyle', () => {
	beforeEach(() => {
		vi.stubGlobal('window', { innerHeight: 800 });
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('opens down when there is enough space below', () => {
		const trigger = mockTrigger({ top: 100, bottom: 140 });

		const style = dropdownListStyle(trigger, 200, true);

		expect(style).toContain('top:144px');
		expect(style).not.toContain('bottom:');
	});

	it('opens up when below is tight but above fits', () => {
		const trigger = mockTrigger({ top: 500, bottom: 540 });

		const style = dropdownListStyle(trigger, 360, true);

		expect(style).toContain('bottom:304px');
		expect(style).not.toContain('top:');
	});

	it('opens up when below slightly underestimates panel height (regression)', () => {
		const trigger = mockTrigger({ top: 410, bottom: 450 });

		const style = dropdownListStyle(trigger, 360, true);

		expect(style).toContain('bottom:394px');
		expect(style).not.toContain('top:');
	});

	it('opens toward the side with more space when neither direction fully fits', () => {
		vi.stubGlobal('window', { innerHeight: 500 });
		const trigger = mockTrigger({ top: 300, bottom: 340 });

		const style = dropdownListStyle(trigger, 360, true);

		expect(style).toContain('bottom:204px');
	});

	it('uses relative positioning when usePortal is false', () => {
		const trigger = mockTrigger({ top: 500, bottom: 540 });

		expect(dropdownListStyle(trigger, 360, false)).toBe('bottom:100%;margin-bottom:4px;top:auto;');
	});
});
