import { describe, expect, it } from 'vitest';
import { parseAppDeepLink } from '$lib/android/deep-link';

describe('parseAppDeepLink', () => {
	it('parses expense shortcut URL', () => {
		expect(parseAppDeepLink('ru.kai_zer.buhgalter://transactions/new?type=expense')).toBe(
			'/transactions/new?type=expense'
		);
	});

	it('parses income, transfer and home deep links', () => {
		expect(parseAppDeepLink('ru.kai_zer.buhgalter://transactions/new?type=income')).toBe(
			'/transactions/new?type=income'
		);
		expect(parseAppDeepLink('ru.kai_zer.buhgalter://transfers/new')).toBe('/transfers/new');
		expect(parseAppDeepLink('ru.kai_zer.buhgalter://')).toBe('/');
		expect(parseAppDeepLink('ru.kai_zer.buhgalter://accounts/a1')).toBe('/accounts/a1');
	});

	it('rejects other schemes', () => {
		expect(parseAppDeepLink('https://example.com/foo')).toBeNull();
	});

	it('rejects empty input', () => {
		expect(parseAppDeepLink('')).toBeNull();
	});
});
