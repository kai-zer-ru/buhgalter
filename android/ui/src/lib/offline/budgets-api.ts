import {
	createBudget as apiCreateBudget,
	updateBudget as apiUpdateBudget,
	deleteBudget as apiDeleteBudget,
	ApiError,
	isTransientHttpError,
	type BudgetItem,
	type BudgetScope
} from '$lib/api/client';
import {
	isConnectionError,
	markServerOffline,
	shouldTryServer
} from '$lib/offline/server-connectivity';
import { shouldUseOfflineQueue } from '$lib/offline/network';
import {
	onBudgetCreated,
	onBudgetDeleted,
	onBudgetUpdated
} from '$lib/offline/ref-cache-mutations';
import {
	enqueueBudgetCreate,
	enqueueBudgetDelete,
	enqueueBudgetUpdate,
	makeLocalKey
} from '$lib/offline/store';
import type { BudgetPayload } from '$lib/offline/types';
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

type BudgetBody = {
	name: string;
	scope: BudgetScope;
	category_id?: string;
	subcategory_id?: string;
	account_id?: string;
	amount: string;
	alert_at_percent?: number;
	is_active?: boolean;
	copy_forward?: boolean;
};

function toPayload(body: BudgetBody, month?: string): BudgetPayload {
	return { ...body, month };
}

function localBudget(id: string, payload: BudgetPayload): BudgetItem {
	const ts = new Date().toISOString();
	const amount = Number.parseFloat(payload.amount) || 0;
	return {
		id,
		name: payload.name,
		scope: payload.scope,
		category_id: payload.category_id,
		subcategory_id: payload.subcategory_id,
		account_id: payload.account_id,
		month: payload.month,
		copy_forward: payload.copy_forward,
		amount,
		amount_display: payload.amount,
		period: 'month',
		alert_at_percent: payload.alert_at_percent ?? 80,
		is_active: payload.is_active ?? true,
		created_at: ts,
		updated_at: ts
	};
}

export async function createBudget(body: BudgetBody, month?: string): Promise<BudgetItem> {
	const payload = toPayload(body, month);
	if (!shouldUseOfflineQueue()) {
		const item = await apiCreateBudget(body, month);
		onBudgetCreated(item, month);
		return item;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiCreateBudget(body, month));
		if (res) {
			onBudgetCreated(res, month);
			scheduleSyncOutbox();
			return res;
		}
	}
	const localKey = makeLocalKey();
	enqueueBudgetCreate(localKey, payload);
	const item = localBudget(localKey, payload);
	onBudgetCreated(item, month);
	return item;
}

export async function updateBudget(
	id: string,
	body: BudgetBody,
	month?: string
): Promise<BudgetItem> {
	const payload = toPayload(body, month);
	if (!shouldUseOfflineQueue()) {
		const item = await apiUpdateBudget(id, body, month);
		onBudgetUpdated(item, month);
		return item;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUpdateBudget(id, body, month));
		if (res) {
			onBudgetUpdated(res, month);
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueBudgetUpdate(id, payload);
	const item = localBudget(id, payload);
	onBudgetUpdated(item, month);
	return item;
}

export async function deleteBudget(id: string, month?: string): Promise<void> {
	if (!shouldUseOfflineQueue()) {
		await apiDeleteBudget(id);
		onBudgetDeleted(id, month);
		return;
	}
	if (isLocalEntityKey(id)) {
		enqueueBudgetDelete(id);
		onBudgetDeleted(id, month);
		return;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiDeleteBudget(id));
		if (res !== null) {
			onBudgetDeleted(id, month);
			scheduleSyncOutbox();
			return;
		}
	}
	enqueueBudgetDelete(id);
	onBudgetDeleted(id, month);
}
