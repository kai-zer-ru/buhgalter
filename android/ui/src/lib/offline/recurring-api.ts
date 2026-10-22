import {
	createRecurringOperation as apiCreateRecurring,
	deleteRecurringOperation as apiDeleteRecurring,
	updateRecurringOperation as apiUpdateRecurring,
	ApiError,
	isTransientHttpError,
	type RecurringOperation
} from '$lib/api/client';
import {
	isConnectionError,
	markServerOffline,
	shouldTryServer
} from '$lib/offline/server-connectivity';
import { shouldUseOfflineQueue } from '$lib/offline/network';
import {
	onRecurringCreated,
	onRecurringDeleted,
	onRecurringUpdated
} from '$lib/offline/ref-cache-mutations';
import {
	enqueueRecurringCreate,
	enqueueRecurringDelete,
	enqueueRecurringUpdate,
	makeLocalKey
} from '$lib/offline/store';
import type { RecurringPayload } from '$lib/offline/types';
import { isLocalEntityKey } from '$lib/offline/types';
import { afterOnlineWrite } from '$lib/offline/after-online-write';

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

function localRecurring(id: string, payload: RecurringPayload): RecurringOperation {
	const ts = new Date().toISOString();
	const amount = Number.parseFloat(payload.amount) || 0;
	return {
		id,
		type: payload.type,
		amount,
		amount_display: payload.amount,
		description: payload.description ?? null,
		account_id: payload.account_id,
		account_name: '',
		category_id: payload.category_id,
		category_name: '',
		subcategory_id: payload.subcategory_id ?? null,
		subcategory_name: null,
		period: payload.period,
		weekday: payload.weekday ?? null,
		day_of_month: payload.day_of_month ?? null,
		start_date: payload.start_date,
		time_local: payload.time_local ?? '08:00',
		next_run_at: payload.start_date,
		last_run_at: null,
		active: payload.active ?? true,
		created_at: ts,
		updated_at: ts
	};
}

export async function createRecurringOperation(
	payload: RecurringPayload
): Promise<RecurringOperation> {
	if (!shouldUseOfflineQueue()) {
		const item = await apiCreateRecurring(payload);
		onRecurringCreated(item);
		return item;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiCreateRecurring(payload));
		if (res) {
			onRecurringCreated(res);
			afterOnlineWrite();
			return res;
		}
	}
	const localKey = makeLocalKey();
	enqueueRecurringCreate(localKey, payload);
	const item = localRecurring(localKey, payload);
	onRecurringCreated(item);
	return item;
}

export async function updateRecurringOperation(
	id: string,
	payload: RecurringPayload
): Promise<RecurringOperation> {
	if (!shouldUseOfflineQueue()) {
		const item = await apiUpdateRecurring(id, payload);
		onRecurringUpdated(item);
		return item;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUpdateRecurring(id, payload));
		if (res) {
			onRecurringUpdated(res);
			afterOnlineWrite();
			return res;
		}
	}
	enqueueRecurringUpdate(id, payload);
	const item = localRecurring(id, payload);
	onRecurringUpdated(item);
	return item;
}

export async function deleteRecurringOperation(id: string): Promise<void> {
	if (!shouldUseOfflineQueue()) {
		await apiDeleteRecurring(id);
		onRecurringDeleted(id);
		return;
	}
	if (isLocalEntityKey(id)) {
		enqueueRecurringDelete(id);
		onRecurringDeleted(id);
		return;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiDeleteRecurring(id));
		if (res !== null) {
			onRecurringDeleted(id);
			afterOnlineWrite();
			return;
		}
	}
	enqueueRecurringDelete(id);
	onRecurringDeleted(id);
}
