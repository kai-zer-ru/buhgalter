import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	applyUiStyle,
	initUiStyle,
	isUiStyle,
	readUiStyle,
	setUiStyle,
	UI_STYLE_STORAGE_KEY
} from './ui-style';

function mockDom() {
	const root = {
		dataset: {} as Record<string, string>
	};
	vi.stubGlobal('document', {
		documentElement: root
	});
	return { root };
}

function mockLocalStorage(initial: Record<string, string> = {}) {
	const store = new Map<string, string>(Object.entries(initial));
	const ls = {
		getItem: vi.fn((key: string) => (store.has(key) ? store.get(key)! : null)),
		setItem: vi.fn((key: string, value: string) => {
			store.set(key, value);
		}),
		removeItem: vi.fn((key: string) => {
			store.delete(key);
		})
	};
	vi.stubGlobal('localStorage', ls);
	return { store, ls };
}

describe('ui_style', () => {
	beforeEach(() => {
		mockDom();
		mockLocalStorage();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('accepts classic and material only', () => {
		expect(isUiStyle('classic')).toBe(true);
		expect(isUiStyle('material')).toBe(true);
		expect(isUiStyle('MaterialBased')).toBe(false);
		expect(isUiStyle('')).toBe(false);
	});

	it('defaults to classic when missing or invalid', () => {
		expect(readUiStyle()).toBe('classic');
		mockLocalStorage({ [UI_STYLE_STORAGE_KEY]: 'weird' });
		expect(readUiStyle()).toBe('classic');
	});

	it('applies dataset.uiStyle on documentElement', () => {
		const { root } = mockDom();
		applyUiStyle('material');
		expect(root.dataset.uiStyle).toBe('material');
		applyUiStyle('classic');
		expect(root.dataset.uiStyle).toBe('classic');
	});

	it('setUiStyle persists and applies', () => {
		const { root } = mockDom();
		const { store } = mockLocalStorage();
		setUiStyle('material');
		expect(store.get(UI_STYLE_STORAGE_KEY)).toBe('material');
		expect(root.dataset.uiStyle).toBe('material');
	});

	it('initUiStyle reads storage and applies', () => {
		const { root } = mockDom();
		mockLocalStorage({ [UI_STYLE_STORAGE_KEY]: 'material' });
		expect(initUiStyle()).toBe('material');
		expect(root.dataset.uiStyle).toBe('material');
	});
});
