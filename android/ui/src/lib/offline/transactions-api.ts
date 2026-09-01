import {
	createTransaction as apiCreateTransaction,
	createTransfer as apiCreateTransfer,
	deleteTransaction as apiDeleteTransaction,
	deleteTransfer as apiDeleteTransfer,
	updateTransaction as apiUpdateTransaction,
	updateTransfer as apiUpdateTransfer,
	ApiError,
	isTransientHttpError,
	type Transaction,
	type Transfer
} from '$lib/api/client';
import {
	isConnectionError,
	markServerOffline,
	shouldTryServer
} from '$lib/offline/server-connectivity';
import { shouldUseOfflineQueue } from '$lib/offline/network';
import {
	enqueueTransactionCreate,
	enqueueTransactionDelete,
	enqueueTransactionUpdate,
	enqueueTransferCreate,
	enqueueTransferDelete,
	enqueueTransferUpdate,
	makeLocalKey
} from '$lib/offline/store';
import type { TransactionPayload, TransferPayload } from '$lib/offline/types';
import { isLocalEntityKey } from '$lib/offline/types';
import { afterOnlineWrite } from '$lib/offline/after-online-write';
import {
	ensureMerchantsTagsFromTransaction,
	ensureSubcategoryInCacheFromTransaction
} from '$lib/offline/ref-cache-mutations';

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

function localTransactionFromPayload(id: string, payload: TransactionPayload): Transaction {
	const ts = new Date().toISOString();
	const tags = [
		...(payload.tag_ids ?? []).map((tagId) => ({ id: tagId, name: '' })),
		...(payload.tag_names ?? []).map((name) => ({ id: '', name }))
	];
	return {
		id,
		account_id: payload.account_id,
		type: payload.type,
		kind: 'manual',
		amount: 0,
		amount_display: payload.amount,
		description: payload.description ?? null,
		merchant_id: payload.merchant_id ?? null,
		merchant_name: payload.merchant_name ?? null,
		tags: tags.length ? tags : undefined,
		category_id: payload.category_id ?? null,
		subcategory_id: payload.subcategory_id ?? null,
		transaction_date: payload.transaction_date,
		created_at: ts,
		updated_at: ts
	} as Transaction;
}

function patchTransactionCaches(tx: Transaction): void {
	ensureMerchantsTagsFromTransaction(tx);
	ensureSubcategoryInCacheFromTransaction(tx);
}

export async function createTransaction(payload: TransactionPayload): Promise<Transaction> {
	if (!shouldUseOfflineQueue()) {
		const tx = await apiCreateTransaction(payload);
		patchTransactionCaches(tx);
		return tx;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiCreateTransaction(payload));
		if (res) {
			patchTransactionCaches(res);
			afterOnlineWrite();
			return res;
		}
	}
	const localKey = makeLocalKey();
	enqueueTransactionCreate(localKey, payload);
	const tx = localTransactionFromPayload(localKey, payload);
	ensureMerchantsTagsFromTransaction(tx, {
		merchantName: payload.merchant_name,
		tagNames: payload.tag_names
	});
	return tx;
}

export async function updateTransaction(
	id: string,
	payload: TransactionPayload
): Promise<Transaction> {
	if (!shouldUseOfflineQueue()) {
		const tx = await apiUpdateTransaction(id, payload);
		patchTransactionCaches(tx);
		return tx;
	}
	if (isLocalEntityKey(id)) {
		enqueueTransactionUpdate(id, payload);
		const tx = localTransactionFromPayload(id, payload);
		ensureMerchantsTagsFromTransaction(tx, {
			merchantName: payload.merchant_name,
			tagNames: payload.tag_names
		});
		return tx;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUpdateTransaction(id, payload));
		if (res) {
			patchTransactionCaches(res);
			afterOnlineWrite();
			return res;
		}
	}
	enqueueTransactionUpdate(id, payload);
	const tx = localTransactionFromPayload(id, payload);
	ensureMerchantsTagsFromTransaction(tx, {
		merchantName: payload.merchant_name,
		tagNames: payload.tag_names
	});
	return tx;
}

export async function deleteTransaction(id: string): Promise<void> {
	if (!shouldUseOfflineQueue()) {
		return apiDeleteTransaction(id);
	}
	if (isLocalEntityKey(id)) {
		enqueueTransactionDelete(id);
		return;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiDeleteTransaction(id));
		if (res !== null) {
			afterOnlineWrite();
			return;
		}
	}
	enqueueTransactionDelete(id);
}

function stubTransfer(groupId: string, payload: TransferPayload): Transfer {
	return {
		group_id: groupId,
		from_account_id: payload.from_account_id,
		to_account_id: payload.to_account_id,
		amount: 0,
		amount_display: payload.amount,
		commission: 0,
		commission_display: payload.commission ?? '0.00',
		description: payload.description ?? null,
		transaction_date: payload.transaction_date,
		kind: 'manual',
		legs: []
	};
}

export async function createTransfer(payload: TransferPayload): Promise<Transfer> {
	if (!shouldUseOfflineQueue()) {
		return apiCreateTransfer(payload);
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiCreateTransfer(payload));
		if (res) {
			afterOnlineWrite();
			return res;
		}
	}
	const localKey = makeLocalKey();
	enqueueTransferCreate(localKey, payload);
	return stubTransfer(localKey, payload);
}

export async function updateTransfer(groupId: string, payload: TransferPayload): Promise<Transfer> {
	if (!shouldUseOfflineQueue()) {
		return apiUpdateTransfer(groupId, payload);
	}
	if (isLocalEntityKey(groupId)) {
		enqueueTransferUpdate(groupId, payload);
		return stubTransfer(groupId, payload);
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUpdateTransfer(groupId, payload));
		if (res) {
			afterOnlineWrite();
			return res;
		}
	}
	enqueueTransferUpdate(groupId, payload);
	return stubTransfer(groupId, payload);
}

export async function deleteTransfer(groupId: string): Promise<void> {
	if (!shouldUseOfflineQueue()) {
		return apiDeleteTransfer(groupId);
	}
	if (isLocalEntityKey(groupId)) {
		enqueueTransferDelete(groupId);
		return;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiDeleteTransfer(groupId));
		if (res !== null) {
			afterOnlineWrite();
			return;
		}
	}
	enqueueTransferDelete(groupId);
}
