export {
	accountTransferConfirmStore as accountDeleteConfirmStore,
	confirmAccountTransfer as confirmAccountDelete,
	resolveAccountTransferConfirm as resolveAccountDeleteConfirm,
	setAccountTransferTarget as setAccountDeleteTransferTarget,
	type AccountTransferConfirmOptions as AccountDeleteConfirmOptions,
	type AccountTransferConfirmState as AccountDeleteConfirmState,
	type AccountTransferConfirmResult as AccountDeleteConfirmResult
} from '$lib/accounts/account-transfer-confirm';

export { needsBalanceTransfer as needsDeleteTransfer } from '$lib/transfer-accounts';
