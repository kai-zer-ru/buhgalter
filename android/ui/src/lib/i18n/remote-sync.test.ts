import { beforeEach, describe, expect, it, vi } from 'vitest';
import { syncRemoteI18nOnMismatch } from './remote-sync';

const addMessages = vi.fn();
const waitLocale = vi.fn(async () => undefined);
const getUIi18n = vi.fn();

const memoryStore = new Map<string, string>();
vi.stubGlobal('localStorage', {
	clear: () => memoryStore.clear(),
	getItem: (key: string) => memoryStore.get(key) ?? null,
	setItem: (key: string, value: string) => {
		memoryStore.set(key, value);
	},
	removeItem: (key: string) => {
		memoryStore.delete(key);
	}
});

vi.mock('$app/environment', () => ({
	browser: true
}));

vi.mock('svelte-i18n', () => ({
	addMessages: (...args: unknown[]) => addMessages(...args),
	waitLocale: (...args: unknown[]) => waitLocale(...args)
}));

vi.mock('$lib/api/client', () => ({
	getUIi18n: (...args: unknown[]) => getUIi18n(...args)
}));

describe('syncRemoteI18nOnMismatch', () => {
	beforeEach(() => {
		memoryStore.clear();
		addMessages.mockClear();
		waitLocale.mockClear();
		getUIi18n.mockReset();
	});

	it('skips when app is not behind server', async () => {
		expect(await syncRemoteI18nOnMismatch('1.5.0', '1.4.0', 'ru')).toBe(false);
		expect(await syncRemoteI18nOnMismatch('1.4.0', '1.4.0', 'ru')).toBe(false);
		expect(getUIi18n).not.toHaveBeenCalled();
		expect(addMessages).not.toHaveBeenCalled();
	});

	it('skips when server version is unknown', async () => {
		expect(await syncRemoteI18nOnMismatch('1.4.0', null, 'ru')).toBe(false);
		expect(getUIi18n).not.toHaveBeenCalled();
	});

	it('fetches, applies and caches when app is behind', async () => {
		getUIi18n.mockResolvedValue({
			version: '1.5.0',
			lang: 'ru',
			messages: { 'app.title': 'Бухгалтер', 'nav.newKey': 'Новое' }
		});

		expect(await syncRemoteI18nOnMismatch('1.4.0', '1.5.0', 'ru')).toBe(true);
		expect(waitLocale).toHaveBeenCalledWith('ru');
		expect(getUIi18n).toHaveBeenCalledWith('ru');
		expect(addMessages).toHaveBeenCalledWith('ru', {
			'app.title': 'Бухгалтер',
			'nav.newKey': 'Новое'
		});
		expect(memoryStore.get('buhgalter.remote_i18n.1.5.0.ru')).toContain('nav.newKey');
	});

	it('uses cache on second call', async () => {
		memoryStore.set(
			'buhgalter.remote_i18n.1.5.0.en',
			JSON.stringify({ 'nav.home': 'Home (remote)' })
		);

		expect(await syncRemoteI18nOnMismatch('1.4.0', '1.5.0', 'en')).toBe(true);
		expect(getUIi18n).not.toHaveBeenCalled();
		expect(addMessages).toHaveBeenCalledWith('en', { 'nav.home': 'Home (remote)' });
	});

	it('soft-fails when network request fails', async () => {
		getUIi18n.mockRejectedValue(new Error('offline'));
		expect(await syncRemoteI18nOnMismatch('1.4.0', '1.5.0', 'ru')).toBe(false);
		expect(addMessages).not.toHaveBeenCalled();
	});
});
