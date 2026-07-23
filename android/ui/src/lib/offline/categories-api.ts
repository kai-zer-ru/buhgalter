import {
	createCategory as apiCreateCategory,
	deleteCategory as apiDeleteCategory,
	updateCategory as apiUpdateCategory,
	ApiError,
	isTransientHttpError,
	type Category
} from '$lib/api/client';
import {
	isConnectionError,
	markServerOffline,
	shouldTryServer
} from '$lib/offline/server-connectivity';
import { shouldUseOfflineQueue } from '$lib/offline/network';
import {
	onCategoryCreated,
	onCategoryDeleted,
	onCategoryUpdated
} from '$lib/offline/ref-cache-mutations';
import { refreshMergeMeta } from '$lib/offline/merge';
import {
	enqueueCategoryCreate,
	enqueueCategoryDelete,
	enqueueCategoryUpdate,
	makeLocalKey
} from '$lib/offline/store';
import type { CategoryPayload, CategoryUpdatePayload } from '$lib/offline/types';
import { isLocalEntityKey } from '$lib/offline/types';
import { scheduleSyncOutbox } from '$lib/offline/sync';

function isOfflineError(err: unknown): boolean {
	return isConnectionError(err) || (err instanceof ApiError && isTransientHttpError(err.status));
}

async function tryOnline<T>(fn: () => Promise<T>): Promise<T | null> {
	try {
		return await fn();
	} catch (err) {
		if (isOfflineError(err)) {
			markServerOffline();
			return null;
		}
		throw err;
	}
}

function localCategory(id: string, payload: CategoryPayload): Category {
	const ts = new Date().toISOString();
	return {
		id,
		name: payload.name,
		type: payload.type,
		icon: payload.icon,
		sort_order: payload.sort_order ?? 0,
		is_primary: false,
		is_system: false,
		subcategory_count: 0,
		created_at: ts
	};
}

export async function createCategory(payload: CategoryPayload): Promise<Category> {
	if (!shouldUseOfflineQueue()) {
		const category = await apiCreateCategory(payload);
		onCategoryCreated(category);
		void refreshMergeMeta();
		return category;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiCreateCategory(payload));
		if (res) {
			onCategoryCreated(res);
			void refreshMergeMeta();
			scheduleSyncOutbox();
			return res;
		}
	}
	const localKey = makeLocalKey();
	enqueueCategoryCreate(localKey, payload);
	const category = localCategory(localKey, payload);
	onCategoryCreated(category);
	void refreshMergeMeta();
	return category;
}

export async function updateCategory(
	id: string,
	payload: CategoryUpdatePayload & { type?: 'income' | 'expense' }
): Promise<Category> {
	if (!shouldUseOfflineQueue()) {
		const category = await apiUpdateCategory(id, payload);
		onCategoryUpdated(category);
		void refreshMergeMeta();
		return category;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUpdateCategory(id, payload));
		if (res) {
			onCategoryUpdated(res);
			void refreshMergeMeta();
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueCategoryUpdate(id, payload);
	const category: Category = {
		id,
		name: payload.name,
		type: payload.type ?? 'expense',
		icon: payload.icon,
		sort_order: payload.sort_order ?? 0,
		is_primary: false,
		is_system: false,
		subcategory_count: 0,
		created_at: new Date().toISOString()
	};
	onCategoryUpdated(category);
	void refreshMergeMeta();
	return category;
}

export async function deleteCategory(id: string, type: 'income' | 'expense'): Promise<void> {
	if (!shouldUseOfflineQueue()) {
		await apiDeleteCategory(id);
		onCategoryDeleted(id, type);
		void refreshMergeMeta();
		return;
	}
	if (isLocalEntityKey(id)) {
		enqueueCategoryDelete(id);
		onCategoryDeleted(id, type);
		void refreshMergeMeta();
		return;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiDeleteCategory(id));
		if (res !== null) {
			onCategoryDeleted(id, type);
			void refreshMergeMeta();
			scheduleSyncOutbox();
			return;
		}
	}
	enqueueCategoryDelete(id);
	onCategoryDeleted(id, type);
	void refreshMergeMeta();
}
