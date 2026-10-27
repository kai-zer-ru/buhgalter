import { describe, expect, it, vi } from 'vitest';
import {
	deepLinkHistoryAction,
	followDeepLinkHistory,
	parseAppDeepLink
} from '$lib/android/deep-link';

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
		expect(parseAppDeepLink('ru.kai_zer.buhgalter://invite/ABC-DEF-GHJ')).toBe(
			'/invite/ABC-DEF-GHJ'
		);
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

describe('deepLinkHistoryAction', () => {
	it('skips when path and query already match', () => {
		expect(
			deepLinkHistoryAction('/transactions/new?type=expense', '/transactions/new?type=expense')
		).toBe('skip');
		expect(deepLinkHistoryAction('/', '/')).toBe('skip');
		expect(deepLinkHistoryAction('/accounts', '/accounts#details')).toBe('skip');
		expect(deepLinkHistoryAction('/transactions/new', '/transactions/new?')).toBe('skip');
	});

	it('pushes when the path or query differs', () => {
		expect(deepLinkHistoryAction('/', '/settings/bank-notifications/drafts')).toBe('push');
		expect(
			deepLinkHistoryAction('/transactions/new?type=expense', '/transactions/new?type=income')
		).toBe('push');
		expect(deepLinkHistoryAction('/accounts', '/transactions')).toBe('push');
	});

	it('never chooses replace', () => {
		const actions = [
			deepLinkHistoryAction('/', '/'),
			deepLinkHistoryAction('/', '/accounts'),
			deepLinkHistoryAction('/transactions/new?type=expense', '/transactions/new?type=income'),
			deepLinkHistoryAction('/transactions/new?type=expense', '/transactions/new?type=expense')
		];
		for (const action of actions) {
			const chosen: string = action;
			expect(['skip', 'push']).toContain(chosen);
			expect(chosen).not.toBe('replace');
		}
	});
});

describe('followDeepLinkHistory', () => {
	it('pushes the given route with replaceState false', () => {
		const navigate = vi.fn();
		const route = '/settings/bank-notifications/drafts';
		followDeepLinkHistory('/', route, navigate);
		expect(navigate).toHaveBeenCalledTimes(1);
		expect(navigate).toHaveBeenCalledWith(route, { replaceState: false });
	});

	it('pushes when only the query differs', () => {
		const navigate = vi.fn();
		const route = '/transactions/new?type=income';
		followDeepLinkHistory('/transactions/new?type=expense', route, navigate);
		expect(navigate).toHaveBeenCalledWith(route, { replaceState: false });
	});

	it('does not navigate when path and query match', () => {
		const navigate = vi.fn();
		followDeepLinkHistory(
			'/transactions/new?type=expense',
			'/transactions/new?type=expense',
			navigate
		);
		expect(navigate).not.toHaveBeenCalled();
	});
});
