import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/platform/system-bars', () => ({
	syncSystemBars: vi.fn(() => Promise.resolve())
}));

import { applyThemePreference, isThemePreference, resolveTheme, THEME_COLORS } from './theme';

type Listener = (event: MediaQueryListEvent) => void;

function mockDom() {
	const classes = new Set<string>();
	const classList = {
		toggle(name: string, force?: boolean) {
			if (force === true) classes.add(name);
			else if (force === false) classes.delete(name);
			else if (classes.has(name)) classes.delete(name);
			else classes.add(name);
		},
		contains(name: string) {
			return classes.has(name);
		},
		remove(...names: string[]) {
			for (const name of names) classes.delete(name);
		}
	};
	const meta = {
		content: THEME_COLORS.light,
		getAttribute(name: string) {
			return name === 'content' ? this.content : null;
		},
		setAttribute(name: string, value: string) {
			if (name === 'content') this.content = value;
		}
	};
	const root = {
		classList,
		dataset: {} as Record<string, string>
	};
	vi.stubGlobal('document', {
		documentElement: root,
		head: { innerHTML: '' },
		querySelector: (sel: string) => (sel.includes('theme-color') ? meta : null)
	});
	return { root, meta, classes };
}

function mockMatchMedia(matches: boolean) {
	const listeners = new Set<Listener>();
	const media = {
		matches,
		media: '(prefers-color-scheme: dark)',
		onchange: null,
		addEventListener: vi.fn((type: string, listener: Listener) => {
			if (type === 'change') listeners.add(listener);
		}),
		removeEventListener: vi.fn((type: string, listener: Listener) => {
			if (type === 'change') listeners.delete(listener);
		}),
		dispatchEvent: vi.fn(),
		addListener: vi.fn(),
		removeListener: vi.fn()
	};
	const matchMediaFn = vi.fn(() => media);
	vi.stubGlobal('matchMedia', matchMediaFn);
	vi.stubGlobal('window', { matchMedia: matchMediaFn });
	return {
		media,
		setMatches(next: boolean) {
			media.matches = next;
			for (const listener of listeners) {
				listener({ matches: next } as MediaQueryListEvent);
			}
		}
	};
}

describe('theme preference', () => {
	beforeEach(() => {
		mockDom();
		mockMatchMedia(false);
	});

	afterEach(() => {
		applyThemePreference('light');
		vi.unstubAllGlobals();
	});

	it('accepts light, dark, system', () => {
		expect(isThemePreference('light')).toBe(true);
		expect(isThemePreference('dark')).toBe(true);
		expect(isThemePreference('system')).toBe(true);
		expect(isThemePreference('auto')).toBe(false);
	});

	it('resolves explicit preferences without media query', () => {
		mockMatchMedia(true);
		expect(resolveTheme('light')).toBe('light');
		expect(resolveTheme('dark')).toBe('dark');
	});

	it('resolves system from prefers-color-scheme', () => {
		mockMatchMedia(true);
		expect(resolveTheme('system')).toBe('dark');
		mockMatchMedia(false);
		expect(resolveTheme('system')).toBe('light');
	});

	it('applies system and reacts to media changes', () => {
		const { root, meta } = mockDom();
		const { media, setMatches } = mockMatchMedia(false);
		applyThemePreference('system');
		expect(root.classList.contains('dark')).toBe(false);
		expect(root.dataset.theme).toBe('light');
		expect(meta.getAttribute('content')).toBe(THEME_COLORS.light);
		expect(media.addEventListener).toHaveBeenCalled();

		setMatches(true);
		expect(root.classList.contains('dark')).toBe(true);
		expect(root.dataset.theme).toBe('dark');
		expect(meta.getAttribute('content')).toBe(THEME_COLORS.dark);
	});

	it('stops listening when switching away from system', () => {
		const { root } = mockDom();
		const { media } = mockMatchMedia(true);
		applyThemePreference('system');
		applyThemePreference('light');
		expect(media.removeEventListener).toHaveBeenCalled();
		expect(root.classList.contains('dark')).toBe(false);
	});
});
