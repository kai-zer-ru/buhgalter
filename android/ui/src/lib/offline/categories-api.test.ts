import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Category } from '$lib/api/client';
import { createCategory, deleteCategory, updateCategory } from '$lib/offline/categories-api';
import {
	categoriesRefPath,
	readRefCache,
	resetRefCacheForTests,
	writeRefCache
} from '$lib/offline/ref-cache';
import { getOutboxEntries, resetOutboxForTests } from '$lib/offline/store';
import { isLocalEntityKey } from '$lib/offline/types';
import * as client from '$lib/api/client';
import * as connectivity from '$lib/offline/server-connectivity';
import * as network from '$lib/offline/network';
import * as sync from '$lib/offline/sync';

vi.mock('$lib/api/client', () => ({
	createCategory: vi.fn(),
	updateCategory: vi.fn(),
	deleteCategory: vi.fn(),
	ApiError: class ApiError extends Error {
		constructor(
			public code: string,
			message: string,
			public status: number
		) {
			super(message);
		}
	},
	isTransientHttpError: (status: number) => status === 503
}));

vi.mock('$lib/offline/server-connectivity', async (importOriginal) => {
	const actual = await importOriginal<typeof connectivity>();
	return {
		...actual,
		shouldTryServer: vi.fn(),
		markServerOffline: vi.fn(),
		isConnectionError: vi.fn()
	};
});

vi.mock('$lib/offline/network', () => ({
	shouldUseOfflineQueue: vi.fn()
}));

vi.mock('$lib/offline/sync', async (importOriginal) => {
	const actual = await importOriginal<typeof sync>();
	return {
		...actual,
		scheduleSyncOutbox: vi.fn()
	};
});

vi.mock('$lib/offline/merge', () => ({
	refreshMergeMeta: vi.fn().mockResolvedValue(undefined)
}));

const categoryPayload = {
	name: 'Еда',
	type: 'expense' as const,
	icon: 'food'
};

const serverCategory: Category = {
	id: 'srv-c1',
	...categoryPayload,
	sort_order: 0,
	is_primary: false,
	is_system: false,
	subcategory_count: 0,
	created_at: '2026-07-08T10:00:00Z'
};

beforeEach(() => {
	resetOutboxForTests();
	resetRefCacheForTests();
	vi.clearAllMocks();
	vi.mocked(network.shouldUseOfflineQueue).mockReturnValue(true);
	vi.mocked(connectivity.shouldTryServer).mockResolvedValue(false);
	vi.mocked(connectivity.isConnectionError).mockReturnValue(true);
	writeRefCache(categoriesRefPath('expense'), []);
});

describe('categories-api offline', () => {
	it('createCategory enqueues local entry and patches cache', async () => {
		const category = await createCategory(categoryPayload);

		expect(isLocalEntityKey(category.id)).toBe(true);
		expect(getOutboxEntries()).toHaveLength(1);
		expect(getOutboxEntries()[0].kind).toBe('category');
		expect(readRefCache<Category[]>(categoriesRefPath('expense'))?.[0]?.id).toBe(category.id);
	});

	it('updateCategory merges type for local id', async () => {
		const created = await createCategory(categoryPayload);
		const updated = await updateCategory(created.id, {
			name: 'Транспорт',
			icon: 'car',
			type: 'expense'
		});

		expect(updated.name).toBe('Транспорт');
		expect(getOutboxEntries()).toHaveLength(1);
		expect(getOutboxEntries()[0].op).toBe('create');
		expect((getOutboxEntries()[0].payload as { name: string }).name).toBe('Транспорт');
	});

	it('deleteCategory removes local entry and cache row', async () => {
		const created = await createCategory(categoryPayload);
		await deleteCategory(created.id, 'expense');

		expect(getOutboxEntries()).toHaveLength(0);
		expect(readRefCache<Category[]>(categoriesRefPath('expense'))).toEqual([]);
	});
});

describe('categories-api online', () => {
	it('createCategory calls API and patches cache when server responds', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);
		vi.mocked(client.createCategory).mockResolvedValue(serverCategory);

		const category = await createCategory(categoryPayload);

		expect(category.id).toBe('srv-c1');
		expect(client.createCategory).toHaveBeenCalledWith(categoryPayload);
		expect(readRefCache<Category[]>(categoriesRefPath('expense'))?.[0]?.id).toBe('srv-c1');
		expect(sync.scheduleSyncOutbox).toHaveBeenCalled();
	});

	it('falls back to outbox when API is unreachable', async () => {
		vi.mocked(connectivity.shouldTryServer).mockResolvedValue(true);
		vi.mocked(client.createCategory).mockRejectedValue(new TypeError('Failed to fetch'));

		const category = await createCategory(categoryPayload);

		expect(isLocalEntityKey(category.id)).toBe(true);
		expect(getOutboxEntries()).toHaveLength(1);
		expect(connectivity.markServerOffline).toHaveBeenCalled();
	});
});
