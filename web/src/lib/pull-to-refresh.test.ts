import { afterEach, describe, expect, it, vi } from 'vitest';

const invalidateRefCachePrefix = vi.fn();

vi.mock('$lib/ref-cache', () => ({
	invalidateRefCachePrefix: (...args: unknown[]) => invalidateRefCachePrefix(...args)
}));

import {
	apiPathsForPtrRoute,
	attachNonPassiveTouchMove,
	isPullToRefreshActive,
	isPullToRefreshRoute,
	isPtrBlockedByModal,
	normalizePtrPathname,
	PTR_VIEWPORT_MAX_PX,
	revalidateCurrentPath
} from './pull-to-refresh';

describe('isPullToRefreshActive', () => {
	it('is always on for Capacitor', () => {
		expect(
			isPullToRefreshActive({
				isCapacitor: true,
				coarsePointer: false,
				viewportWidth: 1400
			})
		).toBe(true);
	});

	it('is on for coarse pointer on wide viewport', () => {
		expect(
			isPullToRefreshActive({
				isCapacitor: false,
				coarsePointer: true,
				viewportWidth: 1400
			})
		).toBe(true);
	});

	it('is on for narrow viewport without coarse pointer', () => {
		expect(
			isPullToRefreshActive({
				isCapacitor: false,
				coarsePointer: false,
				viewportWidth: PTR_VIEWPORT_MAX_PX
			})
		).toBe(true);
		expect(
			isPullToRefreshActive({
				isCapacitor: false,
				coarsePointer: false,
				viewportWidth: PTR_VIEWPORT_MAX_PX - 1
			})
		).toBe(true);
	});

	it('is off for desktop fine-pointer wide viewport', () => {
		expect(
			isPullToRefreshActive({
				isCapacitor: false,
				coarsePointer: false,
				viewportWidth: PTR_VIEWPORT_MAX_PX + 1
			})
		).toBe(false);
	});
});

describe('isPullToRefreshRoute / apiPathsForPtrRoute', () => {
	it('accepts surfaces.md PTR routes', () => {
		for (const path of [
			'/',
			'/transactions',
			'/accounts',
			'/accounts/abc-123',
			'/budget',
			'/credits',
			'/debts'
		]) {
			expect(isPullToRefreshRoute(path)).toBe(true);
			expect(apiPathsForPtrRoute(path).length).toBeGreaterThan(0);
		}
	});

	it('rejects create and nested action paths', () => {
		for (const path of [
			'/accounts/new',
			'/credits/new',
			'/credits/xyz',
			'/debts/new',
			'/transactions/new',
			'/stats',
			'/settings/profile'
		]) {
			expect(isPullToRefreshRoute(path)).toBe(false);
			expect(apiPathsForPtrRoute(path)).toEqual([]);
		}
	});

	it('normalizes query and trailing slash', () => {
		expect(normalizePtrPathname('/transactions/?q=1')).toBe('/transactions');
		expect(apiPathsForPtrRoute('/accounts/abc/?x=1')).toContain('/api/v1/accounts/abc');
	});
});

describe('revalidateCurrentPath', () => {
	afterEach(() => {
		invalidateRefCachePrefix.mockClear();
	});

	it('invalidates API path prefixes for the current route', () => {
		revalidateCurrentPath('/transactions');
		expect(invalidateRefCachePrefix).toHaveBeenCalledWith('/api/v1/transactions');
		expect(invalidateRefCachePrefix).toHaveBeenCalledWith('/api/v1/ui/meta');
	});

	it('no-ops for non-PTR routes', () => {
		revalidateCurrentPath('/stats');
		expect(invalidateRefCachePrefix).not.toHaveBeenCalled();
	});

	it('maps home to dashboard aggregates', () => {
		revalidateCurrentPath('/');
		expect(invalidateRefCachePrefix).toHaveBeenCalledWith('/api/v1/dashboard');
		expect(invalidateRefCachePrefix).toHaveBeenCalledWith('/api/v1/transactions');
		expect(invalidateRefCachePrefix).toHaveBeenCalledWith('/api/v1/budgets/summary');
		expect(invalidateRefCachePrefix).toHaveBeenCalledWith('/api/v1/transaction-templates');
	});
});

describe('isPtrBlockedByModal', () => {
	it('detects open aria-modal dialog', () => {
		const root = {
			querySelector: (sel: string) => (sel === '[role="dialog"][aria-modal="true"]' ? {} : null)
		};
		expect(isPtrBlockedByModal(root as unknown as ParentNode)).toBe(true);
	});

	it('is false when no dialog', () => {
		const root = { querySelector: () => null };
		expect(isPtrBlockedByModal(root as unknown as ParentNode)).toBe(false);
	});

	it('blocks when Android drawer is open', () => {
		const root = {
			querySelector: (sel: string) => (sel === '.android-shell-layout.drawer-open' ? {} : null)
		};
		expect(isPtrBlockedByModal(root as unknown as ParentNode)).toBe(true);
	});
});

describe('attachNonPassiveTouchMove', () => {
	it('registers touchmove with passive: false and disposes cleanly', () => {
		const handler = vi.fn();
		const add = vi.fn();
		const remove = vi.fn();
		const node = {
			addEventListener: add,
			removeEventListener: remove
		} as unknown as HTMLElement;

		const dispose = attachNonPassiveTouchMove(node, handler);
		expect(add).toHaveBeenCalledWith('touchmove', handler, { passive: false });

		dispose();
		expect(remove).toHaveBeenCalledWith('touchmove', handler);
	});
});
