import {
	createAccount as apiCreateAccount,
	updateAccount as apiUpdateAccount,
	archiveAccount as apiArchiveAccount,
	unarchiveAccount as apiUnarchiveAccount,
	ApiError,
	isTransientHttpError,
	type Account
} from '$lib/api/client';
import {
	isConnectionError,
	markServerOffline,
	shouldTryServer
} from '$lib/offline/server-connectivity';
import { shouldUseOfflineQueue } from '$lib/offline/network';
import {
	onAccountArchived,
	onAccountCreated,
	onAccountUnarchived,
	onAccountUpdated
} from '$lib/offline/ref-cache-mutations';
import { refreshMergeMeta } from '$lib/offline/merge';
import {
	enqueueAccountArchive,
	enqueueAccountCreate,
	enqueueAccountUnarchive,
	enqueueAccountUpdate,
	makeLocalKey
} from '$lib/offline/store';
import type { AccountCreatePayload, AccountUpdatePayload } from '$lib/offline/types';
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

function localAccount(id: string, payload: AccountCreatePayload): Account {
	const ts = new Date().toISOString();
	const initial = Number.parseFloat(payload.initial_balance) || 0;
	return {
		id,
		name: payload.name,
		type: payload.type,
		bank_id: payload.bank_id ?? null,
		initial_balance: initial,
		balance: initial,
		balance_display: payload.initial_balance,
		credit_limit: payload.credit_limit ? Number.parseFloat(payload.credit_limit) : null,
		payment_account_id: payload.payment_account_id ?? null,
		status: 'active',
		is_primary: false,
		created_at: ts,
		updated_at: ts
	};
}

export async function createAccount(payload: AccountCreatePayload): Promise<Account> {
	if (!shouldUseOfflineQueue()) {
		const account = await apiCreateAccount(payload);
		onAccountCreated(account);
		void refreshMergeMeta();
		return account;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiCreateAccount(payload));
		if (res) {
			onAccountCreated(res);
			void refreshMergeMeta();
			scheduleSyncOutbox();
			return res;
		}
	}
	const localKey = makeLocalKey();
	enqueueAccountCreate(localKey, payload);
	const account = localAccount(localKey, payload);
	onAccountCreated(account);
	void refreshMergeMeta();
	return account;
}

export async function updateAccount(id: string, payload: AccountUpdatePayload): Promise<Account> {
	if (!shouldUseOfflineQueue()) {
		const account = await apiUpdateAccount(id, payload);
		onAccountUpdated(account);
		void refreshMergeMeta();
		return account;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUpdateAccount(id, payload));
		if (res) {
			onAccountUpdated(res);
			void refreshMergeMeta();
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueAccountUpdate(id, payload);
	const account: Account = {
		id,
		name: payload.name,
		type: 'cash',
		bank_id: payload.bank_id ?? null,
		initial_balance: payload.initial_balance ? Number.parseFloat(payload.initial_balance) : 0,
		balance: payload.initial_balance ? Number.parseFloat(payload.initial_balance) : 0,
		balance_display: payload.initial_balance ?? '0',
		credit_limit: payload.credit_limit ? Number.parseFloat(payload.credit_limit) : null,
		payment_account_id: payload.payment_account_id ?? null,
		auto_topup_enabled: payload.auto_topup_enabled,
		status: 'active',
		is_primary: false,
		created_at: new Date().toISOString(),
		updated_at: new Date().toISOString()
	};
	onAccountUpdated(account);
	void refreshMergeMeta();
	return account;
}

export async function archiveAccount(id: string, transferToAccountId?: string): Promise<Account> {
	if (!shouldUseOfflineQueue()) {
		const account = await apiArchiveAccount(id, transferToAccountId);
		onAccountArchived(account);
		void refreshMergeMeta();
		return account;
	}
	if (isLocalEntityKey(id)) {
		enqueueAccountArchive(id, transferToAccountId);
		const stub = localAccount(id, {
			name: '',
			type: 'cash',
			initial_balance: '0'
		});
		onAccountArchived({ ...stub, status: 'archived' });
		void refreshMergeMeta();
		return { ...stub, status: 'archived' };
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiArchiveAccount(id, transferToAccountId));
		if (res) {
			onAccountArchived(res);
			void refreshMergeMeta();
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueAccountArchive(id, transferToAccountId);
	const stub: Account = {
		id,
		name: '',
		type: 'cash',
		bank_id: null,
		initial_balance: 0,
		balance: 0,
		balance_display: '0',
		status: 'archived',
		is_primary: false,
		created_at: new Date().toISOString(),
		updated_at: new Date().toISOString()
	};
	onAccountArchived(stub);
	void refreshMergeMeta();
	return stub;
}

export async function unarchiveAccount(id: string): Promise<Account> {
	if (!shouldUseOfflineQueue()) {
		const account = await apiUnarchiveAccount(id);
		onAccountUnarchived(account);
		void refreshMergeMeta();
		return account;
	}
	if (await shouldTryServer()) {
		const res = await tryOnline(() => apiUnarchiveAccount(id));
		if (res) {
			onAccountUnarchived(res);
			void refreshMergeMeta();
			scheduleSyncOutbox();
			return res;
		}
	}
	enqueueAccountUnarchive(id);
	const stub: Account = {
		id,
		name: '',
		type: 'cash',
		bank_id: null,
		initial_balance: 0,
		balance: 0,
		balance_display: '0',
		status: 'active',
		is_primary: false,
		created_at: new Date().toISOString(),
		updated_at: new Date().toISOString()
	};
	onAccountUnarchived(stub);
	void refreshMergeMeta();
	return stub;
}
