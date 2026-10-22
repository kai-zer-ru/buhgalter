import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import en from '$lib/i18n/en.json';
import ru from '$lib/i18n/ru.json';

const BANNER_KEYS = [
	'settings.notifications.web_only_banner',
	'settings.notifications.web_only_hint'
] as const;

describe('settings notifications web-only banner', () => {
	it('has ru and en i18n keys', () => {
		for (const key of BANNER_KEYS) {
			expect(ru[key], `missing ru key ${key}`).toBeTruthy();
			expect(en[key], `missing en key ${key}`).toBeTruthy();
		}
	});

	it('page shows banner keys and does not import NotificationsTab', () => {
		const page = readFileSync(join(import.meta.dirname, '+page.svelte'), 'utf8');
		expect(page).not.toMatch(/NotificationsTab/);
		expect(page).toContain('settings.notifications.web_only_banner');
		expect(page).toContain('settings.notifications.web_only_hint');
		expect(page).toContain('EmptyStateCard');
	});
});
