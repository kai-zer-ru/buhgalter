import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import {
	androidAdminNavItems,
	androidHomeNavItem,
	androidMainNavItemsAfterHome,
	androidSettingsNavItems
} from './nav-items';

const routesRoot = join(import.meta.dirname, '../../routes');

function pageFileForPath(pathname: string): string {
	if (pathname === '/') return join(routesRoot, '+page.svelte');
	return join(routesRoot, pathname.replace(/^\//, ''), '+page.svelte');
}

function hrefPath(href: string): string {
	const url = new URL(href, 'https://app.local');
	return url.pathname;
}

function expectRouteExists(pathname: string) {
	const file = pageFileForPath(pathname);
	expect(existsSync(file), `missing route file for ${pathname}: ${file}`).toBe(true);
}

describe('android route coverage', () => {
	it('covers drawer main navigation', () => {
		expectRouteExists('/');
		for (const item of androidMainNavItemsAfterHome()) {
			expectRouteExists(hrefPath(item.href));
		}
	});

	it('covers settings hub navigation', () => {
		expectRouteExists('/settings');
		for (const item of androidSettingsNavItems()) {
			expectRouteExists(hrefPath(item.href));
		}
	});

	it('covers admin hub navigation', () => {
		expectRouteExists('/admin');
		for (const item of androidAdminNavItems()) {
			expectRouteExists(hrefPath(item.href));
		}
	});

	it('covers linked sub-routes from MVP scope', () => {
		expectRouteExists('/debtors/[id]');
		expectRouteExists(hrefPath(androidHomeNavItem().href));
		expectRouteExists('/credits/[id]');
		expectRouteExists('/debts/new');
	});
});
