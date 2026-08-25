import {
	createSubscription as apiCreateSubscription,
	deleteSubscription as apiDeleteSubscription,
	updateSubscription as apiUpdateSubscription,
	ApiError,
	isTransientHttpError,
	type Subscription
} from '$lib/api/client';
import {
	isConnectionError,
	markServerOffline,
	shouldTryServer
} from '$lib/offline/server-connectivity';
import { shouldUseOfflineQueue } from '$lib/offline/network';
import {
	onSubscriptionCreated,
	onSubscriptionDeleted,
	onSubscriptionUpdated
} from '$lib/offline/ref-cache-mutations';
import {
	enqueueSubscriptionCreate,
	enqueueSubscriptionDelete,
	enqueueSubscriptionUpdate,
	makeLocalKey
} from '$lib/offline/store';
import type { SubscriptionPayload } from '$lib/offline/types';
import { isLocalEntityKey } from '$lib/offline/types';
import { afterOnlineWrite } from '$lib/offline/after-online-write';
import { OnlineOnlyError, requireOnline } from '$lib/offline/require-online';

function requireOnlineForAttach(): void {
	if (!requireOnline()) throw new OnlineOnlyError();
}

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

function offlinePayload(payload: SubscriptionPayload): SubscriptionPayload {
	const rest = { ...payload };
	delete rest.attach_transaction_id;
	return rest;
}

function localSubscription(id: string, payload: SubscriptionPayload): Subscription {
	const ts = new Date().toISOString();
	const amount = Number.parseFloat(payload.amount) || 0;
	return {
		id,
		name: payload.name,
		description: payload.description ?? null,
		icon: payload.icon ?? null,
		website_url: payload.website_url ?? null,
		amount,
		amount_display: payload.amount,
		account_id: payload.account_id,
		account_name: '',
		subcategory_id: null,
		subcategory_name: null,
		subcategory_icon: null,
		period: payload.period,
		weekday: payload.weekday ?? null,
		day_of_month: payload.day_of_month ?? null,
		start_date: payload.start_date,
		time_local: payload.time_local ?? '08:00',
		next_run_at: payload.upcoming_run_ats?.[0] ?? payload.start_date,
		upcoming_run_ats: payload.upcoming_run_ats ?? [],
		last_run_at: null,
		active: payload.active ?? true,
		created_at: ts,
		updated_at: ts
	};
}

export async function createSubscription(payload: SubscriptionPayload): Promise<Subscription> {
	if (!shouldUseOfflineQueue()) {
		const item = await apiCreateSubscription(payload);
		onSubscriptionCreated(item);
		return item;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiCreateSubscription(payload));
		if (res) {
			onSubscriptionCreated(res);
			afterOnlineWrite();
			return res;
		}
	}
	if (payload.attach_transaction_id) {
		requireOnlineForAttach();
	}
	const localKey = makeLocalKey();
	const queued = offlinePayload(payload);
	enqueueSubscriptionCreate(localKey, queued);
	const item = localSubscription(localKey, queued);
	onSubscriptionCreated(item);
	return item;
}

export async function updateSubscription(
	id: string,
	payload: SubscriptionPayload
): Promise<Subscription> {
	const queued = offlinePayload(payload);
	if (!shouldUseOfflineQueue()) {
		const item = await apiUpdateSubscription(id, queued);
		onSubscriptionUpdated(item);
		return item;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUpdateSubscription(id, queued));
		if (res) {
			onSubscriptionUpdated(res);
			afterOnlineWrite();
			return res;
		}
	}
	enqueueSubscriptionUpdate(id, queued);
	const item = localSubscription(id, queued);
	onSubscriptionUpdated(item);
	return item;
}

export async function deleteSubscription(id: string): Promise<void> {
	if (!shouldUseOfflineQueue()) {
		await apiDeleteSubscription(id);
		onSubscriptionDeleted(id);
		return;
	}
	if (isLocalEntityKey(id)) {
		enqueueSubscriptionDelete(id);
		onSubscriptionDeleted(id);
		return;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiDeleteSubscription(id));
		if (res !== null) {
			onSubscriptionDeleted(id);
			afterOnlineWrite();
			return;
		}
	}
	enqueueSubscriptionDelete(id);
	onSubscriptionDeleted(id);
}
