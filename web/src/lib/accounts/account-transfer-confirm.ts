import { get, writable } from 'svelte/store';

export type AccountTransferConfirmOptions = {
	message: string;
	needsTransfer: boolean;
	balanceMessageBefore: string;
	balanceMessageAfter: string;
	balanceDisplay: string;
	transferLabel: string;
	noTargetsMessage: string;
	transferOptions: { value: string; label: string }[];
	initialTransferAccountId?: string;
	confirmLabel?: string;
	cancelLabel?: string;
	danger?: boolean;
};

export type AccountTransferConfirmState = {
	open: boolean;
	options: AccountTransferConfirmOptions;
	transferToAccountId: string;
};

export type AccountTransferConfirmResult = {
	ok: boolean;
	transferToAccountId?: string;
};

const closed: AccountTransferConfirmState = {
	open: false,
	options: {
		message: '',
		needsTransfer: false,
		balanceMessageBefore: '',
		balanceMessageAfter: '',
		balanceDisplay: '',
		transferLabel: '',
		noTargetsMessage: '',
		transferOptions: []
	},
	transferToAccountId: ''
};

export const accountTransferConfirmStore = writable<AccountTransferConfirmState>(closed);

/** @deprecated Use accountTransferConfirmStore */
export const accountDeleteConfirmStore = accountTransferConfirmStore;

let pendingResolve: ((value: AccountTransferConfirmResult) => void) | null = null;

export function confirmAccountTransfer(
	options: AccountTransferConfirmOptions
): Promise<AccountTransferConfirmResult> {
	const initialTransfer =
		options.initialTransferAccountId ?? options.transferOptions[0]?.value ?? '';
	return new Promise((resolve) => {
		if (pendingResolve) {
			pendingResolve({ ok: false });
		}
		pendingResolve = resolve;
		accountTransferConfirmStore.set({
			open: true,
			options,
			transferToAccountId: initialTransfer
		});
	});
}

/** @deprecated Use confirmAccountTransfer */
export const confirmAccountDelete = confirmAccountTransfer;

export function resolveAccountTransferConfirm(
	ok: boolean,
	transferToAccountId?: string
): AccountTransferConfirmResult {
	const current = get(accountTransferConfirmStore);
	const target = ok ? (transferToAccountId ?? current.transferToAccountId).trim() : '';
	const result: AccountTransferConfirmResult = ok
		? { ok: true, transferToAccountId: target || undefined }
		: { ok: false };
	accountTransferConfirmStore.set(closed);
	pendingResolve?.(result);
	pendingResolve = null;
	return result;
}

/** @deprecated Use resolveAccountTransferConfirm */
export const resolveAccountDeleteConfirm = resolveAccountTransferConfirm;

export function setAccountTransferTarget(transferToAccountId: string) {
	accountTransferConfirmStore.update((state) => ({ ...state, transferToAccountId }));
}

/** @deprecated Use setAccountTransferTarget */
export const setAccountDeleteTransferTarget = setAccountTransferTarget;
